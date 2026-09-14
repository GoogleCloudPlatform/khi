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

import { Component, inject, signal } from '@angular/core';
import {
  MatDialog,
  MatDialogConfig,
  MatDialogRef,
} from '@angular/material/dialog';
import { JobCommandInputLayoutComponent } from 'src/app/dialogs/job-command-input/components/job-command-input-layout.component';
import {
  parseJobModeCommand,
  ParsedJobCommand,
} from 'src/app/dialogs/startup/utils/job-command-parser';

/**
 * Smart component for the Job Mode Command Input dialog.
 * Handles parsing validation of pasted commands and returns the parsed result.
 */
@Component({
  selector: 'khi-job-command-input-smart',
  imports: [JobCommandInputLayoutComponent],
  templateUrl: './job-command-input-smart.component.html',
  styleUrls: ['./job-command-input-smart.component.scss'],
  host: { style: 'display: contents;' },
})
export class JobCommandInputSmartComponent {
  private readonly dialogRef =
    inject<
      MatDialogRef<JobCommandInputSmartComponent, ParsedJobCommand | null>
    >(MatDialogRef);

  /**
   * The command string entered by the user.
   */
  protected readonly command = signal<string>('');

  /**
   * Validation or parse error message to display in the dialog.
   */
  protected readonly errorMessage = signal<string | null>(null);

  /**
   * Parses the command and closes the dialog with the parsed result if valid.
   */
  protected onSubmit(): void {
    try {
      const parsed = parseJobModeCommand(this.command());
      this.dialogRef.close(parsed);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      this.errorMessage.set(message);
    }
  }

  /**
   * Closes the dialog without submitting.
   */
  protected onCancel(): void {
    this.dialogRef.close(null);
  }
}

/**
 * Opens the Job Command Input dialog.
 *
 * @param dialog The Angular Material MatDialog service instance.
 * @param config Optional dialog configuration overrides.
 * @returns Reference to the opened dialog.
 */
export function openJobCommandInputDialog(
  dialog: MatDialog,
  config?: MatDialogConfig,
): MatDialogRef<JobCommandInputSmartComponent, ParsedJobCommand | null> {
  return dialog.open<JobCommandInputSmartComponent, ParsedJobCommand | null>(
    JobCommandInputSmartComponent,
    {
      width: '640px',
      ...config,
    },
  );
}
