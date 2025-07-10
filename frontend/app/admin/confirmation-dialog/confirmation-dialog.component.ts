import {Component,
        Inject} from '@angular/core';
import {MAT_DIALOG_DATA} from '@angular/material/dialog';

@Component({
  selector: 'awrss-confirmation-dialog',
  templateUrl: './confirmation-dialog.component.html',
  styleUrls: ['./confirmation-dialog.component.scss']
})
export class ConfirmationDialogComponent {
  public text: string[];
  public disabled: boolean = false;

  constructor(
      @Inject(MAT_DIALOG_DATA) public readonly data: {
        title: string;
        text: string | string[];
        dangerous?: boolean;
      }) {
    if (typeof data.text === 'string') {
      this.text = [data.text];
    } else {
      this.text = data.text;
    }

    if (data.dangerous) {
      this.disabled = true;
      setTimeout(() => this.disabled = false, 2000);
    }
  }
}
