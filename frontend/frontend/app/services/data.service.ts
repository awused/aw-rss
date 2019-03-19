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
  tap
} from 'rxjs/operators';

import {Data,
        FilteredData} from '../models/data';
import {Category,
        Feed,
        Item} from '../models/entities';
import {Filters,
        TimeRange} from '../models/filter';
import {Updates} from '../models/updates';

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


// Data about what is present in the cache for this feed
class FeedMetadata {
  constructor(
      public feed: Feed,
      // True if we have all unread items up to DataService.timestamp
      public hasUnread: boolean = true,
      // The time ranges of fetches read items, in order
      // Overlapping time ranges are merged
      public readTimeRanges: TimeRange[] = []) {}
}

// Data about what is present in the cache for this feed
class CategoryMetadata {
  public readonly feedIds: Set<number>;

  constructor(
      public category: Category,
      // All enabled feeds in this category have read items back _at least_ this far
      // 0 -> we have all items
      // This only applies to read items that have been deliberately fetched
      public oldestReadItemId?: number) {
    // TODO -- Add feedIds
    // this.feedIds = new Set(this.category.feedIds);
  }
}

// A component that cares about read items in a category will need to subscribe
// so that category mutations result in appropriate replays/fetches
export interface CategorySubscription {
  readonly category: number;
  readonly readTimeRange?: TimeRange;
}



@Injectable({
  providedIn: 'root'
})
export class DataService {
  private timestamp = -1;
  private data: Data = new Data();
  private readonly data$: ReplaySubject<Data> = new ReplaySubject<Data>(1);
  private readonly updates$: Subject<Updates> = new Subject<Updates>();
  // All enabled feeds have read items back _at least_ this far
  // This only applies to read items that have been deliberately fetched
  private oldestReadItemId: number|undefined = undefined;
  private hasAllFeeds = false;
  private hasAllCategories = false;
  private feedMetadata: Map<number, FeedMetadata> = new Map();
  private categoryMetadata: Map<number, CategoryMetadata> = new Map();

  constructor(
      private readonly http: HttpClient,
      private readonly errorService: ErrorService,
      private readonly refreshService: RefreshService,
      private readonly loadingService: LoadingService) {
    this.getInitialState();
    // The data service will never be destroyed, so never unsubscribe
    this.refreshService.startedObservable()
        .subscribe(() => this.refreshState());
  }

  public dataForFilters(f: Filters): Observable<FilteredData> {
    return this.data$
        .pipe(
            take(1),
            // TODO -- Handle interesting missing data cases here synchronously
            map((data: Data): FilteredData => {
              return new FilteredData(
                  data.filter(f), f);
            }));
  }

  public updates(): Observable<Updates> {
    return this.updates$;
  }

  // This should never be called for a feed that doesn't exist in data
  public getFeed(id: number): Feed {
    if (!this.feedMetadata.has(id)) {
      console.log(id);
    }
    return this.feedMetadata.get(id).feed;
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
    const up = new Updates(true, d);

    this.handleUpdates(up);
    if (su.timestamp > this.timestamp) {
      this.timestamp = su.timestamp;
    }
  }

  private handleUpdates(u: Updates) {
    // Handle cases where feed/entities need to be fetched or replayed

    // Cases where unread items need to be fetched:
    // An "existing" (create_timestamp < timestamp) feed goes from disabled to
    //     enabled and we don't already have the unread items for it

    // Cases where read items need to be fetched:
    // An existing feed goes from disabled to enabled and we have read items
    //     for the all feeds (and we don't have them)
    // An existing feed goes from disabled to enabled and is part of a category
    //     (including just added) with read items fetched (and we don't have them)

    // Cases where feeds (and items of those feeds) need to be replayed:
    // Any cases where a fetch would have happened but we already had the data
    // -- existing feed disabled -> enabled
    // -- part of a category or we had the read items
    // A feed that wasn't previously disabled is added to a category

    // A category going from enabled to disabled is kept but doesn't cause
    // fetches or replays. Users are kicked off of those category pages to the
    // root.
    // A disabled category getting enabled just recalculates the metadata.

    // Handle feeds, then items, then categories
    let changed;
    let d;
    [d, changed] = this.data.merge(u);
    if (changed) {
      // Push to data first, so that subscribers of data can take the unchanged
      // fast-path
      this.data = d;
      this.updateMetadata(u);
      this.data$.next(this.data);
      this.updates$.next(u);
    } else if (u.refresh) {
      this.updates$.next(new Updates(true, new Data()));
    }
  }

  private updateMetadata(u: Updates) {
    u.data.feeds.forEach((f) => {
      if (this.feedMetadata.has(f.id)) {
        const m = this.feedMetadata.get(f.id);
        if (m.feed.commitTimestamp <= f.commitTimestamp) {
          m.feed = f;
        }
        return;
      }

      this.feedMetadata.set(
          f.id,
          new FeedMetadata(f, f.createTimestamp > this.timestamp));
    });

    u.data.categories.forEach((c) => {});
  }

  private getInitialState() {
    this.loadingService.startLoading();
    this.http.get<CurrentState>('/api/current')
        .subscribe(
            (state: CurrentState) => {
              this.timestamp = state.timestamp;
              this.data = new Data(
                  state.categories,
                  state.feeds,
                  state.items);

              this.data.feeds.forEach(
                  (f) => this.feedMetadata.set(f.id, new FeedMetadata(f)));
              this.data.categories.forEach(
                  (c) => this.categoryMetadata.set(c.id, new CategoryMetadata(c)));
              this.data$.next(this.data);
            },
            (error: Error) => {
              this.errorService.showError(error);
              this.loadingService.finishLoading();
            },
            () => this.loadingService.finishLoading(),
        );
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
