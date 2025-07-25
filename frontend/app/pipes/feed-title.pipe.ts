import {Pipe,
        PipeTransform} from '@angular/core';

import {Feed} from '../models/entities';

@Pipe({
    name: 'feedTitle',
    standalone: false
})
export class FeedTitlePipe implements PipeTransform {
  transform(feed: Feed, _args?: any): string {
    return feed.userTitle || feed.title || feed.siteUrl || feed.url;
  }
}
