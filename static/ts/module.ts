import {Main} from './components/main';
import {ItemService} from './services/item';
import {FeedService} from './services/feed';
import {ItemList} from './components/item-list';

import {NgModule, enableProdMode} from '@angular/core';
import {BrowserModule} from '@angular/platform-browser';
import {platformBrowserDynamic} from '@angular/platform-browser-dynamic';
import {HttpModule} from '@angular/http';

@NgModule({
  imports: [BrowserModule, HttpModule],
  declarations: [Main, ItemList],
  providers: [FeedService, ItemService],
  bootstrap: [Main]
})

class Module {
}

const platform = platformBrowserDynamic();
if (!window.location.pathname.startsWith('/dev/')) {
  enableProdMode();
}
platform.bootstrapModule(Module);

