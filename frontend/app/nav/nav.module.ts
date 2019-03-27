import {DragDropModule} from '@angular/cdk/drag-drop';
import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';
import {RouterModule} from '@angular/router';

import {DirectivesModule} from '../directives/directives.module';
import {MaterialModule} from '../material/material.module';

import {AddDialogComponent} from './add-dialog/add-dialog.component';
import {FeedComponent} from './feed/feed.component';
import {NavComponent} from './nav/nav.component';

@NgModule({
  imports: [
    CommonModule,
    MaterialModule,
    RouterModule,
    DragDropModule,
    DirectivesModule,
    FormsModule,
    ReactiveFormsModule,
  ],
  declarations: [NavComponent, FeedComponent, AddDialogComponent],
  entryComponents: [AddDialogComponent],
  exports: [NavComponent],
})
export class NavModule {
}
