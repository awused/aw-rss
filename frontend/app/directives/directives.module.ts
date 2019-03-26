import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';

import {PreserveParamsDirective} from './preserve-params.directive';

@NgModule({
  declarations: [PreserveParamsDirective],
  exports: [PreserveParamsDirective],
  imports: [
    CommonModule
  ]
})
export class DirectivesModule {
}
