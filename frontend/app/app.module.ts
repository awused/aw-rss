import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import {NgModule} from '@angular/core';
import {MAT_DIALOG_DEFAULT_OPTIONS} from '@angular/material/dialog';
import {BrowserModule,
        HAMMER_LOADER,
        Title} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';
import {ServiceWorkerModule} from '@angular/service-worker';

import {environment} from '../environments/environment';

import {AdminModule} from './admin/admin.module';
import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {DirectivesModule} from './directives/directives.module';
import {MainViewModule} from './main-view/main-view.module';
import {MaterialModule} from './material/material.module';
import {NavModule} from './nav/nav.module';
import {PipesModule} from './pipes/pipes.module';

@NgModule({ declarations: [
        AppComponent,
    ],
    bootstrap: [AppComponent], imports: [BrowserModule,
        BrowserAnimationsModule,
        // Before MainView
        AdminModule,
        MainViewModule,
        NavModule,
        PipesModule,
        AppRoutingModule, // last
        MaterialModule,
        DirectivesModule,
        ServiceWorkerModule.register('/ngsw-worker.js', { enabled: environment.production })], providers: [
        Title,
        {
            provide: MAT_DIALOG_DEFAULT_OPTIONS,
            useValue: {
                minWidth: '360px',
                maxWidth: '800px',
                panelClass: 'mat-typography',
                hasBackdrop: true,
                closeOnNavigation: true,
            }
        },
        {
            provide: HAMMER_LOADER,
            useValue: () => new Promise(() => { })
        },
        provideHttpClient(withInterceptorsFromDi())
    ] })
export class AppModule {
}
