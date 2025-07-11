import {Component} from '@angular/core';
import {
  TestBed,
  waitForAsync
} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';

import {PreserveParamsDirective} from './preserve-params.directive';

@Component({
    selector: 'awrss-test-wrapper',
    template: '<a routerLink="/"></a>',
    standalone: false
})
class TestWrapperComponent {
}

describe('PreserveParamsDirective', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [
        PreserveParamsDirective,
        TestWrapperComponent,
      ],
      imports: [RouterTestingModule],
    });
  });

  it('should create an instance', () => {
    // TODO -- even by this project's standards this is a weak test
    const fixture = TestBed.createComponent(TestWrapperComponent);
    const component = fixture.debugElement.children[0].componentInstance;
  });
});
