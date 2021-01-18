import {DragDropModule} from '@angular/cdk/drag-drop';
import {
  ComponentFixture,
  TestBed,
  waitForAsync
} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {MaterialModule} from 'frontend/app/material/material.module';
import {PipesModule} from 'frontend/app/pipes/pipes.module';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';
import {MutateService} from 'frontend/app/services/mutate.service';
import {FakeMutateService} from 'frontend/app/services/mutate.service.fake';

import {FeedComponent} from '../feed/feed.component';

import {NavComponent} from './nav.component';

describe('NavComponent', () => {
  let component: NavComponent;
  let fixture: ComponentFixture<NavComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
             imports: [
               RouterTestingModule,
               DragDropModule,
               MaterialModule,
               PipesModule,
             ],
             declarations: [
               NavComponent,
               FeedComponent,
             ],
             providers: [
               {provide: DataService, useClass: FakeDataService},
               {provide: MutateService, useClass: FakeMutateService}
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NavComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
