import {Component,
        inject} from '@angular/core';
import {MAT_DIALOG_DATA} from '@angular/material/dialog';

@Component({
  selector: 'awrss-confirmation-dialog',
  templateUrl: './confirmation-dialog.component.html',
  styleUrls: ['./confirmation-dialog.component.scss'],
  standalone: false
})
export class ConfirmationDialogComponent {
  readonly data = inject<{
    title: string;
    text: string | string[];
    dangerous?: boolean;
  }>(MAT_DIALOG_DATA);

  public text: string[];
  public disabled: boolean = false;

  constructor() {
    const data = this.data;

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
