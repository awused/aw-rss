import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';
import {UrlSanitizePipe} from './url-sanitize.pipe';

@NgModule({
  declarations: [
    UrlSanitizePipe
  ],
  imports: [
    CommonModule
  ],
  exports: [
    UrlSanitizePipe
  ]
})
export class PipesModule {
}
