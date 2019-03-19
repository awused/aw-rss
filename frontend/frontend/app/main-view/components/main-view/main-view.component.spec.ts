import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {PipesModule} from 'frontend/app/pipes/pipes.module';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';

import {ItemComponent} from '../item/item.component';

import {MainViewComponent} from './main-view.component';

describe('MainViewComponent', () => {
  let component: MainViewComponent;
  let fixture: ComponentFixture<MainViewComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               RouterTestingModule,
               PipesModule
             ],
             declarations: [
               MainViewComponent,
               ItemComponent
             ],
             providers: [
               {provide: DataService, useClass: FakeDataService}
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(MainViewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
