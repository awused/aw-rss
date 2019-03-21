import {Category,
        Feed,
        Item} from './entities';
import {DataFilter,
        EmptyFilters,
        Filters} from './filter';

export type Entity = Category|Feed|Item;

export class Updates {
  constructor(
      // If this came from a user-triggered refresh or not
      // All updates of any kind get transformed into an update object.
      public readonly refresh: boolean = false,
      public readonly categories: ReadonlyArray<Category> = [],
      public readonly feeds: ReadonlyArray<Feed> = [],
      public readonly items: ReadonlyArray<Item> = [],
  ) {}
}

// A collection of categories, items, and feeds
// All arrays are sorted by IDs in ascending order
export class Data {
  constructor(
      public readonly categories: ReadonlyArray<Category> = [],
      public readonly feeds: ReadonlyArray<Feed> = [],
      public readonly items: ReadonlyArray<Item> = [],
  ) {}

  private static mergeEntities<T extends Entity>(
      dEntities: ReadonlyArray<T>,
      uEntities: ReadonlyArray<T>,
      keepExisting: (T) => boolean,
      addNew: (T) => boolean,
      df: DataFilter): [ReadonlyArray<T>, boolean] {
    // Most merges are a relatively small number of updates into a larger list,
    // and tend to be newer items rather than older ones.
    // Optimize by using splice when changes are found rather than always
    // creating a new array.
    const merged = [];
    let changed = false;

    let di = 0;
    let de: T|undefined;
    for (let ui = 0; ui < uEntities.length; ui++) {
      const ue = uEntities[ui];
      for (; di < dEntities.length; di++) {
        de = dEntities[di];
        if (de.id < ue.id) {
          if (keepExisting(de)) {
            merged.push(de);
          } else {
            changed = true;
          }
        } else {
          break;
        }
      }
      if (de && de.id === ue.id) {
        di++;
        if (ue.commitTimestamp < de.commitTimestamp) {
          if (keepExisting(de)) {
            merged.push(de);
            continue;
          }
        } else if (keepExisting(ue)) {
          merged.push(ue);
        }
        changed = true;
      } else if (addNew(ue)) {
        changed = true;
        merged.push(ue);
      }
    }

    for (; di < dEntities.length; di++) {
      de = dEntities[di];
      if (keepExisting(de)) {
        merged.push(de);
      } else {
        changed = true;
      }
    }

    return [merged, changed];
  }

  // Returns the result of the filter and if it was changed
  public filter(filters: Filters = {}): Data {
    return this.merge(new Updates(true), filters)[0];
  }

  // Returns the result of the merge and if anything changed
  public merge(
      u: Updates,
      f: Filters = {}): [Data, boolean] {
    const df = new DataFilter(u.refresh, f);
    let cats: ReadonlyArray<Category> = [];
    let feeds: ReadonlyArray<Feed> = [];
    let items: ReadonlyArray<Item> = [];
    let changed = false;
    let c = false;

    if (!f.excludeCategories) {
      [cats, c] = Data.mergeEntities(
          this.categories,
          u.categories,
          df.keepExistingCategory,
          df.addNewCategory,
          df);
      changed = changed || c;
    }
    if (!f.excludeFeeds) {
      [feeds, c] = Data.mergeEntities(
          this.feeds,
          u.feeds,
          df.keepExistingFeed,
          df.addNewFeed,
          df);
      changed = changed || c;
    }
    if (!f.excludeItems) {
      [items, c] = Data.mergeEntities(
          this.items,
          u.items,
          df.keepExistingItem,
          df.addNewItem,
          df);
      changed = changed || c;
    }
    if (!changed) {
      return [this, changed];
    }
    return [new Data(cats, feeds, items), changed];
  }
}

export class FilteredData {
  public readonly categories: ReadonlyArray<Category>;
  public readonly feeds: ReadonlyArray<Feed>;
  public readonly items: ReadonlyArray<Item>;

  constructor(
      private readonly data: Data,
      public readonly filters: Filters) {
    // Convenience
    this.categories = data.categories;
    this.feeds = data.feeds;
    this.items = data.items;
  }


  public merge(u: Updates): [FilteredData, boolean] {
    let newData, changed;
    [newData, changed] = this.data.merge(u, this.filters);
    if (!changed) {
      return [this, changed];
    }
    return [new FilteredData(newData, this.filters), changed];
  }
}

export const EmptyFilteredData = new FilteredData(new Data(), EmptyFilters);
