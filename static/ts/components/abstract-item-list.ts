import {FeedService} from '../services/feed';
import {ItemService} from '../services/item';
import {Feed, Item} from '../types';

import {Component, OnInit} from '@angular/core';
import {Observable, zip} from 'rxjs';
import {DomSanitizer, SafeResourceUrl} from '@angular/platform-browser';
import {SecurityContext} from '@angular/core';

export abstract class AbstractItemList {
  protected feeds_: {[key:number]:Feed} = {};
  protected items_: Item[] = [];
  private refreshing_: boolean = false;
  private error_: Error | null; 

  constructor(
    protected feedService_: FeedService,
    protected itemService_: ItemService,
    private domSanitizer_: DomSanitizer) {
  }

  abstract ngOnInit(): void;

  markItemAsRead(id: number): Observable<Item> {
    return this.itemService_.markItemAsRead(id);
  }
  
  markItemAsUnread(id: number): Observable<Item> {
    return this.itemService_.markItemAsUnread(id);
  }

  getFeed(fid: number): Feed {
    return this.feeds_[fid];
  }

  getItems(): Item[] {
    return this.items_;
  }

  isRefreshing(): boolean {
    return this.refreshing_;
  }

  getError(): Error | null {
    return this.error_;
  }

  getItemUrl(item: Item): SafeResourceUrl | string | null {
    if (item.url.startsWith('magnet:')) {
      return this.domSanitizer_.bypassSecurityTrustUrl(item.url);
    }
    return this.domSanitizer_.sanitize(SecurityContext.URL, item.url);
  }

  // TODO: factor this out into some kind of plugin system
  getItemMobileUrl(item: Item): string | undefined {
    if (item.url.startsWith("https://manga.madokami.al/")) {
      let path = item.url.split("madokami.al")[1];
      path = encodeURIComponent(path);
      path = path.replace(/%2F/g, "%252F");
      return "https://manga.madokami.al/reader/" + path;
    }
  }

  handleItemClick($event: MouseEvent, item: Item) {
    if ($event.button == 1 && !item.read) {
      this.markItemAsRead(item.id);
    }
  }

  refresh() {
    this.refreshing_ = true;

    zip(this.feedService_.reloadFeeds(), this.itemService_.reloadItems())
        .subscribe(() => {
          this.error_ = null;
          this.refreshing_ = false;
        }, (error) => {
          this.error_ = error;
          this.refreshing_ = false;
        });
  }
}
