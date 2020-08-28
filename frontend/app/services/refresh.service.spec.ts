import {fakeAsync,
        TestBed} from '@angular/core/testing';
import {RefreshService} from './refresh.service';


describe('RefreshService', () => {
  let service: RefreshService;
  let start: jasmine.Spy;
  let finish: jasmine.Spy;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [RefreshService]
    });
    service = TestBed.inject(RefreshService);

    start = jasmine.createSpy('start');
    finish = jasmine.createSpy('finish');

    service.startedObservable().subscribe(start);
    service.finishedObservable().subscribe(finish);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should send events on normal refresh', fakeAsync(() => {
       expect(start).not.toHaveBeenCalled();
       expect(finish).not.toHaveBeenCalled();
       expect(service.isRefreshing()).toBeFalsy();

       service.startRefresh();
       expect(start).toHaveBeenCalled();
       expect(finish).not.toHaveBeenCalled();
       expect(service.isRefreshing()).toBeTruthy();

       service.finishRefresh();
       expect(finish).toHaveBeenCalled();
       expect(service.isRefreshing()).toBeFalsy();
     }));

  it('should not send unnecessary events', fakeAsync(() => {
       service.finishRefresh();
       expect(finish).not.toHaveBeenCalled();

       service.startRefresh();
       service.startRefresh();
       expect(start).toHaveBeenCalledTimes(1);


       service.finishRefresh();
       service.finishRefresh();
       expect(finish).toHaveBeenCalledTimes(1);
     }));
});
