import {ComponentFixture,
        TestBed,
        waitForAsync} from '@angular/core/testing';

import {AdminBodyComponent} from './admin-body.component';

describe('AdminBodyComponent', () => {
  let component: AdminBodyComponent;
  let fixture: ComponentFixture<AdminBodyComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
             declarations: [AdminBodyComponent]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AdminBodyComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
