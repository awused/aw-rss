import {Component,
        Inject} from '@angular/core';
import {FormControl,
        FormGroup} from '@angular/forms';
import {MAT_DIALOG_DATA,
        MatDialogRef} from '@angular/material/dialog';
import {MatSnackBar} from '@angular/material/snack-bar';
import {FilteredData} from 'frontend/app/models/data';
import {Category,
        Feed} from 'frontend/app/models/entities';
import {FeedTitlePipe} from 'frontend/app/pipes/feed-title.pipe';
import {DataService} from 'frontend/app/services/data.service';
import {MutateService} from 'frontend/app/services/mutate.service';

@Component({
    selector: 'awrss-edit-feed-dialog',
    templateUrl: './edit-feed-dialog.component.html',
    styleUrls: ['./edit-feed-dialog.component.scss'],
    standalone: false
})
export class EditFeedDialogComponent {
  public readonly feed: Feed;
  public feedForm: FormGroup;
  public categories: ReadonlyArray<Category> = [];

  // This controller will never be reused
  constructor(
      private readonly dialogRef: MatDialogRef<EditFeedDialogComponent>,
      private readonly dataService: DataService,
      private readonly mutateService: MutateService,
      private readonly snackBar: MatSnackBar,
      private readonly feedTitlePipe: FeedTitlePipe,
      @Inject(MAT_DIALOG_DATA) public readonly data: {
        feed: Feed;
      }) {
    this.feed = data.feed;

    this.feedForm = new FormGroup({
      userTitle: new FormControl(this.feed.userTitle),
      enabled: new FormControl(!this.feed.disabled),
      categoryId: new FormControl(this.feed.categoryId),
    });

    // Uses take(1) internally
    this.dataService.dataForFilters({
                      validOnly: true,
                      excludeFeeds: true,
                      excludeItems: true,
                    })
        .subscribe((fd: FilteredData) => this.categories = fd.categories);
  }

  public isUnchanged(): boolean {
    if (this.feed.userTitle !== this.feedForm.get('userTitle')?.value) {
      return false;
    }

    if (this.feedForm.get('categoryId')?.dirty &&
        this.feed.categoryId !== this.feedForm.get('categoryId')?.value) {
      return false;
    }

    if (this.feed.disabled !== !this.feedForm.get('enabled')?.value) {
      return false;
    }

    return true;
  }

  public submit(formValue: any) {
    const edit: any = {};

    if (this.feed.userTitle !== (formValue.userTitle || undefined)) {
      edit.userTitle = formValue.userTitle || '';
    }

    const categoryId = formValue.categoryId === null ?
        undefined :
        formValue.categoryId;
    if (this.feed.categoryId !== categoryId) {
      if (categoryId !== undefined) {
        edit.categoryId = categoryId;
      } else {
        edit.clearCategory = true;
      }
    }

    if (this.feed.disabled !== !formValue.enabled) {
      edit.disabled = !formValue.enabled;
    }

    this.mutateService
        .editFeed(this.feed, edit)
        .subscribe(() => {
          this.snackBar.open(
              `Edited Feed [${this.feedTitlePipe.transform(this.feed)}]`, '', {
                duration: 3000
              });
          this.dialogRef.close();
        });
  }
}
