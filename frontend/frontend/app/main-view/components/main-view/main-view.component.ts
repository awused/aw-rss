import {
  ChangeDetectorRef,
  Component,
  OnDestroy,
  OnInit
} from '@angular/core';
import {ActivatedRoute,
        ParamMap,
        Router} from '@angular/router';
import {Data,
        EmptyFilteredData,
        Entity,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {
  Category,
  Feed,
  Item
} from 'frontend/app/models/entities';
import {EmptyFilters,
        Filters,
        PartialFilters} from 'frontend/app/models/filter';
import {DataService} from 'frontend/app/services/data.service';
import {ErrorService} from 'frontend/app/services/error.service';
import {Subject} from 'rxjs';
import {
  filter,
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
  public category?: Category;
  public feed?: Feed;
  public sortedItems: Item[] = [];

  constructor(
      private readonly route: ActivatedRoute,
      private readonly router: Router,
      private readonly dataService: DataService,
      private readonly errorService: ErrorService) {}

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
              // TODO -- remove https://github.com/angular/material2/pull/14639
              this.sortedItems = this.sortedItems.slice();
            } else {
              this.sortedItems = this.sortItems(this.filteredData.items);
            }
          }

          if (u.refresh && this.category && this.category.disabled) {
            this.router.navigate(['/'], {replaceUrl: true});
          }
        });

    this.dataService.feedUpdates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(() => {
          if (this.feed) {
            this.feed = this.dataService.getFeed(this.feed.id);
          }
        });

    this.dataService.categoryUpdates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(() => {
          if (this.category) {
            this.category = this.dataService.getCategory(this.category.id);
          }
        });

    this.route.paramMap
        .pipe(
            takeUntil(this.onDestroy$),
            map((p: ParamMap) => this.paramsToFilters(p)),
            filter((f?: Filters) => !!f),
            tap(() => this.filteredData = EmptyFilteredData),
            switchMap((f: Filters) => this.dataService.dataForFilters(f)))
        .subscribe((fd: FilteredData) => this.handleNewFilteredData(fd));
  }

  public getFeed(id: number): Feed {
    return this.dataService.getFeed(id);
  }

  private handleNewFilteredData(fd: FilteredData) {
    this.category = undefined;
    this.feed = undefined;

    if (fd.filters.feedId !== undefined) {
      if (fd.feeds.length !== 1) {
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }

      this.feed = fd.feeds[0];
    }

    if (fd.filters.categoryName !== undefined) {
      if (fd.categories.length !== 1) {
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }

      this.category = fd.categories[0];
    }

    // TODO -- if the category is disabed kick the user to /, here
    // TODO -- subscribe to the category in DataService, if it does exist
    this.filteredData = fd;
    this.sortedItems = this.sortItems(this.filteredData.items);
  }

  private paramsToFilters(p: ParamMap): Filters|undefined {
    const f: PartialFilters = {
      validOnly: true,
      unreadOnly: true,
      isMainView: true,
      keepUnlessRefresh: true,
    };

    if (p.has('feedId')) {
      const fid = p.get('feedId');
      if (!/^\d+$/.test(fid)) {
        this.errorService.showError('Invalid feed ID: ' + fid);
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }
      f.feedId = parseInt(fid);
    }

    if (p.has('categoryName')) {
      const cname = p.get('categoryName');
      if (!/^[a-z][a-z-]+[a-z]$/.test(cname)) {
        this.errorService.showError('Invalid category name: ' + cname);
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }
      f.categoryName = cname;
    }

    // TODO -- translate route params into filters
    return <Filters>f;
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
