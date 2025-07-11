import {inject,
        Pipe,
        PipeTransform,
        SecurityContext} from '@angular/core';
import {DomSanitizer,
        SafeUrl} from '@angular/platform-browser';

const SAFE_ITEM_URL_PATTERN = /^(?:(?:https?|magnet):|[^&:/?#]*(?:[/?#]|$))/gi;

@Pipe({
  name: 'urlSanitize',
  standalone: false
})
export class UrlSanitizePipe implements PipeTransform {
  private readonly domSanitizer = inject(DomSanitizer);

  constructor() {}

  transform(value: string, _args?: void): SafeUrl|null {
    if (value.match(SAFE_ITEM_URL_PATTERN)) {
      return this.domSanitizer.bypassSecurityTrustUrl(value);
    }

    return this.domSanitizer.sanitize(SecurityContext.URL, value);
  }
}
