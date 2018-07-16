import { fakeAsync, TestBed } from '@angular/core/testing';
import { RefreshService } from './refresh.service';


describe('RefreshService', () => {
  let service: RefreshService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [RefreshService]
    });
    service = TestBed.get(RefreshService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should send events on subscription', fakeAsync(() => {
    const first = jasmine.createSpy('first');
    const second = jasmine.createSpy('second');
    service.subject().subscribe(first);

    expect(first).toHaveBeenCalled();

    service.subject().subscribe(second);

    expect(second).toHaveBeenCalled();
    expect(first.calls.count()).toBe(1);
  }));

  it('should send events on refresh', fakeAsync(() => {
    const first = jasmine.createSpy('first');
    const second = jasmine.createSpy('second');
    service.subject().subscribe(first);

    service.refresh();

    service.subject().subscribe(second);

    expect(first.calls.count()).toBe(2);
    expect(second.calls.count()).toBe(1);
  }));
});
