import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  ReplaySubject,
  Subject
} from 'rxjs';
import {
  map,
  share,
  take
} from 'rxjs/operators';

import {Data,
        FilteredData} from '../models/data';
import {Category,
        Feed,
        Item} from '../models/entities';
import {Filters} from '../models/filter';
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
      public hasUnreadItems: boolean = true,
      // 0 -> we have all items
      // This only applies to read items that have been deliberately fetched
      public oldestReadItemId?: number) {}
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
            // TODO -- Handle interesting cases here
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
    if (su.timestamp > this.timestamp) {
      this.timestamp = su.timestamp;
    }
    const d = new Data(su.categories, su.feeds, su.items);
    const up = new Updates(true, d);

    this.handleUpdates(up);
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
    this.data = this.data.merge(u)[0];
    this.updates$.next(u);
    this.data$.next(this.data);
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

              // TODO -- Construct metadata
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
