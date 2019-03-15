import {Injectable} from '@angular/core';
import {Observable,
        Subject} from 'rxjs';

/**
 * Service for causing refreshes throughout the app.
 *
 * Refresh refers to the user refreshing the state of the UI, not reloading the
 * entire page.
 */
@Injectable({
  providedIn: 'root'
})
export class RefreshService {
  private readonly started: Subject<void> = new Subject<void>();
  private readonly finished: Subject<void> = new Subject<void>();

  private refreshing: boolean = false;

  constructor() {}

  public isRefreshing(): boolean {
    return this.refreshing;
  }

  public startRefresh() {
    if (this.refreshing) {
      return;
    }
    this.refreshing = true;
    this.started.next();
  }

  public finishRefresh() {
    if (!this.refreshing) {
      return;
    }

    this.refreshing = false;
    this.finished.next();
  }

  public startedObservable(): Observable<void> {
    return this.started;
  }

  public finishedObservable(): Observable<void> {
    return this.finished;
  }
}
