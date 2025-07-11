import {Component,
        inject,
        OnDestroy,
        OnInit} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {EMPTY_FILTERED_DATA,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {Category} from 'frontend/app/models/entities';
import {DataService} from 'frontend/app/services/data.service';
import {MutateService} from 'frontend/app/services/mutate.service';
import {Subject} from 'rxjs';
import {takeUntil} from 'rxjs/operators';

import {ConfirmationDialogComponent} from '../confirmation-dialog/confirmation-dialog.component';
import {EditCategoryDialogComponent} from '../edit-category-dialog/edit-category-dialog.component';

@Component({
  selector: 'awrss-category-admin',
  templateUrl: './category-admin.component.html',
  styleUrls: ['./category-admin.component.scss'],
  standalone: false
})
export class CategoryAdminComponent implements OnInit, OnDestroy {
  private readonly dataService = inject(DataService);
  private readonly mutateService = inject(MutateService);
  private readonly dialog = inject(MatDialog);

  private readonly onDestroy$: Subject<void> = new Subject();
  private filteredData: FilteredData = EMPTY_FILTERED_DATA;
  public sortedCategories: ReadonlyArray<Category> = [];

  constructor() {}

  ngOnInit() {
    this.dataService.updates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(
            (u: Updates) => {
              this.filteredData = this.filteredData.merge(u)[0];
              this.sortCategories(this.filteredData.categories);
            });

    this.dataService.dataForFilters({
                      excludeFeeds: true,
                      excludeItems: true,
                      validOnly: true,
                    })
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((fd: FilteredData) => {
          this.filteredData = fd;
          this.sortCategories(fd.categories);
        });
  }

  public editCategory(category: Category) {
    this.dialog.open(EditCategoryDialogComponent, {
      data: {category}
    });
  }


  public disableCategory(category: Category) {
    this.dialog.open<any, any, boolean>(ConfirmationDialogComponent, {
                 data: {
                   title: 'Confirm Action',
                   text: [
                     `Delete category
                     ${category.title}?`,
                     `This action is irreversible.`
                   ]
                 }
               })
        .beforeClosed()
        .subscribe((result) => {
          if (result) {
            this.mutateService.editCategory(category, {disabled: true});
          }
        });
  }

  public moveCategory(targetId: number, direction: 'up'|'down') {
    const categoryIds = this.sortedCategories.map((c) => c.id);

    const i = categoryIds.findIndex((cid) => cid === targetId);
    if (i < 0) {
      return;
    }

    const ni = direction === 'up' ? i - 1 : i + 1;
    if (ni < 0 || ni >= categoryIds.length) {
      return;
    }

    categoryIds.splice(i, 1);
    categoryIds.splice(ni, 0, targetId);

    this.mutateService.reorderCategories(categoryIds);
  }

  private sortCategories(categories: ReadonlyArray<Category>) {
    this.sortedCategories = categories.slice().sort((a, b) => {
      if (a.sortPosition !== undefined) {
        if (b.sortPosition === undefined) {
          return -1;
        }

        return a.sortPosition - b.sortPosition;
      }
      if (b.sortPosition !== undefined) {
        return 1;
      }

      return a.id - b.id;
    });
  }

  ngOnDestroy() {
    this.onDestroy$.next();
    this.onDestroy$.complete();
  }
}
