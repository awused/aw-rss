import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {
  firstValueFrom,
  forkJoin,
  Observable,
  of,
  ReplaySubject,
  Subject
} from 'rxjs';
import {
  catchError,
  filter,
  map,
  mergeMap,
  share,
  take,
} from 'rxjs/operators';

import {Data,
        FilteredData,
        Updates} from '../models/data';
import {Category,
        Feed,
        Item} from '../models/entities';
import {Filters} from '../models/filter';

import {ErrorService} from './error.service';
import {LoadingService} from './loading.service';
import {RefreshService} from './refresh.service';

const READ_ITEMS_PAGE_SIZE = 100;

interface CurrentState {
  timestamp: number;
  feeds?: Feed[];
  items?: Item[];
  categories?: Category[];
  newestTimestamps?: {[x: number]: string};
}

interface ServerUpdates {
  timestamp: number;
  feeds?: Feed[];
  items?: Item[];
  categories?: Category[];
  mustRefresh?: boolean;
}

interface GetItemsRequest {
  // At most one of categoryId and feedIds can be set
  categoryId?: number;
  feedIds?: number[];

  // includeFeeds?: boolean;

  // Exactly one of unread, readAfter, and (readBefore, readBeforeCount) must be set
  // Unread can only be set with feedIds, since it is only used for backfilling.
  unread?: boolean;
  readBefore?: Date;
  readBeforeCount?: number;
  readAfter?: Date;
}

interface GetItemsResponse {
  items: Item[];
}


// Data about what is present in the cache for this feed
class FeedMetadata {
  constructor(
      public feed: Feed,
      // True if we have all unread items, at least up to DataService.timestamp
      public hasUnread: boolean,
      // True only when we know we have all read items from the database
      public allRead: boolean,
      // We have all read items after this timestamp for this feed
      // If readAfter <= feed.createTimestamp, then we have all read items
      public readAfter?: Date) {}

  public setReadAfter(d: Date) {
    if (this.readAfter && d >= this.readAfter) {
      return;
    }

    this.readAfter = d;
  }

  public hasReadAfter(d: Date|undefined): boolean {
    return !d || this.allRead || (this.readAfter && this.readAfter >= d) || false;
  }
}

// Data about what is present in the cache for this category
class CategoryMetadata {
  constructor(
      public category: Category,
      // True only when we know we have all read items from the database
      public allRead: boolean,
      public readAfter?: Date) {}

  public setReadAfter(d: Date) {
    if (this.readAfter && d >= this.readAfter) {
      return;
    }

    this.readAfter = d;
  }

  public hasReadAfter(d: Date): boolean {
    return this.allRead || (this.readAfter && this.readAfter >= d) || false;
  }
}


@Injectable({
  providedIn: 'root'
})
export class DataService {
  private readonly data$: ReplaySubject<Data> = new ReplaySubject<Data>(1);
  private readonly updates$: Subject<Updates> = new Subject<Updates>();
  private readonly feedUpdates$: Observable<void>;
  private readonly categoryUpdates$: Observable<void>;

  private timestamp = -1;
  private data: Data = new Data();
  // If the initial fetch failed, then hitting refresh will fetch the initial data instead.
  private retryInitial = false;
  private hasAllFeeds = false;
  // private hasAllCategories = false;
  private feedMetadata: Map<number, FeedMetadata> = new Map();
  private categoryMetadata: Map<number, CategoryMetadata> = new Map();
  // For now it's enough to just do this at initial load
  private initialNewestTimestamps: {[x: number]: string} = {};
  private allRead: boolean = false;
  private readAfter?: Date;
  private lastClean: Date = new Date();

  constructor(
      private readonly http: HttpClient,
      private readonly errorService: ErrorService,
      private readonly refreshService: RefreshService,
      private readonly loadingService: LoadingService) {
    this.getInitialState();
    // The data service will never be destroyed, so never unsubscribe
    this.refreshService.startedObservable()
        .subscribe(() => this.refreshState());
    this.feedUpdates$ =
        this.updates$
            .pipe(
                filter((u: Updates) => u.feeds.length > 0),
                map(() => {}),
                share());
    this.categoryUpdates$ =
        this.updates$
            .pipe(
                filter((u: Updates) => u.categories.length > 0),
                map(() => {}),
                share());
  }

