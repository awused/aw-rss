import {HttpClientTestingModule} from '@angular/common/http/testing';
import {TestBed} from '@angular/core/testing';

import {MutateService} from './mutate.service';

describe('MutateService', () => {
  beforeEach(() => TestBed.configureTestingModule({
    imports: [HttpClientTestingModule],
  }));

  it('should be created', () => {
    const service: MutateService = TestBed.inject(MutateService);
    expect(service).toBeTruthy();
  });
});
