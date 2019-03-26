import {HttpClientModule} from '@angular/common/http';
import {NgModule} from '@angular/core';
import {BrowserModule,
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
    AppRoutingModule,  // last
    MaterialModule,
    DirectivesModule,
    ServiceWorkerModule.register(
        '/ngsw-worker.js', {enabled: environment.production})
  ],
  entryComponents: [AppComponent],
  providers: [Title],
  bootstrap: [AppComponent]
})
export class AppModule {
}
