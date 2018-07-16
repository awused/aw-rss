import {Feed} from '../types';

import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {ReplaySubject, Observable} from 'rxjs';
import {map, share} from 'rxjs/operators';

/**
 * Service that handles Feeds. Keeps an unsorted list of feeds that have been loaded at some point.
 */
@Injectable()
export class FeedService {
  private feeds_: Feed[];
  private feedsSubject_: ReplaySubject<Feed[]> = new ReplaySubject<Feed[]>(1);
  private feedsFetched_: boolean = false;

  constructor(private http_: Http) {}

  public getFeeds() {
    if (!this.feedsFetched_) {
      this.fetchFeeds_();
    }
    return this.feedsSubject_;
  }

  public reloadFeeds(): Observable<any> {
    if (this.feedsFetched_) {
      return this.fetchFeeds_();
    }

    return this.feedsSubject_;
  }

  // Private methods
  private notify_() {
    this.feedsSubject_.next(this.feeds_);
  }

  private fetchFeeds_(): Observable<any> {
    this.feedsFetched_ = true;

    const obs = this.http_.get('/api/feeds/list')
        .pipe(map(response => response.json()), share());
    
    obs.subscribe(data => {
      if (data.error) {
        console.error(data.error);
        return;
      }
      this.feeds_ = data;
      this.notify_();
    }, error => console.error(error));

    return obs;
  }
}

