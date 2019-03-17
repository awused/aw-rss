import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {RouterModule,
        Routes} from '@angular/router';
import {ItemListElementComponent} from './components/item-list-element/item-list-element.component';
import {ItemListComponent} from './components/item-list/item-list.component';


const routes: Routes = [
  {path: '', pathMatch: 'full', component: ItemListComponent}
];

@NgModule({
  imports: [
    CommonModule,
    RouterModule.forChild(routes),
  ],
  declarations: [ItemListElementComponent, ItemListComponent],
  exports: [RouterModule]
})
export class ItemListModule {
}
