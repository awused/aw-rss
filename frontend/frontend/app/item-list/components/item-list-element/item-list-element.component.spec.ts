import {Component} from '@angular/core';
import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {ItemFixtures} from 'frontend/app/models/models.fake';
import {MutateService} from 'frontend/app/services/mutate.service';
import {FakeMutateService} from 'frontend/app/services/mutate.service.fake';

import {ItemListElementComponent} from './item-list-element.component';

@Component({
  selector: 'awrss-test-wrapper',
  template: '<awrss-item-list-element [item]="item"></awrss-item-list-element>'
})
class TestWrapperComponent {
  item = ItemFixtures.emptyItem;
}

describe('ItemListElementComponent', () => {
  let component: ItemListElementComponent;
  let fixture: ComponentFixture<TestWrapperComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             declarations: [
               ItemListElementComponent,
               TestWrapperComponent
             ],
             providers: [
               {provide: MutateService, useClass: FakeMutateService}
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(TestWrapperComponent);
    component = fixture.debugElement.children[0].componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