  public dataForFilters(f: Filters): Observable<FilteredData> {
    if (f.doNotFetch) {
      return this.filteredDataForFilters(f);
    }

    const fetches = [];

    if (!f.validOnly &&
        !f.excludeFeeds &&
        f.feedId === undefined &&
        !this.hasAllFeeds) {
      fetches.push(this.fetchDisabledFeeds());
    }

    if (!f.unreadOnly) {
      fetches.push(
          this.waitForInitialFetch()
              .pipe(
                  mergeMap(() => this.fetchInitialReadForFilters(f))));
    }

    // TODO -- Handle disabled missing data synchronously
    // 1. Disabled feeds
    // 2. data from the past
    if (fetches.length) {
      return forkJoin(fetches)
          .pipe(mergeMap(() => this.filteredDataForFilters(f)));
    }
    return this.filteredDataForFilters(f);
  }

  // Fetches the initial page of read items if we don't already have any.
  private fetchInitialReadForFilters(f: Filters): Observable<void> {
    const fetches = [];

    if (f.feedId !== undefined) {
      const fd = this.feedMetadata.get(f.feedId);
      if (fd) {
        const readAfter = fd.readAfter;
        // It's possible for readAfter to be set by a category/all call, but for no read items to
        // actually be present for this feed.
        const hasLocalRead = fd.allRead ||
            (readAfter !== undefined &&
             !!this.data.items.find(
                 (it) => it.feedId === f.feedId && it.read && new Date(it.timestamp) >= readAfter));

        if (!hasLocalRead) {
          fetches.push(
              this.fetchMoreReadForFeed(f.feedId));
        }
      }
    } else if (f.categoryName !== undefined) {
      // This is going to be rare, and we already do linear searches in handleUpdates.
      let cd;
      for (let cm of this.categoryMetadata.values()) {
        if (cm.category.name === f.categoryName) {
          cd = cm;
          break;
        }
      }

      if (cd) {
        const readAfter = cd.readAfter;
        const categoryId = cd.category.id;
        const itemInCategory =
            ((it: Item) => this.feedMetadata.get(it.feedId)?.feed.categoryId === categoryId);

        // It's possible for readAfter to be set by an all feeds read call, but for no read items to
        // actually be present for this category.
        const hasLocalRead = cd.allRead ||
            (readAfter !== undefined &&
             !!this.data.items.find(
                 (it) => it.read && new Date(it.timestamp) >= readAfter && itemInCategory(it)));

        if (!hasLocalRead) {
          fetches.push(this.fetchMoreReadForCategory(categoryId));
        }
      }
    } else if (!this.hasRead()) {
      // If all the read items are in hidden categories, this might result in no read items being
      // visible on first click, but that's a niche enough problem it's not worth fixing.
      fetches.push(this.fetchMoreReadForAll());
    }

    if (!fetches.length) {
      return of(undefined);
    }
    return forkJoin(fetches).pipe(map(() => {}));
  }

  private filteredDataForFilters(f: Filters): Observable<FilteredData> {
    return this.data$
        .pipe(
            take(1),
            map((data: Data): FilteredData => new FilteredData(
                    data.filter(f), f)));
  }

  public fetchMoreReadForFeed(id: number): Observable<void> {
    const fd = this.feedMetadata.get(id);
    if (!fd || fd.allRead) {
      return of(undefined);
    }

    const readBefore = fd.readAfter || new Date(this.timestamp * 1000);
    return this.getItems({
      feedIds: [id],
      readBefore
    });
  }

