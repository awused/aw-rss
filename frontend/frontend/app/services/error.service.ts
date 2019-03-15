import {Injectable} from '@angular/core';
import {MatSnackBar,
        MatSnackBarDismiss} from '@angular/material/snack-bar';
import {Observable} from 'rxjs';


@Injectable({
  providedIn: 'root'
})
export class ErrorService {
  constructor(
      private readonly snackBar: MatSnackBar) {}

  public showError(message: string|Error): Observable<MatSnackBarDismiss> {
    console.error(message);
    return this.snackBar.open(message.toString(), 'Close')
        .afterDismissed();
  }
}
