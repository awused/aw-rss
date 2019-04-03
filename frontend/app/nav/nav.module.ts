import {DragDropModule} from '@angular/cdk/drag-drop';
import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';
import {RouterModule} from '@angular/router';

import {AdminModule} from '../admin/admin.module';
import {DirectivesModule} from '../directives/directives.module';
import {MaterialModule} from '../material/material.module';
import {PipesModule} from '../pipes/pipes.module';

import {FeedComponent} from './feed/feed.component';
import {NavComponent} from './nav/nav.component';

@NgModule({
  imports: [
    CommonModule,
    MaterialModule,
    RouterModule,
    DragDropModule,
    DirectivesModule,
    PipesModule,
    FormsModule,
    ReactiveFormsModule,
    AdminModule,
  ],
  declarations: [NavComponent, FeedComponent],
  exports: [NavComponent],
})
export class NavModule {
}
