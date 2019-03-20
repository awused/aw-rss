import {Component,
        OnDestroy,
        OnInit} from '@angular/core';
import {ActivatedRoute,
        ParamMap} from '@angular/router';
import {Data,
        EmptyFilteredData,
        Entity,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {Feed,
        Item} from 'frontend/app/models/entities';
import {EmptyFilters,
        Filters} from 'frontend/app/models/filter';
import {DataService} from 'frontend/app/services/data.service';
import {Subject} from 'rxjs';
import {
  map,
  switchMap,
  takeUntil,
  tap
} from 'rxjs/operators';

@Component({
  selector: 'awrss-main-view',
  templateUrl: './main-view.component.html',
  styleUrls: ['./main-view.component.scss']
})
export class MainViewComponent implements OnInit, OnDestroy {
  private readonly onDestroy$: Subject<void> = new Subject();
  private filteredData: FilteredData = EmptyFilteredData;
  public sortedItems: Item[] = [];
  public showFeedOnItems = true;

  constructor(
      private readonly route: ActivatedRoute,
      private readonly dataService: DataService) {}

  ngOnInit() {
    this.dataService.updates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((u: Updates) => {
          let changed;
          const oldItemLength = this.filteredData.items.length;
          [this.filteredData, changed] = this.filteredData.merge(u);
          if (changed) {
            // Fast path
            if (!u.refresh &&
                oldItemLength === this.filteredData.items.length &&
                u.items.length < this.sortedItems.length) {
              this.mergeItems(u.items);
            } else {
              this.sortedItems = this.sortItems(this.filteredData.items);
            }
          }
        });

    this.route.paramMap
        .pipe(
            takeUntil(this.onDestroy$),
            map((p: ParamMap) => this.paramsToFilters(p)),
            tap(() => this.filteredData = EmptyFilteredData),
            switchMap((f: Filters) => this.dataService.dataForFilters(f)))
        .subscribe((fd: FilteredData) => {
          this.showFeedOnItems =
              !fd.filters.feedIds || fd.filters.feedIds.length === 1;

          // TODO -- if the category is disabed kick the user to /, here
          // TODO -- subscribe to the category in DataService, if it does exist
          this.filteredData = fd;
          this.sortedItems = this.sortItems(this.filteredData.items);
        });
  }

  public getFeed(id: number): Feed {
    return this.dataService.getFeed(id);
  }

  private paramsToFilters(p: ParamMap): Filters {
    // TODO -- translate route params into filters
    return {
      validOnly: true,
      unreadOnly: true,
      keepExistingUnlessRefresh: true,
      excludeCategories: true,
    };
  }

  // A faster merge method when the set of items hasn't changed and the update
  // contains a smaller list of items.
  private mergeItems(items: ReadonlyArray<Item>): void {
    let i = 0;
    const sorted = this.sortItems(items);

    sorted.forEach((nit: Item) => {
      for (; i < this.sortedItems.length; i++) {
        const sit = this.sortedItems[i];
        const cmp = this.compareItems(nit, sit);
        if (cmp < 0) {
          return;
        } else if (cmp > 0) {
          continue;
        }

        if (nit.commitTimestamp >= sit.commitTimestamp) {
          this.sortedItems[i] = nit;
        }
      }
    });
  }

  private sortItems(items: ReadonlyArray<Item>): Item[] {
    return items.slice().sort(this.compareItems);
  }

  private compareItems(a: Item, b: Item): number {
    if (a.timestamp === b.timestamp) {
      if (a.id === b.id) {
        return 0;
      }
      return a.id > b.id ? -1 : 1;
    }
    return a.timestamp > b.timestamp ? -1 : 1;
  }

  ngOnDestroy() {
    this.onDestroy$.next();
  }
}
