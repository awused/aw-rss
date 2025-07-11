import {Directive,
        inject} from '@angular/core';
import {RouterLinkWithHref} from '@angular/router';

@Directive({
  // This is adequate as long as I don't programmatically nagivate users
  // during normal operation.
  // TODO -- Decide how this feels and selectively disable it when necessary
  // See https://github.com/angular/angular/issues/12664
  // Use [queryParamsHandling]="''" to disable it
  // tslint:disable directive-selector
  selector: 'a[routerLink]:not([queryParamsHandling])',
  standalone: false
})
export class PreserveParamsDirective {
  private readonly link = inject(RouterLinkWithHref);

  constructor() {
    this.link.queryParamsHandling = 'merge';
  }
}
