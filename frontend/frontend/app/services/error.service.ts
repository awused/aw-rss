import {Injectable} from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';


@Injectable({
  providedIn: 'root'
})
export class ErrorService {
  constructor(
      private readonly snackBar: MatSnackBar) {}

  public showError(message: string|Error) {
    console.error(message);
    this.snackBar.open(message.toString(), 'Close');
  }
}
