import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';
import {RouterModule,
        Routes} from '@angular/router';

import {DirectivesModule} from '../directives/directives.module';
import {MaterialModule} from '../material/material.module';
import {PipesModule} from '../pipes/pipes.module';

import {AddDialogComponent} from './add-dialog/add-dialog.component';
import {AdminBodyComponent} from './admin-body/admin-body.component';
import {AdminHeaderComponent} from './admin-header/admin-header.component';
import {CategoryAdminComponent} from './category-admin/category-admin.component';
import {ConfirmationDialogComponent} from './confirmation-dialog/confirmation-dialog.component';
import {EditCategoryDialogComponent} from './edit-category-dialog/edit-category-dialog.component';
import {EditFeedDialogComponent} from './edit-feed-dialog/edit-feed-dialog.component';
import {FeedAdminComponent} from './feed-admin/feed-admin.component';
import {IndexComponent} from './index/index.component';


const routes: Routes = [
  {path: 'admin', children: [
     {path: 'feeds', pathMatch: 'prefix', component: FeedAdminComponent},
     {path: 'categories', pathMatch: 'prefix', component: CategoryAdminComponent},
     {path: '', pathMatch: 'full', component: IndexComponent},
   ]},
];

@NgModule({
  imports: [
    CommonModule,
    RouterModule.forChild(routes),
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
    IndexComponent,
    FeedAdminComponent,
    AdminHeaderComponent,
    AdminBodyComponent,
    CategoryAdminComponent,
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
