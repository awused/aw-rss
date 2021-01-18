import {
  ComponentFixture,
  TestBed,
  waitForAsync
} from '@angular/core/testing';
import {MAT_DIALOG_DATA} from '@angular/material/dialog';

import {ConfirmationDialogComponent} from './confirmation-dialog.component';

describe('ConfirmationDialogComponent', () => {
  let component: ConfirmationDialogComponent;
  let fixture: ComponentFixture<ConfirmationDialogComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
             declarations: [ConfirmationDialogComponent],
             providers: [
               {provide: MAT_DIALOG_DATA, useValue: {}}
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ConfirmationDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
