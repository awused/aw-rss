import {Component,
        Inject} from '@angular/core';
import {
  AbstractControl,
  FormControl,
  FormGroup,
  Validators
} from '@angular/forms';
import {MAT_DIALOG_DATA,
        MatDialogRef} from '@angular/material/dialog';
import {MatSnackBar} from '@angular/material/snack-bar';
import {FilteredData} from 'frontend/app/models/data';
import {Category} from 'frontend/app/models/entities';
import {DataService} from 'frontend/app/services/data.service';
import {MutateService} from 'frontend/app/services/mutate.service';

export const CATEGORY_NAME_PATTERN = /^[a-z][a-z0-9-]+$/;

@Component({
    selector: 'awrss-edit-category-dialog',
    templateUrl: './edit-category-dialog.component.html',
    styleUrls: ['./edit-category-dialog.component.scss'],
    standalone: false
})
export class EditCategoryDialogComponent {
  public readonly category: Category;
  public readonly initialVisibility: string;
  public categoryForm: FormGroup;
  public categoryNames: ReadonlySet<string> = new Set();

  constructor(
      private readonly dialogRef: MatDialogRef<EditCategoryDialogComponent>,
      private readonly dataService: DataService,
      private readonly mutateService: MutateService,
      private readonly snackBar: MatSnackBar,
      @Inject(MAT_DIALOG_DATA) public readonly data: {
        category: Category;
      }) {
    this.category = data.category;

    let visibility = 'show';
    if (this.category.hiddenNav) {
      visibility = 'hiddenNav';
    } else if (this.category.hiddenMain) {
      visibility = 'hiddenMain';
    }
    this.initialVisibility = visibility;

    this.categoryForm = new FormGroup({
      name: new FormControl(this.category.name, [
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
      title: new FormControl(this.category.title, [Validators.required]),
      visibility: new FormControl(visibility),
    });

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
                        .filter((c) => c.id !== this.category.id)
                        .map((c) => c.name)));
  }

  public isUnchanged(): boolean {
    if (this.category.name !== this.categoryForm.get('name')?.value) {
      return false;
    }

    if (this.category.title !== this.categoryForm.get('title')?.value) {
      return false;
    }

    if (this.initialVisibility !== this.categoryForm.get('visibility')?.value) {
      return false;
    }

    return true;
  }

  public submit(formValue: any) {
    const edit: any = {};


    if (this.category.name !== formValue.name) {
      edit.name = formValue.name;
    }

    if (this.category.title !== formValue.title) {
      edit.title = formValue.title;
    }

    if (this.initialVisibility !== formValue.visibility) {
      edit.hiddenNav = false;
      edit.hiddenMain = false;

      if (formValue.visibility === 'hiddenNav') {
        edit.hiddenNav = true;
      } else if (formValue.visibility === 'hiddenMain') {
        edit.hiddenMain = true;
      }
    }

    this.mutateService
        .editCategory(this.category, edit)
        .subscribe(() => {
          this.snackBar.open(
              `Edited Category [${this.category.title}]`, '', {
                duration: 3000
              });
          this.dialogRef.close();
        });
  }
}
