import {HttpErrorResponse} from '@angular/common/http';
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

  public showError(error: string|Error|HttpErrorResponse): Observable<MatSnackBarDismiss> {
    let m: string;
    if (error instanceof HttpErrorResponse &&
        typeof error.error === 'string') {
      m = `${error.statusText}: ${error.error}`;
    } else if (error instanceof Error || error instanceof HttpErrorResponse) {
      m = error.message;
    } else {
      m = error;
    }
    console.error(error);
    return this.snackBar
        .open(
            m,
            'Close',
            {
              politeness: 'assertive',
              panelClass: 'warn-bg',
            })
        .afterDismissed();
  }
}