  public fetchMoreReadForCategory(id: number): Observable<void> {
    const cm = this.categoryMetadata.get(id);
    if (!cm || cm.allRead) {
      return of(undefined);
    }

    const readBefore = cm.readAfter || new Date(this.timestamp * 1000);
    return this.getItems({
      categoryId: id,
      readBefore
    });
  }

  public fetchMoreReadForAll(): Observable<void> {
    if (this.allRead) {
      return of(undefined);
    }

    const readBefore = this.readAfter || new Date(this.timestamp * 1000);
    return this.getItems({readBefore});
  }

  public updates(): Observable<Updates> {
    return this.updates$;
  }

  public feedUpdates(): Observable<void> {
    return this.feedUpdates$;
  }

  public categoryUpdates(): Observable<void> {
    return this.categoryUpdates$;
  }

  // This should never be called for a feed that doesn't exist in the frontend
  public getFeed(id: number): Feed {
    if (!this.feedMetadata.has(id)) {
      const e = new Error('Tried to log non-existent feed ' + id);
      this.errorService.showError(e);
      throw e;
    }
    return (this.feedMetadata.get(id) as FeedMetadata).feed;
  }

  public feedAllRead(feed: Feed): boolean {
    const fd = this.feedMetadata.get(feed.id);
    return Boolean(fd?.allRead);
  }

  public categoryAllRead(category: Category): boolean {
    const cm = this.categoryMetadata.get(category.id);
    return Boolean(cm?.allRead);
  }

  public hasAllRead(): boolean {
    return this.allRead;
  }

  public getInitialTimestampForFeed(id: number): string|undefined {
    return this.initialNewestTimestamps && this.initialNewestTimestamps[id];
  }

  public getCategory(id: number): Category|undefined {
    if (!this.categoryMetadata.has(id)) {
      return;
    }
    return (this.categoryMetadata.get(id) as CategoryMetadata).category;
  }

  public pushUpdates(u: Updates) {
    this.handleUpdates(u);
  }

  // Only called when refreshing a main view with no read items visible
  public maybeCleanRead() {
    const now = new Date();
    if (now.valueOf() - this.lastClean.valueOf() < 24 * 60 * 60 * 1000) {
      return;
    }

    this.lastClean = now;
    for (let fm of this.feedMetadata.values()) {
      fm.allRead = false;
      fm.readAfter = undefined;
    }
    for (let cm of this.categoryMetadata.values()) {
      cm.allRead = false;
      cm.readAfter = undefined;
    }
    this.allRead = false;
    this.readAfter = undefined;

    const items = this.data.items.filter((it) => !it.read);
    // We don't need to update the data subject because nothing is listening for read items.
    this.data = new Data(this.data.categories, this.data.feeds, items);
  }

  private hasRead(): boolean {
    return this.allRead || this.readAfter !== undefined;
  }

  private handleRefresh(su: ServerUpdates): void {
    if (su.mustRefresh) {
      firstValueFrom(this.errorService.showError('Client state is too old, refreshing'))
          .then(() => window.location.reload());
      return;
    }
    const up = new Updates(true, su.categories, su.feeds, su.items);

    this.handleUpdates(up);
    if (su.timestamp > this.timestamp) {
      this.timestamp = su.timestamp;
    }
  }

  // Returns whether it replayed all data or not
  private handleUpdates(u: Updates, pushEmpty?: boolean): boolean {
    // Handle cases where feed/entities need to be fetched or replayed
    // Handle these asynchronously, after starting a spinner

    // Cases where read items need to be fetched:
    // An existing feed goes from disabled to enabled and we have read items
    //     for the all feeds (and we don't have them)
    // An existing feed goes from disabled to enabled and is part of a category
    //     (including just added) with read items fetched (and we don't have them)

    // A category going from enabled to disabled is kept but doesn't cause
    // fetches or replays.


    let replayed = false;
    const [d, changed] = this.data.merge(u);
    if (changed) {
      // Push to data first, so that subscribers of data can take the unchanged
      // fast-path
      this.data = d;
      const [mustReplay, backfillUnread, backfillRead] = this.mergeMetadata(u);
      this.maybeBackfill(backfillUnread, backfillRead);
      if (mustReplay) {
        replayed = true;
        // Replays are rare so it is fine to be inefficient
        u = new Updates(
            u.refresh, this.data.categories, this.data.feeds, this.data.items);
      }
      this.data$.next(this.data);
    } else {
      u = new Updates(u.refresh);
    }

    if (!u.isEmpty() || pushEmpty) {
      this.updates$.next(u);
    }
    return replayed;
  }

