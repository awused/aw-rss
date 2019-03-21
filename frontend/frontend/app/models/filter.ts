import {Category,
        Feed,
        Item} from './entities';

export interface TimeRange {
  // Matches items in the range (stard, end]
  // Dates are UTC
  // This seems backwards, but items are displayed in reverse order
  readonly end: Date;
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
  readonly excludeHidden?: boolean;
  // Whether to keep existing entities unconditionally
  // on non-refresh updates. When it's not a refresh existing objects will be
  // kept and updated, but new objects won't be added.
  // The purpose of this filter is to avoid unexpected UI shuffling.
  // Only affects updates, meaningless without validOnly or unreadOnly.
  readonly keepUnlessRefresh?: boolean;
  // If feed or category IDs are supplied those will be considered valid even
  // if they would be excluded by validOnly. Feeds not included in either a
  // category or directly by ID will be excluded.
  readonly categoryIds?: ReadonlyArray<number>;
  readonly feedId?: number;
  readonly itemIds?: ReadonlyArray<number>;
  // Exclude these types, mostly to improve performance.
  // These get applied first and will break some other filters.
  readonly excludeCategories?: boolean;
  readonly excludeFeeds?: boolean;
  readonly excludeItems?: boolean;
  readonly timeRange?: TimeRange;
}

export interface PartialFilters extends Filters {
  validOnly?: boolean;
  unreadOnly?: boolean;
  excludeHidden?: boolean;
  keepUnlessRefresh?: boolean;
  categoryIds?: ReadonlyArray<number>;
  feedId?: number;
  itemIds?: ReadonlyArray<number>;
  excludeCategories?: boolean;
  excludeFeeds?: boolean;
  excludeItems?: boolean;
  timeRange?: TimeRange;
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
  readonly itemIds: ReadonlySet<number>;
  readonly categoryFeedIds: Set<number> = new Set<number>();
  readonly includedFeedIds: Set<number> = new Set<number>();
  readonly end?: string;
  readonly start?: string;

  constructor(
      refresh: boolean,
      private readonly f: Filters) {
    this.categoryIds = new Set(f.categoryIds || []);
    this.itemIds = new Set(f.itemIds || []);
    this.keepExisting = !refresh && !!f.keepUnlessRefresh;
    if (f.timeRange) {
      this.end = f.timeRange.end.toISOString();
      this.start = f.timeRange.start.toISOString();
      // TODO -- handle time range
    }
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

    if (this.f.feedId !== undefined && this.f.feedId !== i.feedId) {
      return false;
    }

    if (this.itemIds.size !== 0) {
      return false;
    }
    return true;
  }
}
