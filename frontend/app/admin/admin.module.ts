import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';

import {DirectivesModule} from '../directives/directives.module';
import {MaterialModule} from '../material/material.module';

import {AddDialogComponent} from './add-dialog/add-dialog.component';
import {ConfirmationDialogComponent} from './confirmation-dialog/confirmation-dialog.component';
import {EditFeedDialogComponent} from './edit-feed-dialog/edit-feed-dialog.component';

@NgModule({
  imports: [
    CommonModule,
    MaterialModule,
    DirectivesModule,
    FormsModule,
    ReactiveFormsModule,
  ],
  declarations: [
    AddDialogComponent,
    ConfirmationDialogComponent,
    EditFeedDialogComponent,
  ],
  entryComponents: [
    AddDialogComponent,
    ConfirmationDialogComponent,
    EditFeedDialogComponent,
  ],
  exports: [
    AddDialogComponent,
    ConfirmationDialogComponent,
    EditFeedDialogComponent,
  ],
})
export class AdminModule {
}