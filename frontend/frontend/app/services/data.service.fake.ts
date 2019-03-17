import {
  BehaviorSubject,
  Observable,
  of,
  ReplaySubject,
  Subject
} from 'rxjs';

import {Category} from '../models/entities';
import {Data,
        FilteredData} from '../models/data';
import {Feed} from '../models/entities';
import {EmptyFilters,
        Filters} from '../models/filter';
import {Item} from '../models/entities';
import {FeedFixtures} from '../models/models.fake';
import {Updates} from '../models/updates';

export class FakeDataService {
  constructor() {}


  public dataForFilters(f: Filters): Observable<FilteredData> {
    return of(new FilteredData(new Data(), EmptyFilters));
  }

  public updates(): Observable<Updates> {
    return of(new Updates());
  }

  public getFeed(id: number): Feed {
    return FeedFixtures.emptyFeed;
  }
}
