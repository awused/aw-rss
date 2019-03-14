import {Component,
        OnInit} from '@angular/core';
import {DataService} from 'frontend/app/services/data.service';

@Component({
  selector: 'app-item-list',
  templateUrl: './item-list.component.html',
  styleUrls: ['./item-list.component.scss']
})
export class ItemListComponent implements OnInit {
  constructor(private readonly dataService: DataService) {
  }

  ngOnInit() {
  }
}
