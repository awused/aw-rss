import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import {ItemListElementComponent} from './components/item-list-element/item-list-element.component';

const routes: Routes = [
  { path: '', component: ItemListElementComponent }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class ItemListRoutingModule { }
