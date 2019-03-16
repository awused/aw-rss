import {MediaMatcher} from '@angular/cdk/layout';
import {Component,
        NgZone,
        OnDestroy,
        ViewChild} from '@angular/core';
import {Title} from '@angular/platform-browser';

@Component({
  selector: 'awrss-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnDestroy {
  public mobileQuery: MediaQueryList;
  public openNav = false;

  private mobileQueryListener: () => void;

  constructor(
      private readonly zone: NgZone,
      private readonly media: MediaMatcher,
      private readonly titleService: Title) {
    this.mobileQuery = media.matchMedia('(max-width: 768px)');
    // NgZone is the only option that doesn't break regular change detection
    this.mobileQueryListener = () => zone.run(() => true);
    this.mobileQuery.addEventListener('change', this.mobileQueryListener);
    this.titleService.setTitle('Aw-RSS');
  }

  ngOnDestroy(): void {
    // This should never actually run
    this.mobileQuery.removeEventListener('change', this.mobileQueryListener);
  }
}
