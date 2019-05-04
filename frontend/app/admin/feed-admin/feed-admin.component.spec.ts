import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {PipesModule} from 'frontend/app/pipes/pipes.module';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';

import {AdminBodyComponent} from '../admin-body/admin-body.component';
import {AdminHeaderComponent} from '../admin-header/admin-header.component';

import {FeedAdminComponent} from './feed-admin.component';

describe('FeedAdminComponent', () => {
  let component: FeedAdminComponent;
  let fixture: ComponentFixture<FeedAdminComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               PipesModule,
               RouterTestingModule,
             ],
             declarations: [
               FeedAdminComponent,
               AdminBodyComponent,
               AdminHeaderComponent,
             ],
             providers: [
               {provide: DataService, useClass: FakeDataService},
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(FeedAdminComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
