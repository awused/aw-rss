import {Component} from '@angular/core';
import {
  ComponentFixture,
  TestBed,
  waitForAsync
} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {ItemFixtures} from 'frontend/app/models/models.fake';
import {PipesModule} from 'frontend/app/pipes/pipes.module';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';
import {MutateService} from 'frontend/app/services/mutate.service';
import {FakeMutateService} from 'frontend/app/services/mutate.service.fake';

import {ItemComponent} from './item.component';

@Component({
  selector: 'awrss-test-wrapper',
  template: '<awrss-item [item]="item"></awrss-item>'
})
class TestWrapperComponent {
  item = ItemFixtures.emptyItem;
}

describe('ItemComponent', () => {
  let component: ItemComponent;
  let fixture: ComponentFixture<TestWrapperComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
             imports: [
               RouterTestingModule,
               PipesModule
             ],
             declarations: [
               ItemComponent,
               TestWrapperComponent
             ],
             providers: [
               {provide: MutateService, useClass: FakeMutateService},
               {provide: DataService, useClass: FakeDataService}
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
