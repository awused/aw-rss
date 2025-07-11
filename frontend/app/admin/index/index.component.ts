import {Component,
        inject} from '@angular/core';
import {MutateService} from 'frontend/app/services/mutate.service';

@Component({
  selector: 'awrss-index',
  templateUrl: './index.component.html',
  styleUrls: ['./index.component.scss'],
  standalone: false
})
export class IndexComponent {
  private readonly mutateService = inject(MutateService);

  rerunFailingFeeds() {
    this.mutateService.rerunFailingFeeds();
  }
}
