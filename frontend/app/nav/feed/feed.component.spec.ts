import {DragDropModule} from '@angular/cdk/drag-drop';
import {Component} from '@angular/core';
import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {FeedFixtures} from 'frontend/app/models/models.fake';
import {PipesModule} from 'frontend/app/pipes/pipes.module';

import {FeedData} from '../nav/nav.component';

import {FeedComponent} from './feed.component';

@Component({
  selector: 'awrss-test-wrapper',
  template: '<awrss-feed [fd]="fd"></awrss-feed>'
})
class TestWrapperComponent {
  fd = new FeedData(FeedFixtures.emptyFeed);
}


describe('FeedComponent', () => {
  let component: FeedComponent;
  let fixture: ComponentFixture<TestWrapperComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               DragDropModule,
               RouterTestingModule,
               PipesModule,
             ],
             declarations: [
               FeedComponent,
               TestWrapperComponent
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
