import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';

import { ItemListRoutingModule } from './item-list-routing.module';
import { ItemListElementComponent } from './components/item-list-element/item-list-element.component';
import { ItemListComponent } from './components/item-list/item-list.component';

@NgModule({
  imports: [
    CommonModule,
    ItemListRoutingModule
  ],
  declarations: [ItemListElementComponent, ItemListComponent]
})
export class ItemListModule { }
