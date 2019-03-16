import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {DataService} from 'frontend/app/services/data.service';
import {FakeDataService} from 'frontend/app/services/data.service.fake';

import {ItemListComponent} from './item-list.component';

describe('ItemListComponent', () => {
  let component: ItemListComponent;
  let fixture: ComponentFixture<ItemListComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             declarations: [ItemListComponent],
             providers: [
               {provide: DataService, useClass: FakeDataService}
             ]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ItemListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
