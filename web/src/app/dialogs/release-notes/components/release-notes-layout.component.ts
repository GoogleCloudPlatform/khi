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
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDialogModule } from '@angular/material/dialog';
import { parse } from 'marked';
import DOMPurify from 'dompurify';

/**
 * Layout component for the Release Notes dialog.
 * Displays sanitized markdown release notes and a checkbox to suppress future displays for the version.
 */
@Component({
  selector: 'khi-release-notes-layout',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatCheckboxModule,
    MatDialogModule,
  ],
  templateUrl: './release-notes-layout.component.html',
  styleUrls: ['./release-notes-layout.component.scss'],
})
export class ReleaseNotesLayoutComponent {
  /** The raw markdown content of the release notes. */
  readonly markdownContent = input.required<string>();

  /** Two-way bound model indicating whether to hide release notes for this version. */
  readonly doNotShowAgain = model<boolean>(false);

  /** Event emitted when the close button is clicked. */
  readonly closed = output<void>();

  /** Computed sanitized HTML generated from the raw markdown input. */
  readonly sanitizedHtml = computed<string>(() => {
    const rawHtml = parse(this.markdownContent(), { async: false }) as string;
    return DOMPurify.sanitize(rawHtml);
  });

  /**
   * Triggers the closed event emission.
   */
  protected onClose(): void {
    this.closed.emit();
  }
}
