import {HttpErrorResponse} from '@angular/common/http';
import {inject,
        Injectable} from '@angular/core';
import {MatSnackBar,
        MatSnackBarDismiss} from '@angular/material/snack-bar';
import {Observable} from 'rxjs';


@Injectable({
  providedIn: 'root'
})
export class ErrorService {
  private readonly snackBar = inject(MatSnackBar);

  constructor() {}

  public showError(error: string|Error|HttpErrorResponse): Observable<MatSnackBarDismiss> {
    let m: string;
    if (error instanceof HttpErrorResponse &&
        typeof error.error === 'string' &&
        error.error.indexOf('<html>') === -1) {
      m = `${error.status}: ${error.error}`;
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
