import {MediaMatcher} from '@angular/cdk/layout';
import {Component, NgZone, OnDestroy, ViewChild} from '@angular/core';
import {MatSidenav} from '@angular/material/sidenav';

@Component({
  selector : 'app-root',
  templateUrl : './app.component.html',
  styleUrls : [ './app.component.scss' ]
})
export class AppComponent implements OnDestroy {
  mobileQuery: MediaQueryList;
  openNav: boolean = false;

  private mobileQueryListener: () => void;

  constructor(zone: NgZone, media: MediaMatcher) {
    this.mobileQuery = media.matchMedia('(max-width: 768px)');
    // NgZone is the only option that doesn't break regular change detection
    this.mobileQueryListener = () => zone.run(() => true);
    this.mobileQuery.addListener(this.mobileQueryListener);
  }

  ngOnDestroy(): void {
    // This should never actually run
    this.mobileQuery.removeListener(this.mobileQueryListener);
  }
}
