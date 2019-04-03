import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  ReplaySubject,
  Subject
} from 'rxjs';
import {
  filter,
  map,
  share,
  take,
} from 'rxjs/operators';

import {Data,
        FilteredData,
        Updates} from '../models/data';
import {Category,
        Feed,
        Item} from '../models/entities';
import {Filters,
        TimeRange} from '../models/filter';

import {ErrorService} from './error.service';
import {LoadingService} from './loading.service';
import {RefreshService} from './refresh.service';

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
  categoryId?: number;
  feedIds?: number[];
  includeFeeds?: boolean;
  unread?: boolean;
  readBefore?: Date;
  readBeforeCount?: number;
  readAfter?: Date;
}

interface GetItemsResponse {
  items: Item[];
  feeds?: Feed[];
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
      private readAfter?: Date) {}

  public setReadAfter(d: Date) {
    if (d <= new Date(this.feed.createTimestamp + '000')) {
      this.allRead = true;
    }
    this.readAfter = d;
  }

  public hasReadAfter(d: Date) {
    return this.allRead || this.readAfter >= d;
  }
}

// Data about what is present in the cache for this category
class CategoryMetadata {
  public readonly feedIds: Set<number>;

  constructor(
      public category: Category) {}
}

// A component that cares about read items will need to subscribe
// so that missing data is fetched when necessary
export interface ReadSubscription {
  readonly category?: number;
  readonly readAfter: Date;
}



