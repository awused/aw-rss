import {HttpClientTestingModule,
        HttpTestingController} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';

import {DataService} from './data.service';

describe('DataService', () => {
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
    });

    httpMock = TestBed.inject(HttpTestingController);
  });

  it('should be created', () => {
    const service: DataService = TestBed.inject(DataService);
    expect(service).toBeTruthy();
  });
});
