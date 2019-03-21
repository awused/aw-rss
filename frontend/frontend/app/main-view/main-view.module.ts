import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {RouterModule,
        Routes} from '@angular/router';

import {MaterialModule} from '../material/material.module';
import {PipesModule} from '../pipes/pipes.module';

import {ItemComponent} from './components/item/item.component';
import {MainViewComponent} from './components/main-view/main-view.component';


// TODO -- trying to load a disabled category dumps the user back to the root
const routes: Routes = [
  {path: 'feed', children: [
     {path: ':feedid', pathMatch: 'full', component: MainViewComponent},
     {path: '', pathMatch: 'prefix', redirectTo: '/'},
   ]},
  {path: '', pathMatch: 'full', component: MainViewComponent},
];

@NgModule({
  imports: [
    CommonModule,
    RouterModule.forChild(routes),
    MaterialModule,
    PipesModule,
  ],
  declarations: [ItemComponent, MainViewComponent],
  exports: [RouterModule]
})
export class MainViewModule {
}
