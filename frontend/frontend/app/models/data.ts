import {Category,
        Feed,
        Item} from './entities';
import {DataFilter,
        EmptyFilters,
        Filters} from './filter';

export type Entity = Category|Feed|Item;

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
    // Merging with an empty Updates is a no-op
    return this.merge({refresh: false, data: new Data()}, filters)[0];
  }

  // Returns the result of the merge and if anything changed
  public merge(
      u: {refresh: boolean, data: Data},
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
          u.data.categories,
          df.keepExistingCategory,
          df.addNewCategory,
          df);
      changed = changed || c;
    }
    if (!f.excludeFeeds) {
      [feeds, c] = Data.mergeEntities(
          this.feeds,
          u.data.feeds,
          df.keepExistingFeed,
          df.addNewFeed,
          df);
      changed = changed || c;
    }
    if (!f.excludeItems) {
      [items, c] = Data.mergeEntities(
          this.items,
          u.data.items,
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
      private readonly filters: Filters) {
    // Convenience
    this.categories = data.categories;
    this.feeds = data.feeds;
    this.items = data.items;
  }


  public merge(
      u: {refresh: boolean, data: Data}): [FilteredData, boolean] {
    let newData, changed;
    [newData, changed] = this.data.merge(u, this.filters);
    if (!changed) {
      return [this, changed];
    }
    return [new FilteredData(newData, this.filters), changed];
  }
}

export const EmptyFilteredData = new FilteredData(new Data(), EmptyFilters);
