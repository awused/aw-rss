import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';

import {MaterialModule} from '../material/material.module';

import {NavComponent} from './components/nav/nav.component';

@NgModule({
  imports: [
    CommonModule,
    MaterialModule,
  ],
  declarations: [NavComponent],
  exports: [NavComponent],
})
export class NavModule {
}
