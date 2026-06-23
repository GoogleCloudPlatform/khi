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
import { HttpClient } from '@angular/common/http';
import { toSignal } from '@angular/core/rxjs-interop';
import {
  MatDialog,
  MatDialogConfig,
  MatDialogRef,
} from '@angular/material/dialog';
import { VERSION } from 'src/environments/version';
import { ReleaseNotesLayoutComponent } from './components/release-notes-layout.component';

/** LocalStorage key used to store the version for which release notes are suppressed. */
export const SUPPRESSED_RELEASE_NOTES_VERSION_KEY =
  'khi_suppressed_release_notes_version';

/**
 * Smart component for the Release Notes dialog.
 * Handles dialog lifecycle, fetching release notes markdown from assets, and persistence of suppression preference.
 */
@Component({
  selector: 'khi-release-notes-smart',
  imports: [ReleaseNotesLayoutComponent],
  templateUrl: './release-notes-smart.component.html',
  styleUrls: ['./release-notes-smart.component.scss'],
  host: { style: 'display: contents;' },
})
export class ReleaseNotesDialogSmartComponent {
  private readonly dialogRef =
    inject<MatDialogRef<ReleaseNotesDialogSmartComponent>>(MatDialogRef);
  private readonly http = inject(HttpClient);

  protected readonly version = VERSION;
  protected readonly markdownContent = toSignal(
    this.http.get('assets/release_note/release_note.md', {
      responseType: 'text',
    }),
    { initialValue: '' },
  );
  protected readonly doNotShowAgain = signal<boolean>(false);

  /**
   * Handles dialog close event. Saves suppression preference if requested.
   */
  protected onClose(): void {
    if (this.doNotShowAgain()) {
      localStorage.setItem(SUPPRESSED_RELEASE_NOTES_VERSION_KEY, VERSION);
    }
    this.dialogRef.close();
  }
}

/**
 * Opens the Release Notes dialog if not suppressed for the current version.
 * @param dialog MatDialog service instance.
 * @param force If true, opens the dialog even if suppressed for the current version.
 * @param config Optional dialog configuration overrides.
 * @returns MatDialogRef instance if opened, otherwise null.
 */
export function openReleaseNotesDialog(
  dialog: MatDialog,
  force = false,
  config: Partial<MatDialogConfig> = {},
): MatDialogRef<ReleaseNotesDialogSmartComponent> | null {
  if (!force) {
    const suppressedVersion = localStorage.getItem(
      SUPPRESSED_RELEASE_NOTES_VERSION_KEY,
    );
    if (suppressedVersion === VERSION) {
      return null;
    }
  }

  return dialog.open(ReleaseNotesDialogSmartComponent, {
    maxWidth: '80vw',
    minWidth: '600px',
    height: '90vh',
    maxHeight: '90vh',
    ...config,
  });
}
