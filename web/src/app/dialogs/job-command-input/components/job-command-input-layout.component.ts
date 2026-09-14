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
import { Component, computed, input, model, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * JobCommandInputLayoutComponent is a dumb presentation component for pasting and submitting a Job Mode CLI command.
 */
@Component({
  selector: 'khi-job-command-input-layout',
  templateUrl: './job-command-input-layout.component.html',
  styleUrls: ['./job-command-input-layout.component.scss'],
  imports: [
    CommonModule,
    FormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    KHIIconRegistrationModule,
  ],
})
export class JobCommandInputLayoutComponent {
  /**
   * The command text entered by the user.
   */
  readonly command = model<string>('');

  /**
   * Optional error message to display when parsing fails.
   */
  readonly errorMessage = input<string | null>(null);

  /**
   * Emitted when the user confirms and submits the command.
   */
  readonly submitCommand = output<void>();

  /**
   * Emitted when the user cancels the dialog.
   */
  readonly cancelDialog = output<void>();

  /**
   * Whether the submit button should be disabled.
   */
  readonly isSubmitDisabled = computed(() => {
    return this.command().trim().length === 0;
  });
}
