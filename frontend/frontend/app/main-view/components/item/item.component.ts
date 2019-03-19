import {Component,
        Input,
        OnChanges,
        OnDestroy,
        OnInit,
        SimpleChanges} from '@angular/core';
import {Feed,
        Item} from 'frontend/app/models/entities';
import {Updates} from 'frontend/app/models/updates';
import {DataService} from 'frontend/app/services/data.service';
import {MutateService} from 'frontend/app/services/mutate.service';
import {Subject} from 'rxjs';
import {filter,
        takeUntil} from 'rxjs/operators';

@Component({
  selector: 'awrss-item',
  templateUrl: './item.component.html',
  styleUrls: ['./item.component.scss']
})
export class ItemComponent implements OnInit, OnDestroy, OnChanges {
  @Input()
  public item: Item;

  public feed: Feed;
  public disabled: boolean = false;

  private readonly onDestroy$: Subject<void> = new Subject();

  constructor(
      private readonly mutateService: MutateService,
      private readonly dataService: DataService) {}


  markItemRead(read: boolean) {
    if (this.disabled) {
      return;
    }
    this.disabled = true;
    this.mutateService.markItemRead(this.item, read)
        .subscribe(
            () => this.disabled = false,
            () => this.disabled = false);
  }

  handleItemMouseup(event: MouseEvent) {
    if (event.button == 1 && !this.item.read) {
      this.markItemRead(true);
    }
  }

  ngOnChanges(changes: SimpleChanges) {
    if ('item' in changes && !changes['item'].isFirstChange()) {
      this.handleChange();
    }
  }

  ngOnInit() {
    this.dataService
        .updates()
        .pipe(
            takeUntil(this.onDestroy$),
            filter((u: Updates) => u.data.feeds.length > 0))
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
