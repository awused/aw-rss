import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA} from '@angular/material';

import {ConfirmationDialogComponent} from './confirmation-dialog.component';

describe('ConfirmationDialogComponent', () => {
  let component: ConfirmationDialogComponent;
  let fixture: ComponentFixture<ConfirmationDialogComponent>;

  beforeEach(async(() => {
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
