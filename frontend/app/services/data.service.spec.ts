import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';

import {DataService} from './data.service';
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';

describe('DataService', () => {
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
    imports: [],
    providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting()]
});

    httpMock = TestBed.inject(HttpTestingController);
  });

  it('should be created', () => {
    const service: DataService = TestBed.inject(DataService);
    expect(service).toBeTruthy();
  });
});
