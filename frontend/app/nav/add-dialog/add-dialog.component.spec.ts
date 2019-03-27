import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';
import {MatDialogModule,
        MatDialogRef} from '@angular/material';
import {MutateService} from 'frontend/app/services/mutate.service';
import {FakeMutateService} from 'frontend/app/services/mutate.service.fake';

import {AddDialogComponent} from './add-dialog.component';

describe('AddDialogComponent', () => {
  let component: AddDialogComponent;
  let fixture: ComponentFixture<AddDialogComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               ReactiveFormsModule,
               FormsModule,
               MatDialogModule,
             ],
             declarations: [AddDialogComponent],
             providers: [
               // Spy on this
               {provide: MatDialogRef, useValue: {}},
               {provide: MutateService, useClass: FakeMutateService},
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AddDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
