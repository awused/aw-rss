import {HttpClientModule} from '@angular/common/http';
import {NgModule} from '@angular/core';
import {MAT_DIALOG_DEFAULT_OPTIONS} from '@angular/material';
import {BrowserModule,
        HAMMER_LOADER,
        Title} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
import {ServiceWorkerModule} from '@angular/service-worker';

import {environment} from '../environments/environment';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {DirectivesModule} from './directives/directives.module';
import {MainViewModule} from './main-view/main-view.module';
import {MaterialModule} from './material/material.module';
import {NavModule} from './nav/nav.module';
import {PipesModule} from './pipes/pipes.module';

@NgModule({
  declarations: [
    AppComponent,
  ],
  imports: [
    BrowserModule,
    // After BrowserModule
    HttpClientModule,
    BrowserAnimationsModule,
    MainViewModule,
    NavModule,
    PipesModule,
    AppRoutingModule,  // last
    MaterialModule,
    DirectivesModule,
    ServiceWorkerModule.register(
        '/ngsw-worker.js', {enabled: environment.production})
  ],
  entryComponents: [AppComponent],
  providers: [
    Title,
    {
      provide: MAT_DIALOG_DEFAULT_OPTIONS,
      useValue: {
        minWidth: '400px',
        maxWidth: '800px',
        panelClass: 'mat-typography',
        hasBackdrop: true,
        closeOnNavigation: true,
      }
    },
    {
      provide: HAMMER_LOADER,
      useValue: () => new Promise(() => {})
    }
  ],
  bootstrap: [AppComponent]
})
export class AppModule {
}
