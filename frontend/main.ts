import {enableProdMode} from '@angular/core';
import {platformBrowserDynamic} from '@angular/platform-browser-dynamic';

import {AppModule} from './app/app.module';
import {environment} from './environments/environment';

if (environment.production) {
  enableProdMode();
}

platformBrowserDynamic().bootstrapModule(AppModule).catch(err => console.log(err));

// import {ɵrenderComponent as renderComponent} from '@angular/core';
//
// import {AppComponent} from './app/app.component';
//
// renderComponent(AppComponent);
