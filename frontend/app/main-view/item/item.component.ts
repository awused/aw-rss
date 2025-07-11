import {Component,
        HostBinding,
        inject,
        Input,
        OnChanges,
        OnDestroy,
        OnInit,
        SimpleChanges} from '@angular/core';
import {
  Category,
  Feed,
  Item
} from 'frontend/app/models/entities';
import {DataService} from 'frontend/app/services/data.service';
import {MutateService} from 'frontend/app/services/mutate.service';
import {Subject} from 'rxjs';
import {takeUntil} from 'rxjs/operators';

@Component({
  selector: 'awrss-item',
  templateUrl: './item.component.html',
  styleUrls: ['./item.component.scss'],
  standalone: false
})
export class ItemComponent implements OnInit, OnDestroy, OnChanges {
  private readonly mutateService = inject(MutateService);
  private readonly dataService = inject(DataService);

  @Input()
  public item!: Item;
  @Input()
  public showFeed = false;
  @Input()
  public showCategory = false;

  @HostBinding('class.read')
  get read() {
    return this.item.read;
  }

  @HostBinding('class.item-hover')
  get hover() {
    return this.itemHover;
  }

  public itemHover = true;
  public feed: Feed|undefined;
  public category: Category|undefined;
  public disabled = false;

  private readonly onDestroy$: Subject<void> = new Subject();

  constructor() {}


  toggleItemRead() {
    if (this.disabled) {
      return;
    }
    this.disabled = true;
    this.mutateService.markItemRead(this.item, !this.item.read)
        .subscribe({
          next: () => this.disabled = false,
          error: () => this.disabled = false
        });
  }

  handleItemMouseup(event: MouseEvent) {
    if (event.button === 1 && !event.altKey && !this.item.read) {
      // Chrome will prevent the link from opening if it's replaced
      // optimistically in the same event loop.
      setTimeout(() => this.toggleItemRead());
    }
  }

  // Right click with no modifiers on the blank space
  handleContextMenu(event: MouseEvent) {
    if (!event.ctrlKey && !event.shiftKey) {
      event.preventDefault();
      this.toggleItemRead();
    }
  }

  // Shift click -> toggle read
  handleItemClick(event: MouseEvent) {
    if (event.button === 0 && event.shiftKey) {
      event.preventDefault();
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
    this.dataService
        .categoryUpdates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(() => this.handleChange());
    this.handleChange();
  }

  private handleChange() {
    this.feed = this.dataService.getFeed(this.item.feedId);

    this.category = undefined;
    if (this.feed.categoryId) {
      this.category = this.dataService.getCategory(this.feed.categoryId);
      if (this.category && this.category.disabled) {
        this.category = undefined;
      }
    }
  }

  ngOnDestroy() {
    this.onDestroy$.next();
    this.onDestroy$.complete();
  }
}
