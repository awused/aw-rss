import {DragDropModule} from '@angular/cdk/drag-drop';
import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {MaterialModule} from 'frontend/app/material/material.module';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';

import {FeedComponent} from '../feed/feed.component';

import {NavComponent} from './nav.component';

describe('NavComponent', () => {
  let component: NavComponent;
  let fixture: ComponentFixture<NavComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               RouterTestingModule,
               DragDropModule,
               MaterialModule
             ],
             declarations: [
               NavComponent,
               FeedComponent,
             ],
             providers: [
               {provide: DataService, useClass: FakeDataService}
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
