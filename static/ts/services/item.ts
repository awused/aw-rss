import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable, ReplaySubject} from 'rxjs';
import {map, share} from 'rxjs/operators';
import {Item} from '../types';

/**
 * Service that handles Items.
 * Keeps a sorted list of items that have been unread since the last refresh.
 * This array will include read items or items for disabled/hidden feeds.
 */
@Injectable()
export class ItemService {
  /**
   * May include read items if they've been marked read since the last refresh.
   */
  private items_: Item[] = [];
  private itemsSubject_: ReplaySubject<Item[]> = new ReplaySubject<Item[]>(1);
  private itemsFetched_: boolean = false;

  constructor(private http_: Http) {}
  /*
   * --- Naive, brute force solution
   * -- have some ideas to improve this but they're probably not worth the
   * complexity
   * // Gets unread items from fro
   * getUnreadItems()
   * getItemsForFeed()
   *
   * private updateSubject
   * updates()
   * updatesForFeed(fid)
   * updatesForCategory?
   * updatesForItem(id, fid)
   *
   * ItemUpdates
   * {fid : [items changed]}
   */

  // getItemsSubject
  public getAllUnreadItems() {
    if (!this.itemsFetched_) {
      this.fetchItems_();
    }
    return this.itemsSubject_;
  }

  public markItemAsRead(id: number): Observable<Item> {
    var obs = this.http_.post('/api/items/' + id + '/read', '')
                  .pipe(map(response => response.json()), share());

    obs.subscribe(data => {
      if (data.error) {
        console.error(data.error);
        return;
      }
      this.updateItem_(data);
    }, error => console.error(error));

    return obs;
  }

  public markItemAsUnread(id: number): Observable<Item> {
    const obs = this.http_.post('/api/items/' + id + '/unread', '')
                    .pipe(map(response => response.json()), share());

    obs.subscribe(data => {
      if (data.error) {
        console.error(data.error);
        return;
      }
      this.updateItem_(data);
    }, error => console.error(error));

    return obs;
  }

  public reloadItems(): Observable<any> {
    if (this.itemsFetched_) {
      return this.fetchItems_();
    }
    return this.itemsSubject_;
  }

  // Private methods
  private notify_() { this.itemsSubject_.next(this.items_); }

  // Updates an existing item or adds an unread item to the list.
  private updateItem_(updatedItem: Item) {
    var idx = this.items_.findIndex((item: Item) => item.id === updatedItem.id);
    if (idx != -1) {
      this.items_[idx] = updatedItem;
    } else if (!updatedItem.read) {
      idx = 0;
      while (idx < this.items_.length &&
             this.items_[idx].timestamp > updatedItem.timestamp) {
        idx++;
      }
      this.items_.splice(idx, 0, updatedItem);
    } else {
      return;
    }

    this.notify_();
  }

  private fetchItems_(): Observable<any> {
    this.itemsFetched_ = true;

    const obs = this.http_.get('/api/items/list')
                    .pipe(map(response => response.json()), share());

    obs.subscribe(data => {
      if (data.error) {
        console.error(data.error);
        return;
      }
      this.items_ = data;
      this.notify_();
    }, error => console.error(error));

    return obs;
  }
}