@Injectable({
  providedIn: 'root'
})
export class DataService {
  private timestamp = -1;
  private data: Data = new Data();
  private readonly data$: ReplaySubject<Data> = new ReplaySubject<Data>(1);
  private readonly updates$: Subject<Updates> = new Subject<Updates>();
  private readonly feedUpdates$: Observable<void>;
  private readonly categoryUpdates$: Observable<void>;
  // All enabled feeds have read items back _at least_ this far
  // This only applies to read items that have been deliberately fetched
  private oldestReadItemId: number|undefined = undefined;
  private hasAllFeeds = false;
  private hasAllCategories = false;
  private feedMetadata: Map<number, FeedMetadata> = new Map();
  private categoryMetadata: Map<number, CategoryMetadata> = new Map();
  // For now it's enough to just do this at initial load
  private initialNewestTimestamps: {[x: number]: string};

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
    // TODO -- Handle disabled missing data synchronously
    // 1. Disabled feeds
    // 2. data from the past
    return this.data$
        .pipe(
            take(1),
            map((data: Data): FilteredData => {
              return new FilteredData(
                  data.filter(f), f);
            }));
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
    return this.feedMetadata.get(id).feed;
  }


  public getInitialTimestampForFeed(id: number): string|undefined {
    return this.initialNewestTimestamps[id];
  }

  public getCategory(id: number): Category|undefined {
    if (!this.categoryMetadata.has(id)) {
      return;
    }
    return this.categoryMetadata.get(id).category;
  }

  public pushUpdates(u: Updates) {
    this.handleUpdates(u);
  }

  private handleRefresh(su: ServerUpdates): void {
    if (su.mustRefresh) {
      this.errorService.showError('Client state is too old, refreshing')
          .toPromise()
          .then(() => window.location.reload());
      return;
    }
    const d = new Data(su.categories, su.feeds, su.items);
    const up = new Updates(true, su.categories, su.feeds, su.items);

    this.handleUpdates(up);
    if (su.timestamp > this.timestamp) {
      this.timestamp = su.timestamp;
    }
  }

  // Returns whether it replayed all data or not
  private handleUpdates(u: Updates): boolean {
    // Handle cases where feed/entities need to be fetched or replayed
    // Handle these asynchronously, after starting a spinner

    // Cases where read items need to be fetched:
    // An existing feed goes from disabled to enabled and we have read items
    //     for the all feeds (and we don't have them)
    // An existing feed goes from disabled to enabled and is part of a category
    //     (including just added) with read items fetched (and we don't have them)

    // A category going from enabled to disabled is kept but doesn't cause
    // fetches or replays.


    let changed;
    let d;
    let replayed = false;
    [d, changed] = this.data.merge(u);
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

    if (!u.isEmpty()) {
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

    let mustReplay = false;

    u.categories.forEach((c) => {
      if (this.categoryMetadata.has(c.id)) {
        const m = this.categoryMetadata.get(c.id);
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
          new CategoryMetadata(c));
      mustReplay = true;
    });

    u.feeds.forEach((f) => {
      if (this.feedMetadata.has(f.id)) {
        const m = this.feedMetadata.get(f.id);
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
          // if (this.sub && (!this.sub.id || f.categoryId == this.sub.id)) {
          // Check for missing time ranges in category subscriptions
          // Fetch if necessary
          // }
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
        mustReplay = true;
      }

      if (!isBackfill && f.createTimestamp < this.timestamp) {
        // Do Fetches
        backfillUnread.add(f.id);

        // if (this.sub && (!this.sub.id || f.categoryId == this.sub.id)) {
        // Check for missing time ranges in category subscriptions
        // Fetch if necessary
        // }
      }
    });


    return [mustReplay, backfillUnread, backfillRead];
  }

  private maybeBackfill(unread: Set<number>, read: Set<number>) {
    if (!unread.size && !read.size) {
      return;
    }

    // We will never have feeds where we only care about read items
    const unreadOnly = [...unread].filter((x) => !read.has(x));
    if (unreadOnly.length) {
      this.getItems({
        feedIds: [...unreadOnly],
        unread: true,
      });
    }
    if (read.size) {
      // TODO
      console.log(`Would fetch read items for ${[...read]}`);
    }
  }

  private getItems(req: GetItemsRequest) {
    this.loadingService.startLoading();
    this.http.post<GetItemsResponse>('/api/items', req)
        .subscribe(
            (resp: GetItemsResponse) => {
              const u = new Updates(false, [], resp.feeds, resp.items);
              // We don't trigger backfills from backfills
              // There is a bug where this can cause missing items if a user
              // updates a previously disabled feed to switch its categories
              // between a refresh and when the backfill completes, but that is
              // simply not worth handling.
              const mustReplay = this.mergeMetadata(u, req.unread)[0];
              req.feedIds.forEach((fid: number) => {
                const fm = this.feedMetadata.get(fid);
                if (!fm) {
                  return;
                }

                if (req.unread) {
                  fm.hasUnread = true;
                }
                // TODO -- fill in time ranges here
              });
              const replayed = this.handleUpdates(u);
              if (mustReplay && !replayed) {
                this.updates$.next(
                    new Updates(
                        false,
                        this.data.categories,
                        this.data.feeds,
                        this.data.items));
              }
            },
            (error: Error) => {
              this.errorService.showError(error);
              this.loadingService.finishLoading();
            },
            () => this.loadingService.finishLoading());
  }

  private getInitialState() {
    this.loadingService.startLoading();
    this.http.get<CurrentState>('/api/current')
        .subscribe(
            (state: CurrentState) => {
              this.timestamp = state.timestamp;
              this.initialNewestTimestamps = state.newestTimestamps;
              this.data = new Data(
                  state.categories,
                  state.feeds,
                  state.items);

              this.data.feeds.forEach(
                  (f) => this.feedMetadata.set(
                      f.id, new FeedMetadata(f, true, false)));

              this.data.categories.forEach(
                  (c) => this.categoryMetadata.set(
                      c.id, new CategoryMetadata(c)));

              this.data$.next(this.data);
            },
            (error: Error) => {
              this.errorService.showError(error);
              this.loadingService.finishLoading();
            },
            () => this.loadingService.finishLoading());
  }

  private refreshState(): void {
    if (this.timestamp === -1) {
      return;
    }
    this.loadingService.startLoading();
    this.http.get<ServerUpdates>(`/api/updates/${this.timestamp}`)
        .subscribe(
            (su: ServerUpdates) => this.handleRefresh(su),
            (error: Error) => {
              this.errorService.showError(error);
              this.refreshService.finishRefresh();
              this.loadingService.finishLoading();
            },
            () => {
              this.refreshService.finishRefresh();
              this.loadingService.finishLoading();
            });
  }
}
