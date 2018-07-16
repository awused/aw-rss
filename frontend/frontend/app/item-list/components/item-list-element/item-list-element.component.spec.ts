import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ItemListElementComponent } from './item-list-element.component';

describe('ItemListElementComponent', () => {
  let component: ItemListElementComponent;
  let fixture: ComponentFixture<ItemListElementComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ItemListElementComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ItemListElementComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
