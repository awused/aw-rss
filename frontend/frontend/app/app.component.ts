import {MediaMatcher} from '@angular/cdk/layout';
import {Component,
        HostListener,
        NgZone,
        OnDestroy,
        ViewChild} from '@angular/core';
import {Title} from '@angular/platform-browser';

import {RefreshService} from './services/refresh.service';

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
      private readonly titleService: Title,
      private readonly refreshService: RefreshService) {
    this.mobileQuery = media.matchMedia('(max-width: 768px)');
    // NgZone is the only option that doesn't break regular change detection
    this.mobileQueryListener = () => zone.run(() => true);
    this.mobileQuery.addEventListener('change', this.mobileQueryListener);
    this.titleService.setTitle('Aw-RSS');
  }

  @HostListener('window:keydown', ['$event'])
  handleKeydown($event: KeyboardEvent) {
    if ($event.key === 'r' && !$event.ctrlKey &&
        !$event.altKey && !$event.shiftKey &&
        !$event.metaKey) {
      this.refreshService.startRefresh();
    }
  }

  ngOnDestroy(): void {
    // This should never actually run
    this.mobileQuery.removeEventListener('change', this.mobileQueryListener);
  }
}