  // [mustReplay, backfillUnread, backfillRead]
  private mergeMetadata(
      u: Updates,
      isBackfill: boolean = false): [boolean, Set<number>, Set<number>] {
    const backfillUnread: Set<number> = new Set();
    const backfillRead: Set<number> = new Set();

    // True when all data needs to be replayed.
    // This can happen when a change can have a splash radius beyond the given category/feed.
    // Usually this means updates to the layout of the navigation menu.
    let mustReplay = false;

    u.categories.forEach((c) => {
      if (this.categoryMetadata.has(c.id)) {
        const m = this.categoryMetadata.get(c.id) as CategoryMetadata;
        if (m.category.commitTimestamp > c.commitTimestamp) {
          return;
        }
        const oldc = m.category;
        m.category = c;

        if (c.disabled !== oldc.disabled) {
          mustReplay = true;
        }

        if (oldc.hiddenMain && (!c.hiddenMain || c.disabled)) {
          mustReplay = true;
        }

        if (oldc && (!c.hiddenNav || c.disabled)) {
          mustReplay = true;
        }


        return;
      }

      this.categoryMetadata.set(
          c.id,
          new CategoryMetadata(c, false));
      mustReplay = true;
    });

    u.feeds.forEach((f) => {
      if (this.feedMetadata.has(f.id)) {
        const m = this.feedMetadata.get(f.id) as FeedMetadata;
        if (isBackfill) {
          m.hasUnread = true;
        }

        if (m.feed.commitTimestamp > f.commitTimestamp) {
          return;
        }
        const oldFeed = m.feed;
        m.feed = f;

        if (f.disabled) {
          return;
        }

        if (oldFeed.disabled) {
          mustReplay = true;
        }

        if (!m.hasUnread) {
          backfillUnread.add(f.id);
          // Fetch missing data
        }

        if (oldFeed.categoryId !== f.categoryId) {
          mustReplay = true;
        }

        if (f.categoryId !== undefined) {
          // Handle a feed being updated and hidden
          const cm = this.categoryMetadata.get(f.categoryId);
          if (cm && !cm.category.disabled &&
              (cm.category.hiddenNav || cm.category.hiddenMain)) {
            mustReplay = true;
          }

          if (cm && !m.hasReadAfter(cm.readAfter)) {
            backfillRead.add(f.id);
          }
        }

        if (!m.hasReadAfter(this.readAfter)) {
          backfillRead.add(f.id);
        }

        // Check for missing global time ranges and fetch missing data
        return;
      }

      this.feedMetadata.set(
          f.id,
          new FeedMetadata(
              f,
              isBackfill || f.createTimestamp >= this.timestamp,
              f.createTimestamp >= this.timestamp));
      if (f.disabled) {
        return;
      }

      if (f.categoryId !== undefined) {
        // Catch the case where a feed is both created and hidden
        const cm = this.categoryMetadata.get(f.categoryId);
        if (cm && !cm.category.disabled &&
            (cm.category.hiddenNav || cm.category.hiddenMain)) {
          mustReplay = true;
        }
      }

      if (!isBackfill && f.createTimestamp < this.timestamp) {
        // Do Fetches
        backfillUnread.add(f.id);
      }
    });


    return [mustReplay, backfillUnread, backfillRead];
  }

