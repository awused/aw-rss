import { TestBed } from '@angular/core/testing';

import { MobileService } from './mobile.service';

describe('MobileService', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('should be created', () => {
    const service: MobileService = TestBed.get(MobileService);
    expect(service).toBeTruthy();
  });
});
