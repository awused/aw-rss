import {HttpClientModule} from '@angular/common/http';
import {NgModule} from '@angular/core';
import {BrowserModule,
        Title} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
import {ServiceWorkerModule} from '@angular/service-worker';

import {environment} from '../environments/environment';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {ItemListModule} from './item-list/item-list.module';
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
    ItemListModule,
    NavModule,
    AppRoutingModule,  // last
    MaterialModule,
    ServiceWorkerModule.register(
        '/ngsw-worker.js', {enabled: environment.production})
  ],
  entryComponents: [AppComponent],
  providers: [Title],
  bootstrap: [AppComponent]
})
export class AppModule {
}
