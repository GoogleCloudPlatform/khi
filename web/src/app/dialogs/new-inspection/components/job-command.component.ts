// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { ClipboardModule, Clipboard } from '@angular/cdk/clipboard';
import { CommonModule } from '@angular/common';
import { Component, input, inject } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * JobCommandComponent renders the CLI command example for executing the inspection in job mode.
 * It provides a copy button to let users easily copy the command to their clipboard.
 */
@Component({
  selector: 'khi-job-command',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    MatSnackBarModule,
    ClipboardModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './job-command.component.html',
  styleUrl: './job-command.component.scss',
})
export class JobCommandComponent {
  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);

  /**
   * The copy-pasteable job command.
   */
  readonly command = input.required<string>();

  /**
   * Copies the command to clipboard and triggers a snackbar notification.
   */
  copyCommand() {
    if (this.clipboard.copy(this.command())) {
      this.snackBar.open('Copied!', undefined, { duration: 1500 });
    } else {
      this.snackBar.open('Copy failed.', undefined, { duration: 1500 });
    }
  }
}
