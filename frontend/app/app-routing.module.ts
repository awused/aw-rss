import {NgModule} from '@angular/core';
import {
  RouteReuseStrategy,
  RouterModule,
  Routes
} from '@angular/router';

const routes: Routes = [
  {
    path: '**',
    redirectTo: '',
  }
];

@NgModule({
  imports: [RouterModule.forRoot(routes, {scrollPositionRestoration: 'enabled'})],
  exports: [RouterModule],
})
export class AppRoutingModule {
}
