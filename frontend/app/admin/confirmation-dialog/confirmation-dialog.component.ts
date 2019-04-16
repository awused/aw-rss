import {Component,
        Inject,
        OnInit} from '@angular/core';
import {MAT_DIALOG_DATA} from '@angular/material';

@Component({
  selector: 'awrss-confirmation-dialog',
  templateUrl: './confirmation-dialog.component.html',
  styleUrls: ['./confirmation-dialog.component.scss']
})
export class ConfirmationDialogComponent implements OnInit {
  constructor(
      @Inject(MAT_DIALOG_DATA) public readonly data: {
        title: string,
        text: string
      }) {}

  ngOnInit() {
  }
}
