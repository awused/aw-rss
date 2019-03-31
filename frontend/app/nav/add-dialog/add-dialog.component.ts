import {Component,
        OnInit} from '@angular/core';
import {FormControl,
        FormGroup,
        Validators} from '@angular/forms';
import {MatDialogRef,
        MatSnackBar} from '@angular/material';
import {MutateService} from 'frontend/app/services/mutate.service';

// This is just to let users know they need an HTTP/S url
const FEED_URL_PATTERN = /^https?:\/\//i;
const CATEGORY_NAME_PATTERN = /^[a-z][a-z0-9-]+$/;

@Component({
  selector: 'awrss-add-dialog',
  templateUrl: './add-dialog.component.html',
  styleUrls: ['./add-dialog.component.scss']
})
export class AddDialogComponent {
  public feedForm = new FormGroup({
    url: new FormControl('', [
      Validators.required, Validators.pattern(FEED_URL_PATTERN)
    ]),
    title: new FormControl('')
  });

  public categoryForm = new FormGroup({
    name: new FormControl('', [
      Validators.required, Validators.pattern(CATEGORY_NAME_PATTERN)
    ]),
    title: new FormControl('', [Validators.required]),
    visibility: new FormControl('')
  });

  constructor(
      private readonly dialogRef: MatDialogRef<AddDialogComponent>,
      private readonly mutateService: MutateService,
      private readonly snackBar: MatSnackBar) {}


  public submitFeed(formValue) {
    this.mutateService
        .newFeed(formValue.url, formValue.title, true)
        .subscribe(() => {
          this.snackBar.open(`Added Feed [${formValue.title || formValue.url}]`, '', {
            duration: 3000
          });
          this.dialogRef.close();
        });
  }


  public submitCategory(formValue) {
    const req = {
      name: formValue.name,
      title: formValue.title,
      hiddenNav: formValue.visibility === 'hiddenNav',
      hiddenMain: formValue.visibility === 'hiddenMain'
    };
    this.mutateService
        .newCategory(req)
        .subscribe(() => {
          this.snackBar.open(`Added Category [${formValue.title}]`, '', {
            duration: 3000
          });
          this.dialogRef.close();
        });
  }
}
