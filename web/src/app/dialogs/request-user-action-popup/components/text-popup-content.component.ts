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
import {
  Component,
  OnInit,
  computed,
  input,
  output,
  signal,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { TextFieldModule } from '@angular/cdk/text-field';
import { Client } from '@connectrpc/connect';
import { PopupForm, PopupService } from 'src/app/generated/api/v1/popup_pb';

/**
 * TextPopupContentComponent renders and manages the interactive text input popup content.
 */
@Component({
  selector: 'khi-text-popup-content',
  standalone: true,
  templateUrl: './text-popup-content.component.html',
  styleUrls: ['./text-popup-content.component.scss'],
  imports: [CommonModule, MatButtonModule, MatInputModule, TextFieldModule],
})
export class TextPopupContentComponent implements OnInit {
  /** Active popup form configuration. */
  readonly form = input.required<PopupForm>();

  /** Connect-RPC client for popup operations. */
  readonly client = input.required<Client<typeof PopupService>>();

  /** Optional callback to notify container when interaction completes. */
  readonly onComplete = input<() => void>();

  /** Notifies the container when answer submission completes successfully. */
  readonly completed = output<void>();

  readonly inputValue = signal<string>('');
  readonly validationError = signal<string>('');
  readonly isValid = computed(() => this.validationError() === '');
  readonly isSubmitting = signal<boolean>(false);
  private validationRequestId = 0;

  readonly placeholder = computed(() => {
    const payload = this.form().payload;
    return payload.case === 'text' ? payload.value.placeholder : '';
  });

  async ngOnInit(): Promise<void> {
    await this.runValidation();
  }

  /**
   * Handles user input updates in the textarea and re-validates.
   */
  async onTextInput(event: Event): Promise<void> {
    const textarea = event.target as HTMLTextAreaElement;
    this.inputValue.set(textarea.value);
    await this.runValidation();
  }

  /**
   * Submits the finalized text answer to the backend.
   */
  async onSubmit(): Promise<void> {
    if (this.isSubmitting() || !this.isValid()) {
      return;
    }
    this.isSubmitting.set(true);
    try {
      await this.client().submitPopupAnswer({
        id: this.form().id,
        payload: {
          case: 'text',
          value: { value: this.inputValue() },
        },
      });
      this.completed.emit();
      this.onComplete()?.();
    } finally {
      this.isSubmitting.set(false);
    }
  }

  private async runValidation(): Promise<void> {
    const requestId = ++this.validationRequestId;
    const res = await this.client().validatePopupAnswer({
      id: this.form().id,
      payload: {
        case: 'text',
        value: { value: this.inputValue() },
      },
    });
    if (requestId === this.validationRequestId) {
      this.validationError.set(res.validationError);
    }
  }
}
