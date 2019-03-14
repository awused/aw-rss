import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {ReplaySubject,
        Subject} from 'rxjs';
import {share} from 'rxjs/operators';

import {Category} from '../models/category';
import {Data,
        Updates} from '../models/data';
import {Feed} from '../models/feed';
import {Item} from '../models/item';

import {ErrorService} from './error.service';


// Data about what is present in the cache for this feed
class FeedMetadata {
  hasAllItems: boolean = false;
  // TODO -- track oldest read item for pagination

  constructor(
      readonly feed: Feed) {}
}

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
  incomplete?: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class DataService {
  private timestamp: number = -1;
  private data: Data = new Data();
  private readonly dataSubject: ReplaySubject<Data> = new ReplaySubject<Data>(1);
  private readonly updateSubject: Subject<Updates> = new Subject<Updates>();
  private hasAllFeeds: boolean = false;
  private hasAllCategories: boolean = false;
  private hasAllItems: boolean = false;

  constructor(
      private readonly http: HttpClient,
      private readonly errorService: ErrorService) {
    this.getInitialState();
  }

  public updateState(): Promise<void> {
    if (this.timestamp == -1) {
      return this.dataSubject.toPromise().then(() => {});
    }
    const obs = this.http.get<ServerUpdates>(`/api/updates/${this.timestamp}`)
                    .pipe(share());

    obs.subscribe(
        this.handleServerUpdates(true),
        (error: Error) => this.errorService.showError(error));
    return obs.toPromise().then(() => {});
  }

  private handleServerUpdates(refresh: boolean = false): (ServerUpdates) => void {
    return (su: ServerUpdates) => {
      // TODO -- abort on incomplete, or keep going
      const d = new Data(su.categories, su.feeds, su.items);
      const up = new Updates(refresh, d, su.timestamp);

      console.log(su);
      this.handleUpdates(up);
    };
  }

  private handleUpdates(u: Updates) {
    if (u.refresh && u.timestamp > this.timestamp) {
      this.timestamp = u.timestamp;
    }
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
              state.categories || [],
              state.feeds || [],
              state.items || []);

          console.log(this.data);
          this.dataSubject.next(this.data);
          this.updateState();
        }, (error: Error) => this.errorService.showError(error));
  }
}
