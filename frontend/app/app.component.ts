import {MediaMatcher} from '@angular/cdk/layout';
import {Component,
        HostListener} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {Title} from '@angular/platform-browser';
import {Router} from '@angular/router';
import {Observable} from 'rxjs';

import {AddDialogComponent} from './admin/add-dialog/add-dialog.component';
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
      private readonly dialog: MatDialog,
      private readonly mobileService: MobileService,
      private readonly titleService: Title,
      private readonly refreshService: RefreshService,
      private readonly router: Router) {
    // Uncertain if I actually want to update the page's title constantly,
    // or just the bar at the top
    this.titleService.setTitle(this.title);

    this.mobile = mobileService.mobile();
    // TODO -- Auto close sidenav on navigation
  }

  // For global hotkeys
  @HostListener('window:keydown', ['$event'])
  handleKeydown($event: KeyboardEvent) {
    if (!$event.ctrlKey &&
        !$event.altKey &&
        !$event.shiftKey &&
        !$event.metaKey &&
        !($event.target instanceof HTMLInputElement &&
          $event.target.type === 'text') &&
        !this.dialog.openDialogs.length) {
      if ($event.key === 'r') {
        this.refreshService.startRefresh();
      }

      if ($event.key === 'n') {
        this.dialog.open(AddDialogComponent);
      }

      if ($event.key === 'a') {
        this.router.navigate(['admin']);
      }
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
