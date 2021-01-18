import {
  ComponentFixture,
  TestBed,
  waitForAsync
} from '@angular/core/testing';

import {AdminBodyComponent} from '../admin-body/admin-body.component';
import {AdminHeaderComponent} from '../admin-header/admin-header.component';

import {IndexComponent} from './index.component';

describe('IndexComponent', () => {
  let component: IndexComponent;
  let fixture: ComponentFixture<IndexComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
             declarations: [
               IndexComponent,
               AdminBodyComponent,
               AdminHeaderComponent,
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(IndexComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
