import {Component,
        OnDestroy,
        OnInit} from '@angular/core';
import {MatDialog} from '@angular/material';
import {EmptyFilteredData,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {Feed} from 'frontend/app/models/entities';
import {DataService} from 'frontend/app/services/data.service';
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
  public filteredData: FilteredData = EmptyFilteredData;

  constructor(
      private readonly dataService: DataService,
      private readonly dialog: MatDialog) {}

  ngOnInit() {
    this.dataService.updates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(
            (u: Updates) => this.filteredData = this.filteredData.merge(u)[0]);

    this.dataService.dataForFilters({
                      excludeCategories: true,
                      excludeItems: true,
                      validOnly: false,
                    })
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((fd: FilteredData) => this.filteredData = fd);
  }

  editFeed(feed: Feed) {
    this.dialog.open(EditFeedDialogComponent, {
      data: {feed}
    });
  }

  ngOnDestroy() {
    this.onDestroy$.next();
    this.onDestroy$.complete();
  }
}
