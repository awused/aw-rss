import {DragDropModule} from '@angular/cdk/drag-drop';
import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {RouterModule} from '@angular/router';

import {MaterialModule} from '../material/material.module';

import {FeedComponent} from './components/feed/feed.component';
import {NavComponent} from './components/nav/nav.component';

@NgModule({
  imports: [
    CommonModule,
    MaterialModule,
    RouterModule,
    DragDropModule,
  ],
  declarations: [NavComponent, FeedComponent],
  exports: [NavComponent],
})
export class NavModule {
}
