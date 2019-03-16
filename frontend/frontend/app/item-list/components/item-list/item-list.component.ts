import {Component,
        OnDestroy,
        OnInit} from '@angular/core';
import {ActivatedRoute,
        ParamMap} from '@angular/router';
import {Data,
        Entity,
        FilteredData} from 'frontend/app/models/data';
import {Feed} from 'frontend/app/models/feed';
import {EmptyFilters,
        Filters} from 'frontend/app/models/filter';
import {Item} from 'frontend/app/models/item';
import {Updates} from 'frontend/app/models/updates';
import {DataService} from 'frontend/app/services/data.service';
import {Subject} from 'rxjs';
import {
  map,
  switchMap,
  takeUntil
} from 'rxjs/operators';

@Component({
  selector: 'awrss-item-list',
  templateUrl: './item-list.component.html',
  styleUrls: ['./item-list.component.scss']
})
export class ItemListComponent implements OnInit, OnDestroy {
  private readonly onDestroy$: Subject<void> = new Subject();
  private filteredData: FilteredData = new FilteredData(new Data(), EmptyFilters);
  public sortedItems: Item[] = [];

  constructor(
      private readonly route: ActivatedRoute,
      private readonly dataService: DataService) {}

  ngOnInit() {
    this.dataService.updates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((u: Updates) => {
          let changed;
          [this.filteredData, changed] = this.filteredData.merge(u);
          if (changed) {
            this.sortItems();
          }
        });

    this.route.paramMap
        .pipe(
            takeUntil(this.onDestroy$),
            map((p: ParamMap) => this.paramsToFilters(p)),
            switchMap((f: Filters) => this.dataService.dataForFilters(f)))
        .subscribe((fd: FilteredData) => {
          this.filteredData = fd;
          this.sortItems();
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

  public idFn(e: Entity): number {
    return e.id;
  }

  private sortItems() {
    this.sortedItems =
        this.filteredData.items.slice()
            .sort((a, b) => {
              if (a.timestamp === b.timestamp) {
                return 0;
              }
              return a.timestamp > b.timestamp ? -1 : 1;
            });
  }

  ngOnDestroy() {
    this.onDestroy$.next();
  }
}
