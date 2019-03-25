import {ScrollingModule} from '@angular/cdk/scrolling';
import {NgModule} from '@angular/core';
import {MatButtonModule,
        MatExpansionModule,
        MatIconModule,
        MatSidenavModule,
        MatSnackBarModule,
        MatToolbarModule,
        MatTooltipModule} from '@angular/material';

/**
 * Module only to import Material components, instead of importing them
 * everywhere.
 */
@NgModule({exports: [
  MatButtonModule,
  MatExpansionModule,
  MatIconModule,
  MatSnackBarModule,
  MatSidenavModule,
  MatToolbarModule,
  MatTooltipModule,
  ScrollingModule,
]})
export class MaterialModule {
}
