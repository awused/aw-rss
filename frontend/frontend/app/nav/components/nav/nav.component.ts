import {
  CdkDragDrop,
  CdkDragRelease,
  CdkDragStart
} from '@angular/cdk/drag-drop';
import {Component,
        EventEmitter,
        Input,
        OnInit,
        Output} from '@angular/core';
import {ActivatedRoute,
        ParamMap,
        Router} from '@angular/router';
import {EmptyFilteredData,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {Category,
        Feed,
        Item} from 'frontend/app/models/entities';
import {DataService} from 'frontend/app/services/data.service';
import {ErrorService} from 'frontend/app/services/error.service';
import {ParamService} from 'frontend/app/services/param.service';
import {RefreshService} from 'frontend/app/services/refresh.service';


export class FeedData {
  constructor(
      public feed: Feed,
      public unread: Set<number> = new Set(),
      public lastItem?: Date,
      public failingSinceString?: string,
      public lastItemString?: string) {}
}

class CategoryData {
  constructor(
      public category: Category,
      public unread: number = 0) {}
}

interface NavCategory {
  cData: CategoryData;
  fData: FeedData[];
}

@Component({
  selector: 'awrss-nav',
  templateUrl: './nav.component.html',
  styleUrls: ['./nav.component.scss']
})
export class NavComponent {
  // This controller will never be destroyed
  constructor(
      private readonly route: ActivatedRoute,
      private readonly router: Router,
      private readonly refreshService: RefreshService,
      private readonly dataService: DataService,
      private readonly errorService: ErrorService,
      private readonly paramService: ParamService) {
    this.dataService.updates().subscribe(
        (u: Updates) => this.handleUpdates(u));

    this.dataService.dataForFilters({
                      validOnly: true,
                      unreadOnly: true,
                    })
        .subscribe((fd: FilteredData) => {
          // TODO -- get initial newest item times
          this.handleUpdates(fd);

          this.paramService.mainViewParams()
              .subscribe((p: ParamMap) => this.handleParams(p));
          this.route.queryParamMap
              .subscribe((q: ParamMap) => {
                // Dammit Angular https://github.com/angular/angular/issues/12664
                if (q.has('all')) {
                  this.showAll = q.get('all') === 'true';
                }
              });
        });
  }
  @Input()
  public showHeader: boolean;
  @Output()
  public unreadCount = new EventEmitter<number>();
  @Output()
  public pageTitle = new EventEmitter<string>();

  public selectedCategoryName?: string;
  public selectedFeed?: number;
  public navCategories: NavCategory[];
  public uncategorizedFeeds: FeedData[];
  public dragging = false;
  public draggingCategory?: number;
  public hideCategory?: number;
  public dropTarget: CategoryData|string|undefined;
  public showAll = false;
  public expanded: {[x: number]: boolean} = {};

  private unreadByFeed: Map<number, FeedData> = new Map();
  private unreadByCategory: Map<number, CategoryData> = new Map();
  private mainUnread = 0;
  private categoriesByName: Map<string, number> = new Map();

  private static feedDataComparator(a: FeedData, b: FeedData): number {
    if (a.unread.size && !b.unread.size) {
      return -1;
    }
    if (!a.unread.size && b.unread.size) {
      return 1;
    }

    const aTitle = a.feed.userTitle || a.feed.title;
    const bTitle = b.feed.userTitle || b.feed.title;
    return aTitle.toLowerCase() > bTitle.toLowerCase() ? 1 : -1;
  }

  public shouldHideCategory(c: Category): boolean {
    if (this.showAll) {
      return false;
    }

    if (!c.hiddenNav || c.name === this.selectedCategoryName) {
      return false;
    }

    const fd = this.unreadByFeed.get(this.selectedFeed);
    if (fd && fd.feed.categoryId === c.id) {
      return false;
    }

    return true;
  }

  public isRefreshing(): boolean {
    return this.refreshService.isRefreshing();
  }

  public refresh() {
    this.refreshService.startRefresh();
  }

  public dragStarted(event: CdkDragStart<FeedData|CategoryData>, x: any) {
    const data = event.source.data;
    this.dragging = true;
    this.dropTarget = undefined;
    if (data instanceof FeedData) {
      this.hideCategory = data.feed.categoryId;
    }

    if (data instanceof CategoryData) {
      this.draggingCategory = data.category.id;
    }
  }

  public dragDropped(event: CdkDragDrop<CategoryData|void, FeedData|CategoryData>) {
    this.dragging = false;
    this.hideCategory = undefined;
    this.draggingCategory = undefined;
    if (!event.isPointerOverContainer) {
      return;
    }

    // this.dropTarget is a really hacky workaround for removing feeds from categories
    // it only works with a real mouse
    // TODO -- Implement a better workaround for Material's awful drag and drop on mobile
    const target = this.dropTarget || event.container.data;
    const targetCategory = target instanceof CategoryData ? target : undefined;


    // TODO -- Actually do these things
    const data = event.item.data;
    if (data instanceof CategoryData) {
      if (targetCategory && data.category.id !== targetCategory.category.id) {
        console.log(`would sort ${data.category.id} after ${targetCategory.category.id}`);
      } else {
        console.log(`would sort ${data.category.id} at the end`);
      }
    }

    if (data instanceof FeedData) {
      if (!targetCategory && data.feed.categoryId !== undefined) {
        console.log(`would remove ${data.feed.id} from ${data.feed.categoryId}`);
      }

      if (targetCategory && data.feed.categoryId !== targetCategory.category.id) {
        console.log(`would add ${data.feed.id} to ${targetCategory.category.id}`);
      }
    }
  }

  private handleParams(p: ParamMap|void) {
    this.selectedCategoryName = undefined;
    this.selectedFeed = undefined;
    if (!p) {
      return;
    }

    if (p.has('feedId')) {
      const fid = p.get('feedId');
      if (!/^\d+$/.test(fid)) {
        this.errorService.showError('Invalid feed ID: ' + fid);
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }
      this.selectedFeed = parseInt(fid, 10);
      // Even if a feed is disabled, this will be populated
      if (!this.unreadByFeed.has(this.selectedFeed)) {
        this.errorService.showError('Invalid feed ID: ' + fid);
        this.router.navigate(['/'], {replaceUrl: true});
      }
    }

    if (p.has('categoryName')) {
      const cname = p.get('categoryName');
      if (!/^[a-z][a-z-]+[a-z]$/.test(cname)) {
        this.errorService.showError('Invalid category name: ' + cname);
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }
      this.selectedCategoryName = cname;

      const cd = this.unreadByCategory.get(this.categoriesByName.get(cname));
      // Redirect for disabled categories or categories that have been renamed
      if (!cd || cd.category.disabled || cd.category.name !== cname) {
        this.errorService.showError('Invalid category name: ' + cname);
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }
    }
    this.emit();
  }

  private handleUpdates(u: Updates|FilteredData) {
    // Sorting will also recalculate unreadByCategory and mainUnread
    let mustSort = false;
    // Sometimes it's easier to just recalculate all categories
    let recalculate = false;

    u.categories.forEach((c: Category) => {
      if (c.disabled) {
        const removed = this.unreadByCategory.delete(c.id);
        if (removed) {
          mustSort = true;
          recalculate = true;
        }
        return;
      }
      this.categoriesByName.set(c.name, c.id);

      if (!this.unreadByCategory.has(c.id)) {
        if (!c.disabled) {
          this.unreadByCategory.set(c.id, new CategoryData(c));
          recalculate = true;
          mustSort = true;
        }
      } else {
        const cd = this.unreadByCategory.get(c.id);
        const oldc = cd.category;
        cd.category = c;

        if (this.isHidden(oldc) !== this.isHidden(c)) {
          if (this.isHidden(c)) {
            this.mainUnread -= cd.unread;
          } else {
            this.mainUnread += cd.unread;
          }
        }

        if (oldc.sortOrder !== c.sortOrder) {
          mustSort = true;
        }
      }
    });

    u.feeds.forEach((f: Feed) => {
      let fd = this.unreadByFeed.get(f.id);
      if (!fd) {
        mustSort = true;
        fd = new FeedData(f);
        this.unreadByFeed.set(f.id, fd);
        const lastItem = this.dataService.getInitialTimestampForFeed(f.id);
        if (lastItem) {
          fd.lastItem = new Date(lastItem);
          fd.lastItemString = this.timeAgoString(fd.lastItem);
        }
      }

      const oldf = fd.feed;
      fd.feed = f;

      if (f.failingSince) {
        fd.failingSinceString = this.timeAgoString(new Date(f.failingSince));
      }

      if (f.disabled !== oldf.disabled) {
        recalculate = true;
      }

      if (f.categoryId !== oldf.categoryId) {
        mustSort = true;
        recalculate = true;
      }

      if (f.title !== oldf.title ||
          f.userTitle !== oldf.userTitle) {
        mustSort = true;
      }
    });

    u.items.forEach((it: Item) => {
      if (!this.unreadByFeed.has(it.feedId)) {
        // Possible when there are updates for a disabled feed
        return;
      }

      const fd = this.unreadByFeed.get(it.feedId);
      if (!fd.lastItem || new Date(it.timestamp) > fd.lastItem) {
        fd.lastItem = new Date(it.timestamp);
        fd.lastItemString = this.timeAgoString(fd.lastItem);
      }

      const change = it.read ? -1 : 1;
      if (it.read) {
        const removed = fd.unread.delete(it.id);
        if (!removed) {
          // Item is read, but we weren't counting it anyway
          return;
        }
      } else if (!fd.unread.has(it.id)) {
        fd.unread.add(it.id);
      } else {
        // Item is unread but we are already counting it
        return;
      }

      if (fd.feed.disabled) {
        return;
      }
      const cd = this.unreadByCategory.get(fd.feed.categoryId);
      if (cd) {
        cd.unread += change;
        if (!this.isHidden(cd.category)) {
          this.mainUnread += change;
        }
      } else {
        this.mainUnread += change;
      }
    });

    if (recalculate) {
      this.recalculateUnread();
    }

    if (u instanceof Updates && u.refresh) {
      mustSort = true;
      this.refreshTimeStrings();
    }

    if (mustSort) {
      this.sortNav();
    }
    this.emit();
  }

  private sortNav() {
    this.navCategories = [];
    this.uncategorizedFeeds = [];

    const ncm: Map<number, NavCategory> = new Map();

    this.unreadByCategory.forEach((cd: CategoryData) => {
      ncm.set(cd.category.id, {cData: cd, fData: []});
    });
    this.unreadByFeed.forEach((fd: FeedData) => {
      if (ncm.has(fd.feed.categoryId)) {
        ncm.get(fd.feed.categoryId).fData.push(fd);
      } else {
        this.uncategorizedFeeds.push(fd);
      }
    });

    this.uncategorizedFeeds.sort(NavComponent.feedDataComparator);

    ncm.forEach((nc: NavCategory) => {
      nc.fData.sort(NavComponent.feedDataComparator);
      this.navCategories.push(nc);
    });

    this.navCategories.sort((a, b) => {
      if (a.cData.category.sortOrder !== undefined) {
        if (b.cData.category.sortOrder === undefined) {
          return -1;
        }

        return a.cData.category.sortOrder - b.cData.category.sortOrder;
      }
      return a.cData.category.id - b.cData.category.id;
    });
  }

  private refreshTimeStrings() {
    this.unreadByFeed.forEach((fd: FeedData) => {
      if (fd.lastItem) {
        fd.lastItemString = this.timeAgoString(fd.lastItem);
      }
      if (fd.feed.failingSince) {
        fd.failingSinceString = this.timeAgoString(new Date(fd.feed.failingSince));
      }
    });
  }

  private timeAgoString(ts: Date): string {
    // Only care about days, hours, and minutes
    const intervalM = (new Date().valueOf() - ts.valueOf()) / (1000 * 60);

    if (intervalM > 60 * 24 * 2) {
      return Math.trunc(intervalM / (60 * 24)) + 'd';
    } else if (intervalM > 120) {
      return Math.trunc(intervalM / 60) + 'h';
    }
    return Math.trunc(intervalM) + 'm';
  }

  private emit() {
    const cd = this.unreadByCategory.get(
        this.categoriesByName.get(this.selectedCategoryName));
    if (cd) {
      this.pageTitle.emit(cd.category.title);
      this.unreadCount.emit(cd.unread);
      return;
    }

    const fd = this.unreadByFeed.get(this.selectedFeed);
    if (fd) {
      this.pageTitle.emit(fd.feed.title);
      this.unreadCount.emit(fd.unread.size);
      return;
    }

    this.pageTitle.emit('Aw-RSS');
    this.unreadCount.emit(this.mainUnread);
  }

  private recalculateUnread() {
    this.mainUnread = 0;
    this.unreadByCategory.forEach((cd: CategoryData) => cd.unread = 0);

    this.unreadByFeed.forEach((fd: FeedData) => {
      if (fd.feed.disabled) {
        return;
      }

      const cd = this.unreadByCategory.get(fd.feed.categoryId);
      if (cd) {
        cd.unread += fd.unread.size;
      }
      if (!cd || !this.isHidden(cd.category)) {
        this.mainUnread += fd.unread.size;
      }
    });
  }

  private isHidden(c: Category): boolean {
    return c.hiddenMain || c.hiddenNav;
  }
}
