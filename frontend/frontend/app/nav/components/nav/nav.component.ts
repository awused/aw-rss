import {Component,
        EventEmitter,
        Input,
        OnInit,
        Output} from '@angular/core';
import {EmptyFilteredData,
        FilteredData,
        Updates} from 'frontend/app/models/data';
import {DataService} from 'frontend/app/services/data.service';
import {RefreshService} from 'frontend/app/services/refresh.service';

@Component({
  selector: 'awrss-nav',
  templateUrl: './nav.component.html',
  styleUrls: ['./nav.component.scss']
})
export class NavComponent {
  @Input()
  public showHeader: boolean;
  @Output()
  public unreadCount = new EventEmitter<number>();

  get unread() {
    return this.filteredData.items.length;
  }

  private filteredData: FilteredData = EmptyFilteredData;

  // This controller will never be destroyed
  constructor(
      private readonly refreshService: RefreshService,
      private readonly dataService: DataService) {
    this.dataService.updates().subscribe(
        (u: Updates) => {
          let fd, changed;
          [fd, changed] = this.filteredData.merge(u);

          if (changed) {
            this.filteredData = fd;
            this.unreadCount.emit(this.filteredData.items.length);
          }
        });

    this.dataService.dataForFilters({
                      isNav: true,
                      validOnly: true,
                      unreadOnly: true,
                    })
        .subscribe((fd: FilteredData) => {
          this.filteredData = fd;
          this.unreadCount.emit(fd.items.length);
        });
  }


  public isRefreshing(): boolean {
    return this.refreshService.isRefreshing();
  }

  public refresh() {
    this.refreshService.startRefresh();
  }
}
