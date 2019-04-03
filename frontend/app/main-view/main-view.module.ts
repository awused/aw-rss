import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {RouterModule,
        Routes} from '@angular/router';

import {DirectivesModule} from '../directives/directives.module';
import {MaterialModule} from '../material/material.module';
import {PipesModule} from '../pipes/pipes.module';

import {ItemComponent} from './item/item.component';
import {MainViewHeaderComponent} from './main-view/header.component';
import {MainViewComponent} from './main-view/main-view.component';


// TODO -- trying to load a disabled category dumps the user back to the root
const routes: Routes = [
  {path: 'feed', children: [
     {path: ':feedId', pathMatch: 'full', component: MainViewComponent},
     {path: '', pathMatch: 'prefix', redirectTo: '/'},
   ]},
  {path: 'category', children: [
     {path: ':categoryName', pathMatch: 'full', component: MainViewComponent},
     {path: '', pathMatch: 'prefix', redirectTo: '/'},
   ]},
  // Best effort attempt to redirect /:categoryName to a category.
  {path: ':categoryName', redirectTo: '/category/:categoryName'},
  {path: '', pathMatch: 'full', component: MainViewComponent},
];

@NgModule({
  imports: [
    CommonModule,
    RouterModule.forChild(routes),
    MaterialModule,
    PipesModule,
    DirectivesModule,
  ],
  declarations: [ItemComponent, MainViewComponent, MainViewHeaderComponent],
  exports: [RouterModule]
})
export class MainViewModule {
}
