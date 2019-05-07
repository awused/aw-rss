import {
  ChangeDetectorRef,
  Component,
  Input,
  OnDestroy,
  OnInit,
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
  CATEGORY_NAME_REGEX,
  Feed,
  Item
} from 'frontend/app/models/entities';
import {EmptyFilters,
        Filters,
        PartialFilters} from 'frontend/app/models/filter';
import {DataService} from 'frontend/app/services/data.service';
import {ErrorService} from 'frontend/app/services/error.service';
import {MobileService} from 'frontend/app/services/mobile.service';
import {ParamService} from 'frontend/app/services/param.service';
import {Observable,
        Subject} from 'rxjs';
import {
  filter,
  last,
  map,
  switchMap,
  take,
  takeUntil,
  tap
} from 'rxjs/operators';

@Component({
  selector: 'awrss-main-view',
  templateUrl: './main-view.component.html',
  styleUrls: ['./main-view.component.scss']
})
export class MainViewComponent implements OnInit, OnDestroy {
  // @ViewChild('itemScroll')
  // public itemScroll: CdkVirtualScrollViewport;

  private readonly onDestroy$: Subject<void> = new Subject();
  private filteredData: FilteredData = EmptyFilteredData;
  public category?: Category;
  public feed?: Feed;
  public maxItemId?: number;
  public sortedItems: Item[] = [];
  public mobile: Observable<boolean>;

  public loadingMore = false;
  public hasRead = false;
  public hasAllRead = false;

  constructor(
      private readonly route: ActivatedRoute,
      private readonly router: Router,
      private readonly dataService: DataService,
      private readonly errorService: ErrorService,
      private readonly paramService: ParamService,
      private readonly mobileService: MobileService) {}

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

            if (this.feed && this.filteredData.items.length) {
              this.maxItemId =
                  this.filteredData
                      .items[this.filteredData.items.length - 1]
                      .id;
              // Could do better if counting unread was moved out of NavComponent.
              // But other work is O(n) anyway.
              if (!this.filteredData.items.find((i) => !i.read)) {
                this.maxItemId = undefined;
              }
            }

            if (this.category) {
              this.category = this.dataService.getCategory(this.category.id);

              if (u.refresh && this.category.disabled) {
                this.router.navigate(['/'], {replaceUrl: true});
              }
            }
          }
        });

    this.dataService.feedUpdates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(() => {
          if (this.feed) {
            this.feed = this.dataService.getFeed(this.feed.id);
          }
        });

    // TODO -- this needs to combine params and query params
    // and debounce them
    this.route.paramMap
        .pipe(
            takeUntil(this.onDestroy$),
            map((p: ParamMap) => this.paramsToFilters(p)),
            filter((f?: Filters) => !!f),
            tap(() => {
              this.filteredData = EmptyFilteredData;
              this.loadingMore = false;
              this.hasRead = false;
              this.hasAllRead = false;
            }),
            switchMap((f: Filters) => this.dataService.dataForFilters(f)),
            // At worst, this snapshot will be a page the user is navigating to
            tap(() =>
                    this.paramService.pushMainViewParams(
                        this.route.snapshot.paramMap)),
            // The first takeUntil will prevent unnecessary data requests
            // This one will prevent mangling state strangely
            takeUntil(this.onDestroy$))
        .subscribe((fd: FilteredData) => this.handleNewFilteredData(fd));

    this.mobile = this.mobileService.mobile();

    // TODO -- maybe control if show-more is visible to reduce jank
    /*
    this.itemScroll.renderedRangeStream
      .pipe(takeUntil(this.onDestroy$))
      .subscribe((lr: ListRange) => {
      });
     */
  }

  public getFeed(id: number): Feed {
    return this.dataService.getFeed(id);
  }

  public showRead() {
    if (!this.feed) {
      // TODO
      return;
    }

    const initialFilters = this.filteredData.filters;
    const newFilters =
        Object.assign({}, initialFilters, {unreadOnly: false});

    this.loadingMore = true;
    this.dataService.dataForFilters(newFilters)
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((fd: FilteredData) => {
          if (initialFilters === this.filteredData.filters) {
            this.hasRead = true;
            this.hasAllRead = this.dataService.hasAllRead(this.feed);
            this.handleNewFilteredData(fd);
          }
        }, () => this.loadingMore = false);
  }

  public showMoreRead() {
    if (!this.feed) {
      // TODO
      return;
    }


    this.loadingMore = true;
    this.dataService.fetchMoreReadForFeed(this.feed.id)
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(() => {
          this.loadingMore = false;
          this.hasAllRead = this.dataService.hasAllRead(this.feed);
        }, () => this.loadingMore = false);
  }

  private handleNewFilteredData(fd: FilteredData) {
    this.category = undefined;
    this.feed = undefined;
    this.loadingMore = false;
    if (fd.filters.unreadOnly) {
      this.hasRead = false;
      this.hasAllRead = false;
    }

    if (fd.filters.feedId !== undefined) {
      if (fd.feeds.length !== 1) {
        // Nav component will redirect
        return;
      }

      this.feed = fd.feeds[0];
    }

    if (fd.filters.categoryName !== undefined) {
      if (fd.categories.length !== 1) {
        // Nav component will redirect
        return;
      }

      this.category = fd.categories[0];
    }

    // TODO -- if the category is disabled kick the user to /, here
    // TODO -- subscribe to the category in DataService, if it does exist
    this.filteredData = fd;
    this.sortedItems = this.sortItems(this.filteredData.items);

    if (this.feed && fd.items.length) {
      this.maxItemId =
          fd.items[this.filteredData.items.length - 1].id;
      // Could do better if counting unread was moved out of NavComponent.
      // But other work is O(n) anyway.
      if (!fd.items.find((i) => !i.read)) {
        this.maxItemId = undefined;
      }
    }
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
        // The nav service will log and redirect users
        return;
      }
      f.feedId = parseInt(fid, 10);
    }

    if (p.has('categoryName')) {
      const cname = p.get('categoryName');
      if (!CATEGORY_NAME_REGEX.test(cname)) {
        // The nav service will log and redirect users
        return;
      }
      f.categoryName = cname;
    }

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
    this.onDestroy$.complete();
    this.paramService.pushMainViewParams();
  }
}
