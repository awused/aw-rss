import {Component,
        HostBinding,
        Input,
        OnChanges,
        OnDestroy,
        OnInit,
        SimpleChanges} from '@angular/core';
import {Feed,
        Item} from 'frontend/app/models/entities';
import {DataService} from 'frontend/app/services/data.service';
import {MutateService} from 'frontend/app/services/mutate.service';
import {Subject} from 'rxjs';
import {filter,
        takeUntil} from 'rxjs/operators';

import {Updates} from '../../../models/data';

@Component({
  selector: 'awrss-item',
  templateUrl: './item.component.html',
  styleUrls: ['./item.component.scss']
})
export class ItemComponent implements OnInit, OnDestroy, OnChanges {
  @Input()
  public item: Item;
  @Input()
  public showFeed: boolean;

  @HostBinding('class.read')
  get read() {
    return this.item.read;
  }

  @HostBinding('class.item-hover')
  get hover() {
    return this.itemHover;
  }

  public itemHover = true;
  public feed: Feed;
  public disabled = false;

  private readonly onDestroy$: Subject<void> = new Subject();

  constructor(
      private readonly mutateService: MutateService,
      private readonly dataService: DataService) {}


  toggleItemRead() {
    if (this.disabled) {
      return;
    }
    this.disabled = true;
    this.mutateService.markItemRead(this.item, !this.item.read)
        .subscribe(
            () => this.disabled = false,
            () => this.disabled = false);
  }

  handleItemMouseup(event: MouseEvent) {
    if (event.button === 1 && !this.item.read) {
      this.toggleItemRead();
    }
  }

  ngOnChanges(changes: SimpleChanges) {
    if ('item' in changes && !changes['item'].isFirstChange()) {
      this.handleChange();
    }
  }

  ngOnInit() {
    this.dataService
        .feedUpdates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(() => this.handleChange());
    this.handleChange();
  }

  private handleChange() {
    this.feed = this.dataService.getFeed(this.item.feedId);
  }

  ngOnDestroy() {
    this.onDestroy$.next();
  }
}
