import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {map,
        share,
        tap} from 'rxjs/operators';

import {Data,
        Updates} from '../models/data';
import {Feed,
        Item} from '../models/entities';

import {DataService} from './data.service';
import {ErrorService} from './error.service';
import {LoadingService} from './loading.service';

interface AddFeedResponse {
  candidates?: string[];
  feed?: Feed;
}

// MutateService is used to update components
// This only returns void observables so components know when actions have
// completed. All data updates are pushed through DataService.

// These methods fork observables so they don't need to be subscribed to
// unless the caller actually cares if they complete.
@Injectable({
  providedIn: 'root'
})
export class MutateService {
  constructor(
      private readonly dataService: DataService,
      private readonly http: HttpClient,
      private readonly errorService: ErrorService,
      private readonly loadingService: LoadingService) {}


  public markItemRead(it: Item, read: boolean): Observable<void> {
    const url = `/api/items/${it.id}/${read ? 'read' : 'unread'}`;

    this.loadingService.startLoading();
    const obs =
        this.http
            .post<Item>(url, {})
            .pipe(
                tap((nit: Item) =>
                        this.dataService.pushUpdates(
                            new Updates(false, [], [], [nit]))),
                share());

    this.subscribe(obs);
    return obs.pipe(map(() => {}));
  }

  public newFeed(feedUrl: string, title: string, force: boolean):
      Observable<string[]|void> {
    const url = `/api/feeds/add`;
    const request = {
      url: feedUrl,
      title,
      force
    };

    this.loadingService.startLoading();
    const obs =
        this.http
            .post<AddFeedResponse>(url, request)
            .pipe(
                map((resp: AddFeedResponse) => {
                  if (resp.feed) {
                    this.dataService.pushUpdates(
                        new Updates(false, [], [resp.feed]));
                  }

                  if (resp.candidates) {
                    return resp.candidates;
                  }
                }),
                share());
    this.subscribe(obs);
    return obs;
  }

  private subscribe(obs: Observable<any>) {
    obs.subscribe(
        () => this.loadingService.finishLoading(),
        (err: Error) => {
          this.errorService.showError(err);
          this.loadingService.finishLoading();
        });
  }
}
