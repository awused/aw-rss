import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {
  BehaviorSubject,
  ReplaySubject,
  Subject
} from 'rxjs';
import {share} from 'rxjs/operators';

import {Category} from '../models/category';
import {Data,
        Updates} from '../models/data';
import {Feed} from '../models/feed';
import {Item} from '../models/item';

import {ErrorService} from './error.service';
import {RefreshService} from './refresh.service';

interface CurrentState {
  timestamp: number;
  feeds?: Feed[];
  items?: Item[];
  categories?: Category[];
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
      public oldestReadItemId: number|undefined = undefined) {}
}

// Data about what is present in the cache for this feed
class CategoryMetadata {
  public readonly feedIds: Set<number>;

  constructor(
      public category: Category,
      // All enabled feeds in this category have read items back _at least_ this far
      // 0 -> we have all items
      // This only applies to read items that have been deliberately fetched
      public oldestReadItemId: number|undefined = undefined) {
    // TODO -- Add feedIds
    //this.feedIds = new Set(this.category.feedIds);
  }
}
@Injectable({
  providedIn: 'root'
})
export class DataService {
  private timestamp: number = -1;
  private data: Data = new Data();
  private readonly dataSubject: ReplaySubject<Data> = new ReplaySubject<Data>(1);
  private readonly updateSubject: Subject<Updates> = new Subject<Updates>();
  // All enabled feeds have read items back _at least_ this far
  // This only applies to read items that have been deliberately fetched
  private oldestReadItemId: number|undefined = undefined;
  private hasAllFeeds: boolean = false;
  private hasAllCategories: boolean = false;
  private feedMetadata: Map<number, FeedMetadata> = new Map();
  private categoryMetadata: Map<number, CategoryMetadata> = new Map();

  constructor(
      private readonly http: HttpClient,
      private readonly errorService: ErrorService,
      private readonly refreshService: RefreshService) {
    this.getInitialState();
    // The data service will never be destroyed, so never unsubscribe
    this.refreshService.startedObservable()
        .subscribe(() => this.refreshState());
  }

  private refreshState(): void {
    if (this.timestamp == -1) {
      return;
    }
    this.http.get<ServerUpdates>(`/api/updates/${this.timestamp}`)
        .subscribe(
            (su: ServerUpdates) => this.handleRefresh(su),
            (error: Error) => this.errorService.showError(error),
            () => this.refreshService.finishRefresh());
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

    console.log(su);
    this.handleUpdates(up);
  }

  private handleUpdates(u: Updates) {
    // TODO-- Remove LIMIT / incomplete
    // Handle cases where feeds/items need to be fetched or replayed
    // TODO -- ADD create_timestamp to feed

    // Cases where unread items need to be fetched:
    // An "existing" (create_timestamp < timestamp) feed goes from disabled to enabled and we don't already have the unread items for it

    // Cases where read items need to be fetched:
    // An existing feed goes from disabled to enabled and we have read items for the all feeds (and we don't have them)
    // An existing feed goes from disabled to enabled and is part of a category (including just added) with read items fetched (and we don't have them)

    // Cases where feeds (and items of those feeds) need to be replayed:
    // Any cases where a fetch would have happened but we already had the data
    // -- existing feed disabled -> enabled
    // -- part of a category or we had the read items
    // A feed that wasn't previously disabled is added to a category

    // A category going from disabled to enabled never causes a replay
    this.data = this.data.merge(u);
    this.dataSubject.next(this.data);
    this.updateSubject.next(u);
    console.log(this.data);
    this.errorService.showError('we did it');
  }

  private getInitialState() {
    this.http.get<CurrentState>('/api/current')
        .subscribe((state: CurrentState) => {
          this.timestamp = state.timestamp;
          this.data = new Data(
              state.categories,
              state.feeds,
              state.items);

          console.log(this.data);
          this.dataSubject.next(this.data);
          this.refreshState();
        }, (error: Error) => this.errorService.showError(error));
  }
}
