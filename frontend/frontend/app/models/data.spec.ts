import {inject,
        TestBed} from '@angular/core/testing';
import {Data} from './data';

describe('Data', () => {
  beforeEach(() => {});

  it('should merge in trivial changes', () => {
    let d = new Data();
    expect(d.categories.length).toBe(0);
  });
});
