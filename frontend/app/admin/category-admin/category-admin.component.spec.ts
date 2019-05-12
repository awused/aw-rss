import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';
import {MutateService} from 'frontend/app/services/mutate.service';
import {FakeMutateService} from 'frontend/app/services/mutate.service.fake';

import {AdminBodyComponent} from '../admin-body/admin-body.component';
import {AdminHeaderComponent} from '../admin-header/admin-header.component';

import {CategoryAdminComponent} from './category-admin.component';

describe('CategoryAdminComponent', () => {
  let component: CategoryAdminComponent;
  let fixture: ComponentFixture<CategoryAdminComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               RouterTestingModule,
             ],
             declarations: [
               CategoryAdminComponent,
               AdminBodyComponent,
               AdminHeaderComponent,
             ],
             providers: [
               {provide: DataService, useClass: FakeDataService},
               {provide: MutateService, useClass: FakeMutateService},
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(CategoryAdminComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
