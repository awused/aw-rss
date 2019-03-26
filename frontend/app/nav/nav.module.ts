import {DragDropModule} from '@angular/cdk/drag-drop';
import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {RouterModule} from '@angular/router';

import {DirectivesModule} from '../directives/directives.module';
import {MaterialModule} from '../material/material.module';

import {FeedComponent} from './components/feed/feed.component';
import {NavComponent} from './components/nav/nav.component';

@NgModule({
  imports: [
    CommonModule,
    MaterialModule,
    RouterModule,
    DragDropModule,
    DirectivesModule,
  ],
  declarations: [NavComponent, FeedComponent],
  exports: [NavComponent],
})
export class NavModule {
}
