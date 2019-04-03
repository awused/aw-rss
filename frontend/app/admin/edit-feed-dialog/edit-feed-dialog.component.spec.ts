import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { EditFeedDialogComponent } from './edit-feed-dialog.component';

describe('EditFeedDialogComponent', () => {
  let component: EditFeedDialogComponent;
  let fixture: ComponentFixture<EditFeedDialogComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ EditFeedDialogComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(EditFeedDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
