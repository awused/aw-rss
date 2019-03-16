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

  public showError(error: string|Error): Observable<MatSnackBarDismiss> {
    let m: string;
    if (error instanceof Error) {
      m = error.message;
    } else {
      m = error;
    }
    console.error(error);
    return this.snackBar.open(m, 'Close')
        .afterDismissed();
  }
}
