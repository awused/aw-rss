import {Component,
        OnInit} from '@angular/core';
import {MutateService} from 'frontend/app/services/mutate.service';

@Component({
    selector: 'awrss-index',
    templateUrl: './index.component.html',
    styleUrls: ['./index.component.scss'],
    standalone: false
})
export class IndexComponent implements OnInit {
  constructor(
      private readonly mutateService: MutateService,
  ) {}

  ngOnInit() {
  }

  rerunFailingFeeds() {
    this.mutateService.rerunFailingFeeds();
  }
}
