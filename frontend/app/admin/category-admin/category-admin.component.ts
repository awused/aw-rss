import {Component,
        OnDestroy,
        OnInit} from '@angular/core';
import {MatDialog} from '@angular/material/dialog';
import {EmptyFilteredData,
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
  styleUrls: ['./category-admin.component.scss']
})
export class CategoryAdminComponent implements OnInit, OnDestroy {
  private readonly onDestroy$: Subject<void> = new Subject();
  public filteredData: FilteredData = EmptyFilteredData;

  constructor(
      private readonly dataService: DataService,
      private readonly mutateService: MutateService,
      private readonly dialog: MatDialog) {}

  ngOnInit() {
    this.dataService.updates()
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(
            (u: Updates) => this.filteredData = this.filteredData.merge(u)[0]);

    this.dataService.dataForFilters({
                      excludeFeeds: true,
                      excludeItems: true,
                      validOnly: true,
                    })
        .pipe(takeUntil(this.onDestroy$))
        .subscribe((fd: FilteredData) => this.filteredData = fd);
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

  ngOnDestroy() {
    this.onDestroy$.next();
    this.onDestroy$.complete();
  }
}
