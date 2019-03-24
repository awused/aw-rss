import {Component,
        Input,
        OnInit} from '@angular/core';
import {FeedData} from '../nav/nav.component';


@Component({
  selector: 'awrss-feed',
  templateUrl: './feed.component.html',
  styleUrls: ['./feed.component.scss']
})
export class FeedComponent {
  @Input()
  public fd: FeedData;

  constructor() {}
}
