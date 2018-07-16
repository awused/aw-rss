import {NgModule} from '@angular/core';
import {
  MatSidenavModule,
  MatToolbarModule,
} from '@angular/material';

const modules = [
  MatSidenavModule,
  MatToolbarModule,
];

/**
 * Module only to import Material components, instead of importing them
 * everywhere.
 */
@NgModule({imports : modules, exports : modules})
export class MaterialModule {
}
