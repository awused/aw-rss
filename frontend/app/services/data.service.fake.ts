import {
  Observable,
  of
} from 'rxjs';

import {Data,
        FilteredData,
        Updates} from '../models/data';
import {Feed} from '../models/entities';
import {EMPTY_FILTERS,
        Filters} from '../models/filter';
import {FEED_FIXTURES} from '../models/models.fake';

export class FakeDataService {
  constructor() {}


  public dataForFilters(_f: Filters): Observable<FilteredData> {
    return of(new FilteredData(new Data(), EMPTY_FILTERS));
  }

  public updates(): Observable<Updates> {
    return of(new Updates());
  }

  public feedUpdates(): Observable<void> {
    return of();
  }

  public categoryUpdates(): Observable<void> {
    return of();
  }

  public getFeed(id: number): Feed {
    return FEED_FIXTURES.emptyFeed;
  }
}
