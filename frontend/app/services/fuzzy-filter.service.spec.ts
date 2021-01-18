import { TestBed } from '@angular/core/testing';

import { FuzzyFilterService } from './fuzzy-filter.service';

describe('FuzzyFilterService', () => {
  let service: FuzzyFilterService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(FuzzyFilterService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
