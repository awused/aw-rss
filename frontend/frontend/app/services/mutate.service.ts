import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {map,
        share,
        tap} from 'rxjs/operators';

import {Data} from '../models/data';
import {Item} from '../models/entities';
import {Updates} from '../models/updates';

import {DataService} from './data.service';
import {ErrorService} from './error.service';
import {LoadingService} from './loading.service';

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
    const url = `/ap./entitiess/${it.id}/${read ? 'read' : 'unread'}`;

    this.loadingService.startLoading();
    const obs =
        this.http
            .post<Item>(url, {})
            .pipe(
                tap((nit: Item) =>
                        this.dataService.pushUpdates(
                            new Updates(false, new Data([], [], [nit])))),
                share());

    this.subscribe(obs);
    return obs.pipe(map(() => {}));
  }

  private subscribe(obs: Observable<any>) {
    obs.subscribe(
        () => {},
        (err: Error) => this.errorService.showError(err),
        () => this.loadingService.finishLoading());
  }
}
