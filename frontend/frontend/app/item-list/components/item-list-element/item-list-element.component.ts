import {Component,
        Input,
        OnChanges,
        OnDestroy,
        OnInit,
        SimpleChanges} from '@angular/core';
import {Item} from 'frontend/app/models/entities';
import {MutateService} from 'frontend/app/services/mutate.service';
import {Subject} from 'rxjs';

@Component({
  selector: 'awrss-item-list-element',
  templateUrl: './item-list-element.component.html',
  styleUrls: ['./item-list-element.component.scss']
})
export class ItemListElementComponent implements OnInit, OnDestroy {
  @Input()
  item: Item;

  private readonly onDestroy$: Subject<void> = new Subject();

  constructor(private readonly mutateService: MutateService) {}


  ngOnInit() {
  }

  ngOnDestroy() {
    this.onDestroy$.next();
  }
}
