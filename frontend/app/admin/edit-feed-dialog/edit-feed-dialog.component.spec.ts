import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {FormsModule,
        ReactiveFormsModule} from '@angular/forms';
import {
  MAT_DIALOG_DATA,
  MatDialogModule,
  MatDialogRef
} from '@angular/material/dialog';
import {FeedFixtures} from 'frontend/app/models/models.fake';
import {PipesModule} from 'frontend/app/pipes/pipes.module';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';
import {MutateService} from 'frontend/app/services/mutate.service';
import {FakeMutateService} from 'frontend/app/services/mutate.service.fake';

import {EditFeedDialogComponent} from './edit-feed-dialog.component';

describe('EditFeedDialogComponent', () => {
  let component: EditFeedDialogComponent;
  let fixture: ComponentFixture<EditFeedDialogComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               PipesModule,
               ReactiveFormsModule,
               FormsModule,
               MatDialogModule,
             ],
             declarations: [EditFeedDialogComponent],
             providers: [
               {provide: DataService, useClass: FakeDataService},
               // Spy on this
               {provide: MatDialogRef, useValue: {}},
               {provide: MAT_DIALOG_DATA, useValue: {feed: FeedFixtures.emptyFeed}},
               {provide: MutateService, useClass: FakeMutateService},
             ]
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
