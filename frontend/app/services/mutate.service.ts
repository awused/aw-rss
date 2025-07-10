import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {map,
        share} from 'rxjs/operators';

import {Updates} from '../models/data';
import {
  Category,
  Feed,
  Item
} from '../models/entities';

import {DataService} from './data.service';
import {ErrorService} from './error.service';
import {LoadingService} from './loading.service';

interface AddFeedResponse {
  status: 'success'|'invalid'|'candidates';
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

    // The data service will merge in an item with the same commit_timestamp.
    // This will be overridden by the API response.
    const optimisticItem: Item = Object.assign({}, it, {read});

    this.loadingService.startLoading();
    this.dataService.pushUpdates(new Updates(false, [], [], [optimisticItem]));
    const obs =
        this.http
            .post<Item>(url, {})
            .pipe(
                map((nit: Item) =>
                        this.dataService.pushUpdates(
                            new Updates(false, [], [], [nit]))),
                share());

    this.subscribe(
        obs,
        () => this.dataService.pushUpdates(new Updates(false, [], [], [it])));
    return obs.pipe();
  }

  public newFeed(feedUrl: string, title: string, force: boolean):
      Observable<string[]|'invalid'|void> {
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
                    return this.dataService.pushUpdates(
                        new Updates(false, [], [resp.feed]));
                  }

                  if (resp.status === 'invalid') {
                    return resp.status;
                  }

                  if (resp.candidates) {
                    return resp.candidates;
                  }
                }),
                share());
    this.subscribe(obs);
    return obs;
  }

  public editFeed(feed: Feed, edit: {
    categoryId?: number;
    clearCategory?: boolean;
    disabled?: boolean;
    userTitle?: string;
  }): Observable<void> {
    const url = `/api/feeds/${feed.id}/edit`;
    const req = {edit};

    const merge: any = {};
    if (edit.categoryId !== undefined) {
      merge.categoryId = edit.categoryId;
    } else if (edit.clearCategory) {
      merge.categoryId = undefined;
    }

    if (edit.disabled !== undefined) {
      merge.disabled = edit.disabled;
    }

    if (edit.userTitle !== undefined) {
      merge.userTitle = edit.userTitle;
    }

    const optimisticFeed = Object.assign({}, feed, merge);


    this.loadingService.startLoading();
    this.dataService.pushUpdates(new Updates(false, [], [optimisticFeed]));
    const obs =
        this.http.post<Feed>(url, req)
            .pipe(
                map((f: Feed) =>
                        this.dataService.pushUpdates(
                            new Updates(false, [], [f]))),
                share());

    this.subscribe(
        obs,
        () => this.dataService.pushUpdates(new Updates(false, [], [feed])));
    return obs;
  }

  public newCategory(req: {
    name: string;
    title: string;
    hiddenNav: boolean;
    hiddenMain: boolean;
  }): Observable<void> {
    const url = `/api/categories/add`;

    this.loadingService.startLoading();
    const obs =
        this.http
            .post<Category>(url, req)
            .pipe(
                map((cat: Category) =>
                        this.dataService.pushUpdates(
                            new Updates(false, [cat]))),
                share());
    this.subscribe(obs);
    return obs;
  }

  public editCategory(category: Category, edit: {
    name?: string;
    title?: string;
    hiddenMain?: boolean;
    hiddenNav?: boolean;
    disabled?: boolean;
  }): Observable<void> {
    const url = `/api/categories/${category.id}/edit`;
    const req = {edit};

    const optimisticCategory = Object.assign({}, category, edit);

    this.loadingService.startLoading();
    this.dataService.pushUpdates(new Updates(false, [optimisticCategory]));
    const obs =
        this.http.post<Category>(url, req)
            .pipe(
                map((c: Category) =>
                        this.dataService.pushUpdates(
                            new Updates(false, [c]))),
                share());

    this.subscribe(
        obs,
        () => this.dataService.pushUpdates(new Updates(false, [category])));
    return obs;
  }

  public markFeedAsRead(
      feedId: number,
      maxItemId: number,
      ): Observable<void> {
    const url = `/api/feeds/${feedId}/read`;

    this.loadingService.startLoading();
    const obs =
        this.http.post<{items: Item[]}>(url, {maxItemId})
            .pipe(
                map((response: {items: Item[]}) =>
                        this.dataService.pushUpdates(
                            new Updates(false, [], [], response.items))),
                share());

    this.subscribe(obs);
    return obs;
  }

  public markBulkItemsAsRead(
      feedId: number,
      itemIds: number[],
      ): Observable<void> {
    const url = `/api/items/read`;

    this.loadingService.startLoading();
    const obs =
        this.http.post<{items: Item[]}>(url, {feedId, itemIds})
            .pipe(
                map((response: {items: Item[]}) =>
                        this.dataService.pushUpdates(
                            new Updates(false, [], [], response.items))),
                share());

    this.subscribe(obs);
    return obs;
  }

  // For now, this is fire and forget
  public rerunFeed(
      feedId: number,
      ): Observable<void> {
    const url = `/api/feeds/${feedId}/rerun`;

    this.loadingService.startLoading();
    const obs =
        this.http.post<Record<string, never>>(url, {})
            .pipe(
                map(() => undefined),
                share());

    this.subscribe(obs);
    return obs;
  }

  public rerunFailingFeeds(): Observable<void> {
    const url = `/api/feeds/rerun-failing`;

    this.loadingService.startLoading();
    const obs =
        this.http.post<Record<string, never>>(url, {})
            .pipe(
                map(() => undefined),
                share());

    this.subscribe(obs);
    return obs;
  }

  public reorderCategories(categoryIds: number[]): Observable<void> {
    // This is probably rare enough to not bother doing optimistically
    const url = `/api/categories/reorder`;

    this.loadingService.startLoading();
    const obs =
        this.http.post<{categories: Category[]}>(url, {categoryIds})
            .pipe(
                map((response: {categories: Category[]}) =>
                        this.dataService.pushUpdates(
                            new Updates(false, response.categories))),
                share());

    this.subscribe(obs);
    return obs;
  }

  private subscribe(
      obs: Observable<any>,
      rollback: () => void = () => undefined) {
    obs.subscribe({
      next: () => this.loadingService.finishLoading(),
      error: (err: Error) => {
        this.errorService.showError(err);
        rollback();
        this.loadingService.finishLoading();
      }
    });
  }
}
