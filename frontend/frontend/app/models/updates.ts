import {Data} from './data';

export class Updates {
  constructor(
      // If this came from a user-triggered refresh or not
      // All updates of any kind get transformed into an update object.
      public readonly refresh: boolean = false,
      public readonly data: Data = new Data(),
  ) {}
}
