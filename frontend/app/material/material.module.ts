import {ScrollingModule} from '@angular/cdk/scrolling';
import {NgModule} from '@angular/core';
import {MatButtonModule,
        MatDialogModule,
        MatExpansionModule,
        MatFormFieldModule,
        MatIconModule,
        MatInputModule,
        MatSelectModule,
        MatSidenavModule,
        MatSnackBarModule,
        MatTabsModule,
        MatToolbarModule,
        MatTooltipModule} from '@angular/material';

/**
 * Module only to import Material components, instead of importing them
 * everywhere.
 */
@NgModule({exports: [
  MatButtonModule,
  MatDialogModule,
  MatExpansionModule,
  MatFormFieldModule,
  MatIconModule,
  MatInputModule,
  MatSelectModule,
  MatSidenavModule,
  MatSnackBarModule,
  MatTabsModule,
  MatToolbarModule,
  MatTooltipModule,
  ScrollingModule,
]})
export class MaterialModule {
}
