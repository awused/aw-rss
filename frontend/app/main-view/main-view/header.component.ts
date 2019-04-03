import {Component,
        Input,
        OnInit} from '@angular/core';
import {Category,
        Feed} from 'frontend/app/models/entities';

@Component({
  selector: 'awrss-main-view-header',
  templateUrl: './header.component.html',
  styleUrls: ['./header.component.scss']
})
export class MainViewHeaderComponent implements OnInit {
  @Input()
  public feed?: Feed;
  @Input()
  public category?: Category;
  @Input()
  public mobile: boolean;

  constructor() {}

  ngOnInit() {
  }
}
