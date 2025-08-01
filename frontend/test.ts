// This file is required by karma.conf.js and loads recursively all the .spec and framework files

import 'zone.js/testing';

import {getTestBed} from '@angular/core/testing';
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting
} from '@angular/platform-browser-dynamic/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';

import {
  MaterialModule
} from './app/material/material.module';

declare const require: any;

// First, initialize the Angular testing environment.
getTestBed().initTestEnvironment(
    [
      BrowserDynamicTestingModule,
      MaterialModule,
      NoopAnimationsModule
    ],
    platformBrowserDynamicTesting(),
);
// Then we find all the tests.
const context = require.context('./', true, /\.spec\.ts$/);
// And load the modules.
context.keys().map(context);
