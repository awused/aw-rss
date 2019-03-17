import {Observable,
        of} from 'rxjs';

import {Item} from '../models/entities';

export class FakeMutateService {
  constructor() {}


  public setItemRead(it: Item, read: boolean): Observable<void> {
    return of();
  }
}
