import {TestBed} from '@angular/core/testing';
import {DomSanitizer} from '@angular/platform-browser';

import {UrlSanitizePipe} from './url-sanitize.pipe';

describe('UrlSanitizePipe', () => {
  let pipe: UrlSanitizePipe;

  beforeEach(() => {
    TestBed.configureTestingModule({});

    pipe = new UrlSanitizePipe(TestBed.get(DomSanitizer));
  });

  it('create an instance', () => {
    expect(pipe).toBeTruthy();
  });
});
