import {NgModule} from '@angular/core';
import {MatButtonModule} from '@angular/material';
import {MatSidenavModule} from '@angular/material/sidenav';
import {MatSnackBarModule} from '@angular/material/snack-bar';
import {MatToolbarModule} from '@angular/material/toolbar';

const modules = [
  MatButtonModule,
  MatSnackBarModule,
  MatSidenavModule,
  MatToolbarModule,
];

/**
 * Module only to import Material components, instead of importing them
 * everywhere.
 */
@NgModule({exports: modules})
export class MaterialModule {
}
