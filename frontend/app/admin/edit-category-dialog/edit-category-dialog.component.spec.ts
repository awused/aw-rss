import {
  ComponentFixture,
  TestBed,
  waitForAsync
} from '@angular/core/testing';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';
import {MAT_DIALOG_DATA,
        MatDialogModule,
        MatDialogRef} from '@angular/material/dialog';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';
import {MutateService} from 'frontend/app/services/mutate.service';
import {FakeMutateService} from 'frontend/app/services/mutate.service.fake';

import {EditCategoryDialogComponent} from './edit-category-dialog.component';

describe('EditCategoryDialogComponent', () => {
  let component: EditCategoryDialogComponent;
  let fixture: ComponentFixture<EditCategoryDialogComponent>;

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
             imports: [
               ReactiveFormsModule,
               FormsModule,
               MatDialogModule,
             ],
             declarations: [EditCategoryDialogComponent],
             providers: [
               {provide: DataService, useClass: FakeDataService},
               // Spy on this
               {provide: MatDialogRef, useValue: {}},
               {provide: MAT_DIALOG_DATA, useValue: {category: {}}},
               {provide: MutateService, useClass: FakeMutateService},
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(EditCategoryDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
