import {Component,
        Input} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {ConfirmationDialogComponent} from 'frontend/app/admin/confirmation-dialog/confirmation-dialog.component';
import {EditCategoryDialogComponent} from 'frontend/app/admin/edit-category-dialog/edit-category-dialog.component';
import {EditFeedDialogComponent} from 'frontend/app/admin/edit-feed-dialog/edit-feed-dialog.component';
import {Category,
        Feed} from 'frontend/app/models/entities';
import {FeedTitlePipe} from 'frontend/app/pipes/feed-title.pipe';
import {FuzzyFilterService} from 'frontend/app/services/fuzzy-filter.service';
import {MutateService} from 'frontend/app/services/mutate.service';

@Component({
  selector: 'awrss-main-view-header',
  templateUrl: './header.component.html',
  styleUrls: ['./header.component.scss']
})
export class MainViewHeaderComponent {
  @Input()
  public feed?: Feed;
  @Input()
  public category?: Category;
  @Input()
  public mobile = false;

  @Input()
  public maxItemId?: number;

  public fuzzyString: string;


  constructor(
      private readonly dialog: MatDialog,
      private readonly feedTitle: FeedTitlePipe,
      private readonly mutateService: MutateService,
      public readonly fuzzyFilterService: FuzzyFilterService) {
    this.fuzzyString = this.fuzzyFilterService.getFuzzyFilterString();
  }

  public edit() {
    if (this.feed) {
      this.dialog.open(EditFeedDialogComponent, {
        data: {feed: this.feed}
      });
    } else if (this.category) {
      this.dialog.open(EditCategoryDialogComponent, {
        data: {category: this.category}
      });
    }
  }


  public markFeedAsRead() {
    // TODO -- enable fuzzy filtering for marking all as read
    if (!this.feed || !this.maxItemId || this.fuzzyString) {
      return;
    }

    const feed = this.feed;
    const maxItemId = this.maxItemId;

    this.dialog.open<any, any, boolean>(ConfirmationDialogComponent, {
                 data: {
                   title: 'Confirm Action',
                   text: [
                     `Mark all items read for
                     ${this.feedTitle.transform(this.feed)}?`,
                     `This action is irreversible.`
                   ]
                 }
               })
        .beforeClosed()
        .subscribe((result) => {
          if (result) {
            this.mutateService.markFeedAsRead(feed.id, maxItemId);
          }
        });
  }

  public rerunFeed() {
    if (!this.feed) {
      return;
    }

    this.mutateService.rerunFeed(this.feed.id);
  }

  handleFuzzy(value: string) {
    this.fuzzyString = value;
    this.fuzzyFilterService.pushFuzzyFilterString(value);
  }
}
