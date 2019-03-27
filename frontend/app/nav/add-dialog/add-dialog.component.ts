import {Component,
        OnInit} from '@angular/core';
import {FormControl,
        FormGroup,
        Validators} from '@angular/forms';
import {MatDialogRef} from '@angular/material';
import {MutateService} from 'frontend/app/services/mutate.service';

// This is just to let users know they need an HTTP/S url
const FEED_URL_PATTERN = /^https?:\/\//i;

@Component({
  selector: 'awrss-add-dialog',
  templateUrl: './add-dialog.component.html',
  styleUrls: ['./add-dialog.component.scss']
})
export class AddDialogComponent {
  public feedUrl = new FormControl('', [
    Validators.required, Validators.pattern(FEED_URL_PATTERN)
  ]);
  public title: string;

  constructor(
      public readonly dialogRef: MatDialogRef<AddDialogComponent>,
      private readonly mutateService: MutateService) {}


  public submitFeed() {
    this.mutateService
        .newFeed(this.feedUrl.value, this.title, true)
        .subscribe(() => this.dialogRef.close());
  }
}
