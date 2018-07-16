import { Injectable } from '@angular/core';
import { Observable, BehaviorSubject } from 'rxjs';

/**
 * Service for causing refreshes throughout the app.
 */
@Injectable({
  providedIn: 'root'
})
export class RefreshService {
  private subj: BehaviorSubject<null> = new BehaviorSubject<null>(null);

  constructor() { }

  public refresh() {
    this.subj.next(null);
  }

  public subject(): BehaviorSubject<null> {
    return this.subj;
  }
}
