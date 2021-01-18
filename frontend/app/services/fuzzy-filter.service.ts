import {Injectable} from '@angular/core';
import {BehaviorSubject,
        Observable,
} from 'rxjs';
import {debounceTime} from 'rxjs/operators';

// This class exists solely because RouterReuseStrategy is not usable to reuse
// components so it's necessary to store the fuzzy string.
@Injectable({
  providedIn: 'root'
})
export class FuzzyFilterService {
  private readonly fuzzyFilterString$: BehaviorSubject<string> =
      new BehaviorSubject<string>('');

  constructor() {}

  public getFuzzyFilterString(): string {
    return this.fuzzyFilterString$.getValue();
  }

  public pushFuzzyFilterString(s: string) {
    this.fuzzyFilterString$.next(s);
  }

  public fuzzyFilterString(): Observable<string> {
    return this.fuzzyFilterString$.pipe(debounceTime(0));
  }
}
