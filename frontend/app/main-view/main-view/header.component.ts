import {Component,
        Input,
        OnInit} from '@angular/core';
import {MatDialog} from '@angular/material';
import {ConfirmationDialogComponent} from 'frontend/app/admin/confirmation-dialog/confirmation-dialog.component';
import {EditFeedDialogComponent} from 'frontend/app/admin/edit-feed-dialog/edit-feed-dialog.component';
import {Category,
        Feed} from 'frontend/app/models/entities';
import {FeedTitlePipe} from 'frontend/app/pipes/feed-title.pipe';
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
  public mobile: boolean;

  @Input()
  public maxItemId?: number;

  constructor(
      private readonly dialog: MatDialog,
      private readonly feedTitle: FeedTitlePipe,
      private readonly mutateService: MutateService) {}

  public edit() {
    if (this.feed) {
      this.dialog.open(EditFeedDialogComponent);
    } else if (this.category) {
    }
  }


  public markFeedAsRead() {
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
            this.mutateService.markFeedAsRead(this.feed.id, this.maxItemId);
          }
        });
  }
}
