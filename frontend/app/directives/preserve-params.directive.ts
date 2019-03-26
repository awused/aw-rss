import {Directive,
        OnDestroy,
        OnInit} from '@angular/core';
import {ActivatedRoute,
        RouterLinkWithHref} from '@angular/router';
import {Subject} from 'rxjs';
import {takeUntil} from 'rxjs/operators';

@Directive({
  // This is adequate as long as I don't programmatically nagivate users
  // during normal operation.
  // TODO -- Decide how this feels and selectively disable it when necessary
  // See https://github.com/angular/angular/issues/12664
  // Use [queryParamsHandling]="''" to disable it
  selector: 'a[routerLink]:not([queryParamsHandling])'
})
export class PreserveParamsDirective implements OnInit, OnDestroy {
  private readonly onDestroy$: Subject<void> = new Subject();

  constructor(
      private readonly link: RouterLinkWithHref,
      private readonly route: ActivatedRoute) {}

  ngOnInit() {
    this.route.queryParamMap
        .pipe(takeUntil(this.onDestroy$))
        .subscribe(queryParams => {
          this.link.queryParams = Object.assign({}, this.route.snapshot.queryParams);
        });
  }

  ngOnDestroy() {
    this.onDestroy$.next();
  }
}
