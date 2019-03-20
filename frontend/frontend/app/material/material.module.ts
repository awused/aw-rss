import {NgModule} from '@angular/core';
import {MatButtonModule,
        MatIconModule,
        MatSidenavModule,
        MatSnackBarModule,
        MatToolbarModule} from '@angular/material';

/**
 * Module only to import Material components, instead of importing them
 * everywhere.
 */
@NgModule({exports: [
  MatButtonModule,
  MatIconModule,
  MatSnackBarModule,
  MatSidenavModule,
  MatToolbarModule,
]})
export class MaterialModule {
}
