import {Category,
        Feed,
        Item} from './entities';

// Filters for applying updates or filtering data
// By default everything is kept unconditionally
export interface Filters {
  // Discard all invalid (disabled, read, etc) feeds or items
  // unread items for discarded feeds are also "invalid"
  readonly validOnly?: boolean;
  // Exclude items that have been read.
  readonly unreadOnly?: boolean;
  // Whether to keep existing disabled feeds/categories or read items
  // on non-refresh updates. When it's not a refresh existing objects will be
  // kept and updated, but new objects won't be added.
  // The purpose of this filter is to avoid unexpected UI shuffling.
  // Only affects updates, meaningless without validOnly or unreadOnly.
  readonly keepExistingUnlessRefresh?: boolean;
  // If feed or category IDs are supplied those will be considered valid even
  // if they would be excluded by validOnly. Feeds not included in either a
  // category or directly by ID will be excluded.
  // Setting multiple at the same time is treated as a union.
  // An empty array is the same as not specifying it.
  readonly categoryIds?: number[];
  readonly feedIds?: number[];
  readonly itemIds?: number[];
  // Exclude these types, mostly to improve performance.
  // These get applied first and will break some other filters.
  readonly excludeCategories?: boolean;
  readonly excludeFeeds?: boolean;
  readonly excludeItems?: boolean;
}

export const EmptyFilters: Filters = {
  excludeCategories: true,
  excludeFeeds: true,
  excludeItems: true,
};

// Use an interface to make it convenient to specify filters elsewhere
export class DataFilter {
  readonly keepExisting: boolean;
  readonly categoryIds: ReadonlySet<number>;
  readonly feedIds: ReadonlySet<number>;
  readonly itemIds: ReadonlySet<number>;
  readonly categoryFeedIds: Set<number> = new Set<number>();
  readonly includedFeedIds: Set<number> = new Set<number>();

  constructor(
      refresh: boolean,
      private readonly f: Filters) {
    this.categoryIds = new Set(f.categoryIds || []);
    this.feedIds = new Set(f.feedIds || []);
    this.itemIds = new Set(f.itemIds || []);
    this.keepExisting = !refresh && !!f.keepExistingUnlessRefresh;
  }

  keepExistingCategory = (c: Category): boolean => {
    if (this.categoryIds.size !== 0 && this.categoryIds.has(c.id)) {
      // TODO -- add all feed ids to categoryFeedIds
    }

    if (this.keepExisting) {
      return true;
    }

    return this.addNewCategory(c);
  }

  addNewCategory = (c: Category): boolean => {
    if (this.categoryIds.size !== 0) {
      if (this.categoryIds.has(c.id)) {
        // TODO -- add all feed ids to categoryFeedIds
        return true;
      }
      return false;
    }

    return !this.f.validOnly || !c.disabled;
  }

  keepExistingFeed = (f: Feed): boolean => {
    if (this.keepExisting) {
      this.includedFeedIds.add(f.id);
      return true;
    }

    return this.addNewFeed(f);
  }

  addNewFeed = (f: Feed): boolean => {
    if (this.feedIds.size !== 0) {
      if (this.feedIds.has(f.id)) {
        this.includedFeedIds.add(f.id);
        return true;
      }
    }

    if (this.f.validOnly && f.disabled) {
      return false;
    }

    if (this.categoryIds.size !== 0) {
      if (f.disabled) {
        // A disabled feed will never be newly included in a category
        return false;
      }
      if (this.categoryFeedIds.has(f.id)) {
        this.includedFeedIds.add(f.id);
        return true;
      }
    }

    if (this.feedIds.size !== 0) {
      return false;
    }
    this.includedFeedIds.add(f.id);
    return true;
  }

  keepExistingItem = (i: Item): boolean => {
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

    if (this.includedFeedIds.size !== 0 && !this.includedFeedIds.has(i.feedId)) {
      return false;
    }

    if (this.itemIds.size !== 0) {
      return false;
    }
    return true;
  }
}