  private maybeBackfill(unread: Set<number>, read: Set<number>) {
    if (!unread.size && !read.size) {
      return;
    }

    if (unread.size) {
      this.getItems({
            feedIds: [...unread],
            unread: true,
          })
          .subscribe();
    }

    if (read.size) {
      // It should be rare that we need to backfill read items from multiple categories.
      // So just find the minimum read time between all categories and go with that.
      let minRead = this.readAfter;
      for (let fid of read) {
        const fm = this.feedMetadata.get(fid);
        if (!fm) {
          continue;
        }

        const cm = fm.feed.categoryId !== undefined && this.feedMetadata.get(fm.feed.categoryId);
        if (cm && cm.readAfter !== undefined) {
          if (minRead === undefined || cm.readAfter < minRead) {
            minRead = cm.readAfter;
          }
        }
      }

      this.getItems({
            feedIds: [...read],
            unread: false,
            readAfter: minRead,
          })
          .subscribe();
    }
  }

  private getFeedsInCategory(categoryId: number): Set<number> {
    const feeds = new Set<number>();
    for (let fm of this.feedMetadata.values()) {
      if (fm.feed.categoryId === categoryId) {
        feeds.add(fm.feed.id);
      }
    }
    return feeds;
  }

  private getItems(req: GetItemsRequest): Observable<void> {
    if (req.readBefore && !req.readBeforeCount) {
      req.readBeforeCount = READ_ITEMS_PAGE_SIZE;
    }

    let oldFeeds = new Set<number>();
    if (req.categoryId !== undefined) {
      oldFeeds = this.getFeedsInCategory(req.categoryId);
    }

    this.loadingService.startLoading();
    // forkJoin so that initial data is populated
    // Object notation doesn't work right
    const obs =
        forkJoin([
          this.http.post<GetItemsResponse>('/api/items', req),
          this.waitForInitialFetch()
        ])
            .pipe(
                map((results) => {
                  const resp = results[0];
                  const u = new Updates(false, [], resp.feeds, resp.items);
                  let allRead = false;
                  let minRead: Date|undefined;
                  let feedIds = req.feedIds;
                  let pushEmpty = false;

                  if (req.readBefore) {
                    minRead = req.readBefore;

                    resp.items.forEach((item: Item) => {
                      if (item.read &&
                          (!minRead || new Date(item.timestamp) < minRead)) {
                        minRead = new Date(item.timestamp);
                      }
                    });

                    const pageSize = req.readBeforeCount ||
                        READ_ITEMS_PAGE_SIZE;
                    if (resp.items.length < pageSize) {
                      // It's possible for some feeds inside a category to have
                      // allRead but for this to be false, but that's fine.
                      allRead = true;
                    }
                  }

                  const readAfter = req.readAfter || minRead;

                  // We don't trigger backfills from backfills
                  // There is a bug where this can cause missing items if a user
                  // updates a previously disabled feed to switch its categories
                  // between a refresh and when the backfill completes, but that is
                  // simply not worth handling.
                  // It could be better solved by preventing concurrent updates and refreshes.
                  const mustReplay = this.mergeMetadata(u, req.unread)[0];

                  if (req.categoryId === undefined && !feedIds && req.readBefore) {
                    this.readAfter = readAfter;
                    this.allRead = allRead;

                    for (let cm of this.categoryMetadata.values()) {
                      if (readAfter) {
                        cm.setReadAfter(readAfter);
                      }
                      if (allRead) {
                        cm.allRead = true;
                      }
                    }

                    for (let fm of this.feedMetadata.values()) {
                      if (readAfter) {
                        fm.setReadAfter(readAfter);
                      }
                      if (allRead) {
                        fm.allRead = true;
                      }
                    }
                  }

                  if (req.categoryId !== undefined && req.readBefore) {
                    const cm = this.categoryMetadata.get(req.categoryId);
                    if (cm) {
                      const newFeeds = this.getFeedsInCategory(req.categoryId);

                      if (readAfter) {
                        cm.setReadAfter(readAfter);
                      }

                      feedIds = [];
                      for (let fid of oldFeeds) {
                        if (newFeeds.has(fid)) {
                          feedIds.push(fid);
                        }
                      }

                      if (allRead) {
                        cm.allRead = true;
                      }
                    }
                  }

                  if (feedIds) {
                    feedIds.forEach((fid: number) => {
                      const fm = this.feedMetadata.get(fid);
                      if (!fm) {
                        return;
                      }

                      if (req.unread) {
                        fm.hasUnread = true;
                      }

                      if (readAfter) {
                        fm.setReadAfter(readAfter);
                      }

                      if (allRead) {
                        fm.allRead = true;
                      } else if (fm.feed.categoryId !== undefined) {
                        const cm = this.categoryMetadata.get(fm.feed.categoryId);
                        if (cm && cm.allRead) {
                          cm.allRead = false;
                          // The category itself hasn't changed, but the main view reads
                          // categoryMetadata.allRead on every update, so an update needs to happen.
                          pushEmpty = true;
                        }

                        if (this.allRead) {
                          this.allRead = false;
                          pushEmpty = true;
                        }
                      }
                    });
                  }

                  const replayed = this.handleUpdates(u, pushEmpty);
                  if (mustReplay && !replayed) {
                    this.updates$.next(
                        new Updates(
                            false,
                            this.data.categories,
                            this.data.feeds,
                            this.data.items));
                  }

                  this.loadingService.finishLoading();
                }),
                catchError((error: Error) => {
                  this.errorService.showError(error);
                  this.loadingService.finishLoading();
                  return of(undefined);
                }),
                share());

    obs.subscribe();
    return obs;
  }

