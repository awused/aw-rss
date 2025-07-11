import {Component} from '@angular/core';
import {
  AbstractControl,
  FormControl,
  FormGroup,
  Validators
} from '@angular/forms';
import {MatDialogRef} from '@angular/material/dialog';
import {MatSnackBar} from '@angular/material/snack-bar';
import {FilteredData} from 'frontend/app/models/data';
import {DataService} from 'frontend/app/services/data.service';
import {MutateService} from 'frontend/app/services/mutate.service';

import {CATEGORY_NAME_PATTERN} from '../edit-category-dialog/edit-category-dialog.component';

// This is just to let users know they need an HTTP/S url
const FEED_URL_PATTERN = /^https?:\/\//i;

@Component({
    selector: 'awrss-add-dialog',
    templateUrl: './add-dialog.component.html',
    styleUrls: ['./add-dialog.component.scss'],
    standalone: false
})
export class AddDialogComponent {
  public categoryNames: ReadonlySet<string> = new Set();

  public feedForm = new FormGroup({
    url: new FormControl('', [
      Validators.required, Validators.pattern(FEED_URL_PATTERN)
    ]),
    title: new FormControl(''),
    force: new FormControl(false),
  });

  public categoryForm = new FormGroup({
    name: new FormControl('', [
      Validators.required,
      Validators.pattern(CATEGORY_NAME_PATTERN),
      (nameControl: AbstractControl) => {
        if (this.categoryNames.has(nameControl.value)) {
          return {
            nameTaken: {
              valid: false
            }
          };
        }
        return null;
      }
    ]),
    title: new FormControl('', [Validators.required]),
    visibility: new FormControl('')
  });

  constructor(
      private readonly dialogRef: MatDialogRef<AddDialogComponent>,
      private readonly dataService: DataService,
      private readonly mutateService: MutateService,
      private readonly snackBar: MatSnackBar) {
    // Uses take(1) internally
    this.dataService.dataForFilters({
                      validOnly: true,
                      excludeFeeds: true,
                      excludeItems: true,
                    })
        .subscribe(
            (fd: FilteredData) =>
                this.categoryNames = new Set(
                    fd.categories
                        .map((c) => c.name)));
  }


  public submitFeed(formValue: any) {
    this.mutateService
        .newFeed(formValue.url, formValue.title, formValue.force)
        .subscribe(() => {
          this.snackBar.open(`Added Feed [${formValue.title || formValue.url}]`, '', {
            duration: 3000
          });
          this.dialogRef.close();
        });
  }


  public submitCategory(formValue: any) {
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
