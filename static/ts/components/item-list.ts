import {FeedService} from '../services/feed';
import {ItemService} from '../services/item';
import {Feed, Item} from '../types';
import {AbstractItemList} from './abstract-item-list';

import {Component, OnInit, HostListener} from '@angular/core';
import {DomSanitizer} from '@angular/platform-browser';

@Component({
  selector: 'item-list',
  templateUrl: '/static/templates/item-list.html'
})

/**
 * An item list that shows unread items for enabled feeds, or items that were marked read since the last refresh.
 * This is the most optimized case.
 */
export class ItemList extends AbstractItemList {
  constructor(
      feedService_: FeedService,
      itemService_: ItemService,
      domSanitizationService_: DomSanitizer) {
    super(feedService_, itemService_, domSanitizationService_)
  }
  
  ngOnInit() {
    this.feedService_.getFeeds().subscribe(updatedFeeds => {
      this.feeds_ = {};
      for (var f of updatedFeeds) {
        this.feeds_[f.id] = f;
      }
    });
    this.itemService_.getAllUnreadItems().subscribe(updatedItems => {
      this.items_ = updatedItems;
    });
  }

  @HostListener('window:keydown', ['$event'])
  handleKeydown($event : KeyboardEvent) {
    if ($event.key == 'r' && !$event.ctrlKey &&
        !$event.altKey && !$event.shiftKey &&
        !$event.metaKey) {
      this.refresh();
    }
  }

  getItems() {
    return this.items_.filter(
      item => this.feeds_[item.feedId] && !this.feeds_[item.feedId].disabled);
  }
}
