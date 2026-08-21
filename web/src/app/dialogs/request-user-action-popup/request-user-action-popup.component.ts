/**
 * Copyright 2026 Google LLC
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
import { Component, computed, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PopupRegistry } from 'src/app/dialogs/request-user-action-popup/popup-registry.service';
import { PopupContentInputs } from 'src/app/dialogs/request-user-action-popup/popup-content-component';
import { PopupFormWithClient } from 'src/app/services/popup/popup-manager';

/**
 * The request passed into RequestUserActionPopupComponent dialog data.
 */
export interface RequestUserActionPopupRequest {
  readonly formRequest: PopupFormWithClient;
}

/**
 * A dialog container component shown when the KHI backend server requests user interaction.
 */
@Component({
  selector: 'khi-request-user-action-popup',
  standalone: true,
  templateUrl: './request-user-action-popup.component.html',
  styleUrls: ['./request-user-action-popup.component.scss'],
  imports: [CommonModule],
})
export class RequestUserActionPopupComponent {
  readonly data = inject<RequestUserActionPopupRequest>(MAT_DIALOG_DATA);
  private readonly dialogRef = inject<MatDialogRef<object, void>>(MatDialogRef);
  private readonly popupRegistry = inject(PopupRegistry);

  /**
   * The popup form data and its client.
   */
  readonly formRequest: PopupFormWithClient = this.data.formRequest;

  /**
   * Dynamically resolved child content component from the registry.
   */
  readonly contentComponent = computed(() => {
    const caseName = this.formRequest.form.payload.case;
    return caseName ? this.popupRegistry.getComponent(caseName) : undefined;
  });

  /**
   * Inputs passed to the dynamic child component.
   */
  readonly contentInputs = computed<PopupContentInputs>(() => ({
    form: this.formRequest.form,
    client: this.formRequest.client,
    onComplete: () => this.onCompleted(),
  }));

  constructor() {
    this.dialogRef.disableClose = true;
  }

  /**
   * Closes the dialog when child component interaction finishes.
   */
  onCompleted(): void {
    this.dialogRef.close();
  }
}
