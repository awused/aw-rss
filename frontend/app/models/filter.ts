import {Category,
        Feed,
        Item} from './entities';

export interface TimeRange {
  // Matches items in the range [stard, end)
  // Dates are UTC
  readonly end?: Date;
  readonly start: Date;
}

// TODO -- Simplify: we'll never care about multiple categories or feeds at once

// Filters for applying updates or filtering data
// By default everything is kept unconditionally
export interface Filters {
  // Discard all invalid (disabled, read, etc) feeds or items
  // unread items for discarded feeds are also "invalid"
  readonly validOnly?: boolean;
  // Exclude items that have been read.
  readonly unreadOnly?: boolean;
  readonly isMainView?: boolean;
  // Whether to keep existing entities unconditionally
  // on non-refresh updates. When it's not a refresh existing objects will be
  // kept and updated, but new objects won't be added.
  // The purpose of this filter is to avoid unexpected UI shuffling.
  // Only affects updates, meaningless without validOnly or unreadOnly.
  readonly keepUnlessRefresh?: boolean;
  // If feed or category IDs are supplied those will be considered valid even
  // if they would be excluded by validOnly. Feeds not included in either a
  // category or directly by ID will be excluded.
  readonly categoryName?: string;
  readonly feedId?: number;
  readonly itemIds?: ReadonlyArray<number>;
  // Exclude these types, mostly to improve performance.
  // These get applied first and will break some other filters.
  readonly excludeCategories?: boolean;
  readonly excludeFeeds?: boolean;
  readonly excludeItems?: boolean;
  // Used by components that will never request new data on their own
  readonly doNotFetch?: boolean;
}

export interface PartialFilters extends Filters {
  validOnly?: boolean;
  unreadOnly?: boolean;
  isMainView?: boolean;
  keepUnlessRefresh?: boolean;
  categoryName?: string;
  feedId?: number;
  itemIds?: ReadonlyArray<number>;
  excludeCategories?: boolean;
  excludeFeeds?: boolean;
  excludeItems?: boolean;
  doNotFetch?: boolean;
}

export const EmptyFilters: Filters = {
  excludeCategories: true,
  excludeFeeds: true,
  excludeItems: true,
};

// Use an interface to make it convenient to specify filters elsewhere
export class DataFilter {
  private readonly keepExisting: boolean;
  private readonly itemIds: ReadonlySet<number>;
  private readonly excludedCategories: Set<number> = new Set<number>();
  private readonly includedFeedIds: Set<number> = new Set<number>();
  private categoryId?: number;

  constructor(
      refresh: boolean,
      private readonly f: Filters) {
    this.itemIds = new Set(f.itemIds || []);
    this.keepExisting = !refresh && !!f.keepUnlessRefresh;
  }

  keepExistingCategory = (c: Category): boolean => {
    if (this.f.categoryName !== undefined) {
      if (this.f.categoryName === c.name) {
        this.categoryId = c.id;
        return true;
      }
      return false;
    }

    if (this.keepExisting) {
      return true;
    }

    return this.addNewCategory(c);
  }

  addNewCategory = (c: Category): boolean => {
    if (this.f.categoryName !== undefined) {
      if (this.f.categoryName === c.name) {
        this.categoryId = c.id;
        return true;
      }
      return false;
    }

    if (!c.disabled &&
        this.f.isMainView && (c.hiddenMain || c.hiddenNav)) {
      // hiddenMain categories are only included when referenced
      // directly by name
      this.excludedCategories.add(c.id);
      return false;
    }

    return !this.f.validOnly || !c.disabled;
  }

  keepExistingFeed = (f: Feed): boolean => {
    if (this.f.feedId !== undefined) {
      if (this.f.feedId === f.id) {
        this.includedFeedIds.add(f.id);
        return true;
      }
      return false;
    }

    if (this.categoryId !== undefined && f.categoryId !== this.categoryId) {
      return false;
    }

    if (f.categoryId !== undefined &&
        this.excludedCategories.has(f.categoryId)) {
      return false;
    }

    if (this.keepExisting) {
      this.includedFeedIds.add(f.id);
      return true;
    }

    return this.addNewFeed(f);
  }

  addNewFeed = (f: Feed): boolean => {
    if (this.f.feedId !== undefined) {
      if (this.f.feedId === f.id) {
        this.includedFeedIds.add(f.id);
        return true;
      }
      return false;
    }

    if (this.f.validOnly && f.disabled) {
      return false;
    }

    if (f.categoryId !== undefined &&
        this.excludedCategories.has(f.categoryId)) {
      return false;
    }

    if (this.f.categoryName &&
        (this.categoryId === undefined || f.categoryId !== this.categoryId)) {
      return false;
    }

    this.includedFeedIds.add(f.id);
    return true;
  }

  keepExistingItem = (i: Item): boolean => {
    if ((this.includedFeedIds.size !== 0 || this.categoryId !== undefined) &&
        !this.includedFeedIds.has(i.feedId)) {
      return false;
    }

    if (this.keepExisting) {
      return true;
    }

    return this.addNewItem(i);
  }

  addNewItem = (i: Item): boolean => {
    if (this.itemIds.size !== 0) {
      if (this.itemIds.has(i.id)) {
        return true;
      }
    }

    if (this.f.unreadOnly && i.read) {
      return false;
    }

    if (!this.includedFeedIds.has(i.feedId)) {
      return false;
    }

    if (this.f.feedId !== undefined && this.f.feedId !== i.feedId) {
      return false;
    }

    if (this.itemIds.size !== 0) {
      return false;
    }
    return true;
  }
}
