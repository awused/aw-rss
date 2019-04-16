import {CommonModule} from '@angular/common';
import {NgModule} from '@angular/core';

import {FeedTitlePipe} from './feed-title.pipe';
import {UrlSanitizePipe} from './url-sanitize.pipe';

@NgModule({
  declarations: [
    UrlSanitizePipe,
    FeedTitlePipe
  ],
  imports: [
    CommonModule
  ],
  exports: [
    UrlSanitizePipe,
    FeedTitlePipe
  ],
  providers: [
    FeedTitlePipe
  ]
})
export class PipesModule {
}
