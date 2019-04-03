import {MediaMatcher} from '@angular/cdk/layout';
import {Component,
        HostListener} from '@angular/core';
import {Title} from '@angular/platform-browser';
import {Observable} from 'rxjs';

import {MobileService} from './services/mobile.service';
import {RefreshService} from './services/refresh.service';

@Component({
  selector: 'awrss-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent {
  public mobile: Observable<boolean>;
  public openNav = false;
  public unread = 0;
  public title = 'Aw-RSS';
  public link?: string;

  constructor(
      private readonly mobileService: MobileService,
      private readonly titleService: Title,
      private readonly refreshService: RefreshService) {
    // Uncertain if I actually want to update the page's title constantly,
    // or just the bar at the top
    this.titleService.setTitle(this.title);

    this.mobile = mobileService.mobile();
    // TODO -- Auto close sidenav on navigation
  }

  @HostListener('window:keydown', ['$event'])
  handleKeydown($event: KeyboardEvent) {
    if ($event.key === 'r' && !$event.ctrlKey &&
        !$event.altKey && !$event.shiftKey &&
        !$event.metaKey) {
      this.refreshService.startRefresh();
    }
  }

  // TODO -- Close nav on route change for mobile

  public refresh() {
    this.refreshService.startRefresh();
  }

  public isRefreshing(): boolean {
    return this.refreshService.isRefreshing();
  }
}
