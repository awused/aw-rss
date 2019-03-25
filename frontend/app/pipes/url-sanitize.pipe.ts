import {Pipe,
        PipeTransform,
        SecurityContext} from '@angular/core';
import {DomSanitizer,
        SafeUrl} from '@angular/platform-browser';

const SAFE_ITEM_URL_PATTERN = /^(?:(?:https?|magnet):|[^&:/?#]*(?:[/?#]|$))/gi;

@Pipe({
  name: 'urlSanitize'
})
export class UrlSanitizePipe implements PipeTransform {
  constructor(private readonly domSanitizer: DomSanitizer) {}

  transform(value: string, args?: void): SafeUrl {
    if (value.match(SAFE_ITEM_URL_PATTERN)) {
      return this.domSanitizer.bypassSecurityTrustUrl(value);
    }

    return this.domSanitizer.sanitize(SecurityContext.URL, value);
  }
}
