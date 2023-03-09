import {Component,
        OnDestroy,
        OnInit} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {EMPTY_FILTERED_DATA,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {Feed} from 'frontend/app/models/entities';
import {DataService} from 'frontend/app/services/data.service';
import {filter as fuzzyFilter,
        FilterOptions} from 'fuzzy/lib/fuzzy.js';
import {Subject} from 'rxjs';
import {takeUntil} from 'rxjs/operators';

import {EditFeedDialogComponent} from '../edit-feed-dialog/edit-feed-dialog.component';

@Component({
  selector: 'awrss-feed-admin',
  templateUrl: './feed-admin.component.html',
  styleUrls: ['./feed-admin.component.scss']
})
export class FeedAdminComponent implements OnInit, OnDestroy {
  private readonly onDestroy$: Subject<void> = new Subject();
  private fuzzyFilterString = '';
  private readonly fuzzyOptions: FilterOptions<Feed> = {
    extract: (f: Feed) => (f.userTitle ?? '') + ' ' + f.title + ' ' + f.siteUrl + ' ' + f.url,
  };

  public filteredData: FilteredData = EMPTY_FILTERED_DATA;
  public fuzzyFeeds: ReadonlyArray<Feed> = [];

  constructor(
      private readonly dataService: DataService,
      private readonly dialog: MatDialog) {}

  ngOnInit() {
    this.dataService.updates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(
            (u: Updates) => {
              this.filteredData = this.filteredData.merge(u)[0];
              this.handleFuzzy(this.fuzzyFilterString);
            });

    this.dataService.dataForFilters({
                      excludeCategories: true,
                      excludeItems: true,
                      validOnly: false,
                    })
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((fd: FilteredData) => {
          this.filteredData = fd;
          this.handleFuzzy(this.fuzzyFilterString);
        });
  }

  editFeed(feed: Feed) {
    this.dialog.open(EditFeedDialogComponent, {
      data: {feed}
    });
  }

  handleFuzzy(value: string) {
    this.fuzzyFilterString = value;
    if (!value) {
      this.fuzzyFeeds = this.filteredData.feeds;
      return;
    }

    this.fuzzyFeeds =
        fuzzyFilter(
            value, this.filteredData.feeds.slice(), this.fuzzyOptions)
            .map(x => x.original);
  }

  ngOnDestroy() {
    this.onDestroy$.next();
    this.onDestroy$.complete();
  }
}
