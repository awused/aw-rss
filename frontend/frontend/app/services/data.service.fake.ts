import {
  BehaviorSubject,
  Observable,
  of,
  ReplaySubject,
  Subject
} from 'rxjs';

import {Data,
        FilteredData,
        Updates} from '../models/data';
import {Category} from '../models/entities';
import {Feed} from '../models/entities';
import {Item} from '../models/entities';
import {EmptyFilters,
        Filters} from '../models/filter';
import {FeedFixtures} from '../models/models.fake';

export class FakeDataService {
  constructor() {}


  public dataForFilters(f: Filters): Observable<FilteredData> {
    return of(new FilteredData(new Data(), EmptyFilters));
  }

  public updates(): Observable<Updates> {
    return of(new Updates());
  }

  public feedUpdates(): Observable<void> {
    return of();
  }


  public getFeed(id: number): Feed {
    return FeedFixtures.emptyFeed;
  }
}
