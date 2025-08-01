import {Component,
        HostListener,
        inject} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {Title} from '@angular/platform-browser';
import {Router} from '@angular/router';
import {Observable} from 'rxjs';

import {AddDialogComponent} from './admin/add-dialog/add-dialog.component';
import {FuzzyFilterService} from './services/fuzzy-filter.service';
import {MobileService} from './services/mobile.service';
import {RefreshService} from './services/refresh.service';

@Component({
  selector: 'awrss-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  standalone: false
})
export class AppComponent {
  private readonly dialog = inject(MatDialog);
  private readonly titleService = inject(Title);
  private readonly refreshService = inject(RefreshService);
  private readonly router = inject(Router);
  private readonly fuzzyFilterService = inject(FuzzyFilterService);

  public mobile: Observable<boolean>;
  public openNav = false;
  public unread = 0;
  public title = 'Aw-RSS';
  public link?: string;
  public fuzzyString: string;

  constructor() {
    const mobileService = inject(MobileService);

    // Uncertain if I actually want to update the page's title constantly,
    // or just the bar at the top
    this.titleService.setTitle(this.title);

    this.mobile = mobileService.mobile();

    this.fuzzyString = this.fuzzyFilterService.getFuzzyFilterString();
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

  handleFuzzy(value: string) {
    this.fuzzyString = value;
    this.fuzzyFilterService.pushFuzzyFilterString(value);
  }
}
