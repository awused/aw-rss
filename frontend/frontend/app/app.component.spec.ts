import {async,
        TestBed} from '@angular/core/testing';
import {Title} from '@angular/platform-browser';
import {RouterTestingModule} from '@angular/router/testing';

import {AppComponent} from './app.component';

describe('AppComponent', () => {
  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               RouterTestingModule
             ],
             declarations: [
               AppComponent
             ],
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
