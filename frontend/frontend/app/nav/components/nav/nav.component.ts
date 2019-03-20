import {Component,
        Input,
        OnInit} from '@angular/core';
import {RefreshService} from 'frontend/app/services/refresh.service';

@Component({
  selector: 'awrss-nav',
  templateUrl: './nav.component.html',
  styleUrls: ['./nav.component.scss']
})
export class NavComponent implements OnInit {
  @Input()
  public showHeader: boolean;

  constructor(
      private readonly refreshService: RefreshService) {}

  ngOnInit() {
  }

  public isRefreshing(): boolean {
    return this.refreshService.isRefreshing();
  }

  public refresh() {
    this.refreshService.startRefresh();
  }
}
