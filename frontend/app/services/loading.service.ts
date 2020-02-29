import {Injectable} from '@angular/core';
import {BehaviorSubject,
        Observable,
        Subject} from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class LoadingService {
  private loadingSem = 0;
  private loading$: Subject<boolean> = new BehaviorSubject<boolean>(false);

  constructor() {}

  public startLoading() {
    if (this.loadingSem === 0) {
      this.loading$.next(true);
    }
    this.loadingSem++;
  }

  public finishLoading() {
    if (this.loadingSem > 0) {
      this.loadingSem--;
      if (this.loadingSem === 0) {
        this.loading$.next(false);
      }
    }
  }

  public isLoading(): boolean {
    return this.loadingSem > 0;
  }

  public loading(): Observable<boolean> {
    return this.loading$;
  }
}
