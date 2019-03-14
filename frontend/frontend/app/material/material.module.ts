import {NgModule} from '@angular/core';
import {MatSidenavModule} from '@angular/material/sidenav';
import {MatSnackBarModule} from '@angular/material/snack-bar';
import {MatToolbarModule} from '@angular/material/toolbar';

const modules = [
  MatSidenavModule,
  MatToolbarModule,
  MatSnackBarModule,
];

/**
 * Module only to import Material components, instead of importing them
 * everywhere.
 */
@NgModule({imports: modules, exports: modules})
export class MaterialModule {
}
