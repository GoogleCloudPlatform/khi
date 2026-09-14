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

import { Component, DestroyRef, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ClipboardModule } from '@angular/cdk/clipboard';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Dumb component that renders a monospace code or text block with a copy button.
 */
@Component({
  selector: 'khi-metadata-code-viewer',
  imports: [
    CommonModule,
    ClipboardModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './metadata-code-viewer.component.html',
  styleUrls: ['./metadata-code-viewer.component.scss'],
})
export class MetadataCodeViewerComponent {
  /** Optional title shown in the header of the code block. */
  readonly title = input<string>('');

  /** Optional badge text shown next to the title. */
  readonly badge = input<string>('');

  /** The code or text content to display. */
  readonly code = input.required<string>();

  /** Maximum height CSS value for the code block. */
  readonly maxHeight = input<string>('240px');

  /** Internal state indicating recent copy action for visual feedback. */
  protected readonly isCopied = signal(false);

  private readonly destroyRef = inject(DestroyRef);
  private copyTimerId: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    this.destroyRef.onDestroy(() => {
      if (this.copyTimerId !== null) {
        clearTimeout(this.copyTimerId);
        this.copyTimerId = null;
      }
    });
  }

  /** Handles the copy success event and displays temporary feedback. */
  protected onCopied(): void {
    this.isCopied.set(true);
    if (this.copyTimerId !== null) {
      clearTimeout(this.copyTimerId);
    }
    this.copyTimerId = setTimeout(() => {
      this.isCopied.set(false);
      this.copyTimerId = null;
    }, 1500);
  }
}
