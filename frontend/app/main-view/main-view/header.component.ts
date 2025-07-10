import {Component,
        EventEmitter,
        Input,
        Output} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {EditCategoryDialogComponent} from 'frontend/app/admin/edit-category-dialog/edit-category-dialog.component';
import {EditFeedDialogComponent} from 'frontend/app/admin/edit-feed-dialog/edit-feed-dialog.component';
import {Category,
        Feed} from 'frontend/app/models/entities';
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

  @Output()
  public markAsRead = new EventEmitter<void>();

  @Input()
  public enableMarkAsRead?: boolean;

  public fuzzyString: string;


  constructor(
      private readonly dialog: MatDialog,
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


  public handleMarkReadClick() {
    this.markAsRead.emit();
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
