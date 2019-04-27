import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';

import {DirectivesModule} from '../directives/directives.module';
import {MaterialModule} from '../material/material.module';
import {PipesModule} from '../pipes/pipes.module';

import {AddDialogComponent} from './add-dialog/add-dialog.component';
import {ConfirmationDialogComponent} from './confirmation-dialog/confirmation-dialog.component';
import {EditCategoryDialogComponent} from './edit-category-dialog/edit-category-dialog.component';
import {EditFeedDialogComponent} from './edit-feed-dialog/edit-feed-dialog.component';

@NgModule({
  imports: [
    CommonModule,
    MaterialModule,
    DirectivesModule,
    FormsModule,
    ReactiveFormsModule,
    PipesModule,
  ],
  declarations: [
    AddDialogComponent,
    ConfirmationDialogComponent,
    EditFeedDialogComponent,
    EditCategoryDialogComponent,
  ],
  entryComponents: [
    AddDialogComponent,
    ConfirmationDialogComponent,
    EditFeedDialogComponent,
    EditCategoryDialogComponent,
  ],
  exports: [
    AddDialogComponent,
    ConfirmationDialogComponent,
    EditFeedDialogComponent,
    EditCategoryDialogComponent,
  ],
})
export class AdminModule {
}
