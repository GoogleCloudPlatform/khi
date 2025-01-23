/**
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { CommonModule } from '@angular/common';
import { Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatInputModule } from '@angular/material/input';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import {
  exhaustMap,
  filter,
  interval,
  map,
  ReplaySubject,
  shareReplay,
  Subject,
  switchMap,
  take,
  takeUntil,
} from 'rxjs';
import { PopupFormRequestWithClient } from 'src/app/services/popup/popup-manager';

/**
 * The request given from server side to ask something additional from user. (e.g OAuth, additionally requied text input ...etc).
 */
export interface RequestUserActionPopupRequest {
  formRequest: PopupFormRequestWithClient;
}

/**
 * The label string to submit the answer.
 * The buton will be hidden when the label is empty.
 */
const nextButtonLabel: { [key: string]: string } = {
  text: 'Submit',
  popup_redirect: '',
};

/**
 * A dialog component shown when KHI backend server to do something by the user.
 * (e.g requesting some additional input/redirecting them to ask authenticate)
 */
@Component({
  selector: 'khi-request-user-action-popup',
  standalone: true,
  templateUrl: './request-user-action-popup.component.html',
  styleUrls: ['./request-user-action-popup.component.sass'],
  imports: [
    CommonModule,
    MatButtonModule,
    MatInputModule,
    MatProgressBarModule,
  ],
})
export class RequestUserActionPopupComponent implements OnInit, OnDestroy {
  /**
   * The request from server to show the popup.
   */
  readonly formRequest: PopupFormRequestWithClient;

  /**
   * Emit on component destroying to be used for terminating subscriptions.
   */
  readonly destoroyed: Subject<void> = new Subject<void>();

  /**
   * The observable to emit the current value to be sent.
   */
  validationRequests: ReplaySubject<string> = new ReplaySubject<string>(1);

  validationError = this.validationRequests.pipe(
    exhaustMap((value) =>
      this.data.formRequest.client.validate({
        id: this.data.formRequest.id,
        value,
      }),
    ),
    map((result) => result.validationError),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  isValid = this.validationError.pipe(map((e) => e === ''));

  nextButtonText = '';

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: RequestUserActionPopupRequest,
    private readonly dialogRef: MatDialogRef<object, void>,
  ) {
    dialogRef.disableClose = true;
    this.formRequest = data.formRequest;
    this.nextButtonText = nextButtonLabel[this.formRequest.type];
  }

  ngOnInit(): void {
    if (this.formRequest.type === 'popup_redirect') {
      this.openRedirectPopup();

      // Repeat sending validation request to server every 1s.
      // The server will return no validation error when the check passed(e.g the timing when user completed OAuth authentication,..etc).
      interval(1000)
        .pipe(takeUntil(this.destoroyed))
        .subscribe(() => {
          this.validationRequests.next('');
        });

      // Redirect popup automatically closes the popup itself when the validation passes.
      this.isValid
        .pipe(
          takeUntil(this.destoroyed),
          filter((isValid) => isValid),
        )
        .subscribe(() => {
          this.onSubmit();
        });
    } else if (this.formRequest.type === 'text') {
      // Send validation requests with empty to receive the initial error mesage.
      this.validationRequests.next('');
    }
  }

  onTextAreaUpdate(event: Event) {
    const textarea = event.target as HTMLTextAreaElement;
    const inputValue = textarea.value;
    this.onUpdateInput(inputValue);
  }

  onUpdateInput(inputValue: string) {
    this.validationRequests.next(inputValue);
  }

  /**
   * Send the determined value to backend and close the dialog.
   */
  onSubmit() {
    this.validationRequests
      .pipe(
        take(1),
        switchMap((value) =>
          this.data.formRequest.client.answer({
            id: this.data.formRequest.id,
            value,
          }),
        ),
      )
      .subscribe(() => {
        this.dialogRef.close();
      });
  }

  /**
   * Open a new window with the URL specified in redirectTo option.
   *
   * @returns
   */
  openRedirectPopup() {
    if (this.formRequest.type !== 'popup_redirect') return;
    if (!this.formRequest.options.redirectTo) {
      throw new Error('The redirect target url is missing on the form request');
    }
    window.open(
      this.formRequest.options.redirectTo,
      'oauth login',
      'width=400px,height=500px',
    );
  }

  ngOnDestroy(): void {
    this.destoroyed.next();
  }
}
