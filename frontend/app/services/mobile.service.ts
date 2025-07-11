import {MediaMatcher} from '@angular/cdk/layout';
import {inject,
        Injectable} from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  Subject
} from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class MobileService {
  private readonly mobileQuery: MediaQueryList;
  private readonly mobile$: Subject<boolean>;

  private mobileQueryListener: () => void;

  constructor() {
    const media = inject(MediaMatcher);

    this.mobileQuery = media.matchMedia('(max-width: 768px)');
    this.mobile$ = new BehaviorSubject(this.mobileQuery.matches);
    this.mobileQueryListener = () => this.mobile$.next(this.mobileQuery.matches);
    // This service never gets destroyed
    this.mobileQuery.addEventListener('change', this.mobileQueryListener);
  }

  public mobile(): Observable<boolean> {
    return this.mobile$;
  }
}
