import {Injectable} from '@angular/core';
import {ParamMap} from '@angular/router';
import {
  BehaviorSubject,
  Observable,
  Subject
} from 'rxjs';
import {debounceTime} from 'rxjs/operators';

@Injectable({
  providedIn: 'root'
})
export class ParamService {
  private readonly mainViewParams$: Subject<ParamMap|void> =
      new BehaviorSubject(undefined);

  constructor() {}


  public pushMainViewParams(p: ParamMap|void) {
    this.mainViewParams$.next(p);
  }

  public mainViewParams(): Observable<ParamMap|void> {
    return this.mainViewParams$.pipe(debounceTime(0));
  }
}
