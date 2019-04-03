import {async,
        ComponentFixture,
        TestBed} from '@angular/core/testing';
import {PipesModule} from 'frontend/app/pipes/pipes.module';

import {MainViewHeaderComponent} from './header.component';


describe('MainViewHeaderComponent', () => {
  let component: MainViewHeaderComponent;
  let fixture: ComponentFixture<MainViewHeaderComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
             imports: [
               PipesModule
             ],
             declarations: [MainViewHeaderComponent]
           })
        .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(MainViewHeaderComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
