import {Component,
        Input,
        OnInit} from '@angular/core';
import {MatDialog} from '@angular/material';
import {EditFeedDialogComponent} from 'frontend/app/admin/edit-feed-dialog/edit-feed-dialog.component';
import {Category,
        Feed} from 'frontend/app/models/entities';

@Component({
  selector: 'awrss-main-view-header',
  templateUrl: './header.component.html',
  styleUrls: ['./header.component.scss']
})
export class MainViewHeaderComponent implements OnInit {
  @Input()
  public feed?: Feed;
  @Input()
  public category?: Category;
  @Input()
  public mobile: boolean;

  constructor(
      private readonly dialog: MatDialog) {}

  ngOnInit() {
  }

  public edit() {
    if (this.feed) {
      this.dialog.open(EditFeedDialogComponent, {
        width: '400px',
      });

    } else if (this.category) {
    }
  }
}
