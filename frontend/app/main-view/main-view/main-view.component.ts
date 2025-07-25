import {Component,
        inject,
        OnDestroy,
        OnInit} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute,
        ParamMap,
        Router} from '@angular/router';
import {ConfirmationDialogComponent} from 'frontend/app/admin/confirmation-dialog/confirmation-dialog.component';
import {EMPTY_FILTERED_DATA,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {
  Category,
  CATEGORY_NAME_REGEX,
  Feed,
  Item
} from 'frontend/app/models/entities';
import {Filters,
        PartialFilters} from 'frontend/app/models/filter';
import {FeedTitlePipe} from 'frontend/app/pipes/feed-title.pipe';
import {DataService} from 'frontend/app/services/data.service';
import {FuzzyFilterService} from 'frontend/app/services/fuzzy-filter.service';
import {MobileService} from 'frontend/app/services/mobile.service';
import {MutateService} from 'frontend/app/services/mutate.service';
import {ParamService} from 'frontend/app/services/param.service';
import {filter as fuzzyFilter,
        FilterOptions} from 'fuzzy/lib/fuzzy.js';
import {Observable,
        Subject} from 'rxjs';
import {
  filter,
  map,
  switchMap,
  takeUntil,
  tap
} from 'rxjs/operators';

const MAX_FUZZY_READ_ITEMS: number = 500;

@Component({
  selector: 'awrss-main-view',
  templateUrl: './main-view.component.html',
  styleUrls: ['./main-view.component.scss'],
  standalone: false
})
export class MainViewComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly dataService = inject(DataService);
  private readonly dialog = inject(MatDialog);
  private readonly feedTitlePipe = inject(FeedTitlePipe);
  private readonly paramService = inject(ParamService);
  private readonly mobileService = inject(MobileService);
  private readonly mutateService = inject(MutateService);
  private readonly fuzzyFilterService = inject(FuzzyFilterService);

  // @ViewChild('itemScroll')
  // public itemScroll: CdkVirtualScrollViewport;

  public fuzzyFilterString?: string;

  private readonly onDestroy$: Subject<void> = new Subject();
  private filteredData: FilteredData = EMPTY_FILTERED_DATA;

  private sortedItems: Item[] = [];
  private readonly fuzzyOptions: FilterOptions<Item> = {
    extract: (item: Item) => {
      if (this.feed) {
        return item.title;
      } else if (this.category) {
        return item.title + ' ' + this.dataService.getFeed(item.feedId).title;
      }

      const cid = this.dataService.getFeed(item.feedId).categoryId;
      const cat = cid !== undefined && this.dataService.getCategory(cid)?.title || '';
      return item.title + ' ' + this.dataService.getFeed(item.feedId).title + ' ' + cat;
    },
  };

  public category?: Category;
  public feed?: Feed;
  public enableMarkAsRead: boolean = false;
  public fuzzyItems: Item[] = [];
  public mobile: Observable<boolean>;

  public loadingMore = false;
  public hasRead = false;
  public hasAllRead = false;

  constructor() {
    this.mobile = this.mobileService.mobile();
    this.handleFuzzy(this.fuzzyFilterService.getFuzzyFilterString());
  }

  ngOnInit() {
    this.dataService.updates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((u: Updates) => {
          let changed;
          const oldItemLength = this.filteredData.items.length;
          [this.filteredData, changed] = this.filteredData.merge(u);

          if (u.refresh && !this.hasRead) {
            this.dataService.maybeCleanRead();
          }

          if (changed) {
            // Fast path
            if (!u.refresh &&
                oldItemLength === this.filteredData.items.length &&
                u.items.length < this.sortedItems.length &&
                this.tryMergeItems(u.items)) {
              // TODO -- remove https://github.com/angular/material2/pull/14639
              this.sortedItems = this.sortedItems.slice();
            } else {
              this.sortedItems = this.sortItems(this.filteredData.items);
            }

            if (this.fuzzyFilterString) {
              this.fuzzyItems =
                  fuzzyFilter(
                      this.fuzzyFilterString,
                      this.sortedItems,
                      this.fuzzyOptions)
                      .map(x => x.original);
            } else {
              this.fuzzyItems = this.sortedItems;
            }

            this.enableMarkAsRead = !!this.feed && !!this.fuzzyItems.find((i) => !i.read);

            if (this.category) {
              this.category = this.dataService.getCategory(this.category.id);

              if (!this.category || u.refresh && this.category.disabled) {
                this.router.navigate(['/'], {replaceUrl: true});
              }
            }
          }

          // These can turn false during backfill when feeds are added to categories or re-enabled.
          if (this.category &&
              this.hasAllRead &&
              !this.dataService.categoryAllRead(this.category)) {
            this.hasAllRead = false;
          } else if (
              !this.category && !this.feed && this.hasAllRead && !this.dataService.hasAllRead()) {
            this.hasAllRead = false;
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
            filter((f: Filters|undefined): f is Filters => Boolean(f)),
            tap(() => {
              this.filteredData = EMPTY_FILTERED_DATA;
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

    this.fuzzyFilterService.fuzzyFilterString()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((s: string) => this.handleFuzzy(s));
  }

  public getFeed(id: number): Feed {
    return this.dataService.getFeed(id);
  }

  // The first time the user clicks "show read" we start keeping read items through refreshes.
  public showRead() {
    const initialFilters = this.filteredData.filters;
    const newFilters =
        Object.assign({}, initialFilters, {unreadOnly: false});

    this.loadingMore = true;
    let next;
    if (this.category) {
      const c = this.category;
      next = (fd: FilteredData) => {
        if (!this.category || this.category.id !== c.id) {
          this.loadingMore = false;
          return;
        }

        if (initialFilters === this.filteredData.filters) {
          this.hasRead = true;
          this.hasAllRead = this.dataService.categoryAllRead(this.category);
          this.handleNewFilteredData(fd);
        }
      };

    } else if (this.feed) {
      const f = this.feed;

      next = (fd: FilteredData) => {
        if (!this.feed || this.feed.id !== f.id) {
          this.loadingMore = false;
          return;
        }

        if (initialFilters === this.filteredData.filters) {
          this.hasRead = true;
          this.hasAllRead = this.dataService.feedAllRead(this.feed);
          this.handleNewFilteredData(fd);
        }
      };
    } else {
      next = (fd: FilteredData) => {
        if (this.feed || this.category) {
          this.loadingMore = false;
          return;
        }

        if (initialFilters === this.filteredData.filters) {
          this.hasRead = true;
          this.hasAllRead = this.dataService.hasAllRead();
          this.handleNewFilteredData(fd);
        }
      };
    }

    this.dataService.dataForFilters(newFilters)
        .pipe(takeUntil(this.onDestroy$))
        .subscribe({
          next,
          error: () => this.loadingMore = false
        });
  }

  public showMoreRead() {
    this.loadingMore = true;
    let obs;
    let next;

    if (this.category) {
      const c = this.category;

      obs = this.dataService.fetchMoreReadForCategory(this.category.id);
      next = () => {
        this.loadingMore = false;
        if (!this.category || this.category.id !== c.id) {
          return;
        }

        this.hasAllRead = this.dataService.categoryAllRead(this.category);
      };
    } else if (this.feed) {
      const f = this.feed;

      obs = this.dataService.fetchMoreReadForFeed(this.feed.id);
      next = () => {
        this.loadingMore = false;
        if (!this.feed || this.feed.id !== f.id) {
          return;
        }

        this.hasAllRead = this.dataService.feedAllRead(this.feed);
      };
    } else {
      obs = this.dataService.fetchMoreReadForAll();
      next = () => {
        this.loadingMore = false;
        if (this.category || this.feed) {
          return;
        }

        this.hasAllRead = this.dataService.hasAllRead();
      };
    }

    obs.pipe(takeUntil(this.onDestroy$))
        .subscribe({
          next,
          error: () => this.loadingMore = false
        });
  }

  public markAsRead() {
    if (!this.enableMarkAsRead || !this.feed) {
      return;
    }

    const feed = this.feed;

    if (!this.fuzzyFilterString) {
      const maxItemId = this.filteredData.items[this.filteredData.items.length - 1].id;

      this.dialog.open<any, any, boolean>(ConfirmationDialogComponent, {
                   data: {
                     title: 'Confirm Action',
                     text: [
                       `Mark all items read for
                     ${this.feedTitlePipe.transform(feed)}?`,
                       `This action is irreversible.`
                     ],
                     dangerous: true,
                   }
                 })
          .beforeClosed()
          .subscribe((result) => {
            if (result) {
              this.mutateService.markFeedAsRead(feed.id, maxItemId);
            }
          });

      return;
    }


    let itemIds = this.fuzzyItems.filter((it) => !it.read).map((it) => it.id);
    if (itemIds.length == 0) {
      // This shouldn't be possible
      return;
    }

    let msg =
        `Mark the ${itemIds.length} visible unread item${itemIds.length > 1 ? 's' : ''} as read?`;

    if (itemIds.length > MAX_FUZZY_READ_ITEMS) {
      msg = `Mark the bottom ${
          MAX_FUZZY_READ_ITEMS} of the ${itemIds.length} visible unread items as read?`;
      itemIds = itemIds.slice(-MAX_FUZZY_READ_ITEMS);
    }

    this.dialog.open<any, any, boolean>(ConfirmationDialogComponent, {
                 data: {
                   title: 'Confirm Action',
                   text: [
                     msg,
                     `This action is irreversible.`
                   ],
                   dangerous: true,
                 }
               })
        .beforeClosed()
        .subscribe((result) => {
          if (result) {
            this.mutateService.markBulkItemsAsRead(feed.id, itemIds);
          }
        });
  }

  private handleFuzzy(filterString: string) {
    this.fuzzyFilterString = filterString;
    if (this.fuzzyFilterString) {
      this.fuzzyItems =
          fuzzyFilter(
              this.fuzzyFilterString, this.sortedItems, this.fuzzyOptions)
              .map(x => x.original);
    } else {
      this.fuzzyItems = this.sortedItems;
    }

    this.enableMarkAsRead = !!this.feed && !!this.fuzzyItems.find((i) => !i.read);
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

    if (this.fuzzyFilterString) {
      this.fuzzyItems =
          fuzzyFilter(
              this.fuzzyFilterString, this.sortedItems, this.fuzzyOptions)
              .map(x => x.original);
    } else {
      this.fuzzyItems = this.sortedItems;
    }

    this.enableMarkAsRead = !!this.feed && !!this.fuzzyItems.find((i) => !i.read);
  }

  private paramsToFilters(p: ParamMap): Filters|undefined {
    const f: PartialFilters = {
      validOnly: true,
      unreadOnly: true,
      isMainView: true,
      keepUnlessRefresh: true,
    };

    const fid = p.get('feedId');
    if (fid) {
      if (!/^\d+$/.test(fid)) {
        // The nav service will log and redirect users
        return;
      }
      f.feedId = parseInt(fid, 10);
    }

    const cname = p.get('categoryName');
    if (cname) {
      if (!CATEGORY_NAME_REGEX.test(cname)) {
        // The nav service will log and redirect users
        return;
      }
      f.categoryName = cname;
    }

    return f as Filters;
  }

  // A faster merge method when the set of items hasn't changed and the update
  // contains a smaller list of items.
  private tryMergeItems(items: ReadonlyArray<Item>): boolean {
    let i = 0;
    const sorted = this.sortItems(items);

    outer: for (let j = 0; j < sorted.length; j++) {
      const nit = sorted[j];

      while (i < this.sortedItems.length) {
        const sit = this.sortedItems[i];
        const cmp = this.compareItems(nit, sit);
        if (cmp < 0) {
          console.log(`Failed to merge updates for item:`, nit);
          return false;
        } else if (cmp > 0) {
          i++;
          continue;
        }

        if (nit.commitTimestamp >= sit.commitTimestamp) {
          this.sortedItems[i] = nit;
        }
        i++;
        continue outer;
      }

      console.log(`Failed to merge updates for item:`, nit);
      return false;
    }

    return true;
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
