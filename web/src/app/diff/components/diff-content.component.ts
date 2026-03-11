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

import { Component, inject, input, model, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ResourceRevision } from '../../store/revision';
import { DiffToolbarComponent } from './diff-toolbar.component';
import { UnifiedDiffComponent } from 'ngx-diff';
import { HighlightModule } from 'ngx-highlightjs';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';

/**
 * Component for displaying the unified diff of a resource revision.
 */
@Component({
  selector: 'khi-diff-content',
  templateUrl: './diff-content.component.html',
  styleUrls: ['./diff-content.component.scss'],
  imports: [
    CommonModule,
    DiffToolbarComponent,
    UnifiedDiffComponent,
    HighlightModule,
  ],
})
export class DiffContentComponent {
  private readonly _clipboard = inject(Clipboard);
  private readonly _snackBar = inject(MatSnackBar);

  /**
   * The current revision being viewed.
   */
  readonly currentRevision = input.required<ResourceRevision | null>();

  /**
   * The content string of the current revision.
   */
  readonly currentRevisionContent = input.required<string>();

  /**
   * The content string of the previous revision to diff against.
   */
  readonly previousRevisionContent = input.required<string>();

  /**
   * Two-way bound state for showing managed fields in the diff.
   */
  readonly showManagedFields = model.required<boolean>();

  /**
   * Emitted when requesting to open the diff in a new window/tab.
   */
  readonly openInNewTab = output<void>();

  /**
   * Triggers the openInNewTab output event.
   */
  protected _openInNewTab() {
    this.openInNewTab.emit();
  }

  /**
   * Copies the current revision's content to the clipboard and displays a snackbar notification.
   */
  protected copyContent() {
    const content = this.currentRevisionContent();
    let snackbarMessage = 'Copy failed';
    if (this._clipboard.copy(content)) {
      snackbarMessage = 'Copied!';
    }
    this._snackBar.open(snackbarMessage, undefined, { duration: 1000 });
  }
}
