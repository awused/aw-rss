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


interface FeedData {
  feed?: Feed;
  unread: Set<number>;
  lastItem?: Date;
}

interface CategoryData {
  category: Category;
  unread: number;
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
  @Input()
  public showHeader: boolean;
  @Output()
  public unreadCount = new EventEmitter<number>();
  @Output()
  public pageTitle = new EventEmitter<string>();

  public selectedCategoryName: string;
  public selectedFeed: number;
  public navCategories: NavCategory[];
  public uncategorizedFeeds: FeedData[];

  private unreadByFeed: Map<number, FeedData> = new Map();
  private unreadByCategory: Map<number, CategoryData> = new Map();
  private mainUnread: number = 0;
  private categoriesByName: Map<string, number> = new Map();

  private filteredData: FilteredData = EmptyFilteredData;

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
        });
  }

  public isRefreshing(): boolean {
    return this.refreshService.isRefreshing();
  }

  public refresh() {
    this.refreshService.startRefresh();
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

      if (!this.categoriesByName.has(cname)) {
        this.errorService.showError('Invalid category name: ' + cname);
        this.router.navigate(['/'], {replaceUrl: true});
        return;
      }

      const cd = this.unreadByCategory.get(this.categoriesByName.get(cname));
      // Redirect for disabled categories or categories that have been renamed
      if (!cd || cd.category.disabled || cd.category.name !== cname) {
        this.errorService.showError('Invalid category name: ' + cname);
        this.router.navigate(['/'], {replaceUrl: true});
      }
    }
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
          this.unreadByCategory.set(c.id, {
            category: c,
            unread: 0
          });
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
      if (!this.unreadByFeed.has(f.id)) {
        mustSort = true;
        this.unreadByFeed.set(f.id, {feed: f, unread: new Set()});
      }

      const fd = this.unreadByFeed.get(f.id);
      const oldf = fd.feed;
      fd.feed = f;

      if (f.disabled !== oldf.disabled) {
        recalculate = true;
      }

      if (f.categoryId !== oldf.categoryId) {
        mustSort = true;
        recalculate = true;
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
      }

      let change = it.read ? -1 : 1;
      if (it.read) {
        const removed = fd.unread.delete(it.id);
        if (!removed) {
          // Item is read, but we weren't counting it anyway
          return;
        }
        if (fd.unread.size === 0) {
          mustSort = true;
        }
      } else if (!fd.unread.has(it.id)) {
        if (fd.unread.size === 0) {
          mustSort = true;
        }
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

  private emit() {
    const cd = this.unreadByCategory.get(
        this.categoriesByName.get(this.selectedCategoryName))
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
}
