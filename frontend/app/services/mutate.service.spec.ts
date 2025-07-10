import { provideHttpClientTesting } from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';

import {MutateService} from './mutate.service';
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';

describe('MutateService', () => {
  beforeEach(() => TestBed.configureTestingModule({
    imports: [],
    providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting()]
}));

  it('should be created', () => {
    const service: MutateService = TestBed.inject(MutateService);
    expect(service).toBeTruthy();
  });
});
