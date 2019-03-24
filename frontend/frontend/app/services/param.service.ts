import {Injectable} from '@angular/core';
import {ParamMap} from '@angular/router';
import {
  BehaviorSubject,
  ReplaySubject,
  Subject
} from 'rxjs';

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

  public mainViewParams(): Subject<ParamMap|void> {
    return this.mainViewParams$;
  }
}