  private getInitialState() {
    this.loadingService.startLoading();
    this.http.get<CurrentState>('/api/current')
        .subscribe({
          next: (state: CurrentState) => {
            this.timestamp = state.timestamp;
            this.initialNewestTimestamps = state.newestTimestamps || [];
            this.data = new Data(
                state.categories,
                state.feeds,
                state.items);

            this.data.feeds.forEach(
                (f) => this.feedMetadata.set(
                    f.id, new FeedMetadata(f, /* hasUnread= */ true, /* allRead= */ false)));

            this.data.categories.forEach(
                (c) => this.categoryMetadata.set(
                    c.id, new CategoryMetadata(c, /* allRead= */ false)));

            this.data$.next(this.data);
          },
          error: (error: Error) => {
            this.errorService.showError(error);
            this.loadingService.finishLoading();
            this.retryInitial = true;
          },
          complete: () => this.loadingService.finishLoading()
        });
  }

  private refreshState(): void {
    if (this.retryInitial) {
      this.retryInitial = false;
      this.refreshService.finishRefresh();
      this.getInitialState();
      return;
    } else if (this.timestamp === -1) {
      this.refreshService.finishRefresh();
      return;
    }
    this.loadingService.startLoading();
    this.http.get<ServerUpdates>(`/api/updates/${this.timestamp}`)
        .subscribe({
          next: (su: ServerUpdates) => this.handleRefresh(su),
          error: (error: Error) => {
            this.errorService.showError(error);
            this.refreshService.finishRefresh();
            this.loadingService.finishLoading();
          },
          complete: () => {
            this.refreshService.finishRefresh();
            this.loadingService.finishLoading();
          }
        });
  }

  private fetchDisabledFeeds(): Observable<void> {
    this.loadingService.startLoading();
    // forkJoin so that initial data is populated
    // Object notation doesn't work right
    const obs =
        forkJoin([
          this.http.get<Feed[]>(`/api/feeds/disabled`),
          this.waitForInitialFetch()
        ])
            .pipe(map((results) => {
                    this.handleUpdates(new Updates(false, [], results[0]));
                    this.hasAllFeeds = true;
                    this.loadingService.finishLoading();
                  }),
                  catchError((error: Error) => {
                    this.errorService.showError(error);
                    this.loadingService.finishLoading();
                    return of(undefined);
                  }),
                  share());

    obs.subscribe();
    return obs;
  }

  private waitForInitialFetch(): Observable<void> {
    return this.data$.pipe(take(1), map(() => {}));
  }
}
