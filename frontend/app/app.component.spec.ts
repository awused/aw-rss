import {async,
        TestBed} from '@angular/core/testing';
import {Title} from '@angular/platform-browser';
import {RouterTestingModule} from '@angular/router/testing';

import {AppComponent} from './app.component';
import {NavModule} from './nav/nav.module';
import {DataService} from './services/data.service';
import {FakeDataService} from './services/data.service.fake';
import {MutateService} from './services/mutate.service';
import {FakeMutateService} from './services/mutate.service.fake';

describe('AppComponent', () => {
  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               RouterTestingModule,
               NavModule,
             ],
             declarations: [
               AppComponent
             ],
             providers: [
               {provide: DataService, useClass: FakeDataService},
               {provide: MutateService, useClass: FakeMutateService}
             ]
           })
        .compileComponents();
  }));

  it('should create the app', async(() => {
       const fixture = TestBed.createComponent(AppComponent);
       const app = fixture.debugElement.componentInstance;
       expect(app).toBeTruthy();
     }));

  it(`should initially set the title`, async(() => {
       const fixture = TestBed.createComponent(AppComponent);
       const title: Title = TestBed.get(Title);
       expect(title.getTitle()).toEqual('Aw-RSS');
     }));
});
