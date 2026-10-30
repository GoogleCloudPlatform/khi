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
import { Component, computed, input, output } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import {
  MaskExportPhase,
  MaskExportProgressViewModel,
} from 'src/app/mask/types/mask-viewmodel';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Displays the Mask pane footer with determinate export progress, status feedback, and export/clear actions.
 */
@Component({
  selector: 'khi-mask-export-footer',
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './mask-export-footer.component.html',
  styleUrls: ['./mask-export-footer.component.scss'],
})
export class MaskExportFooterComponent {
  /**
   * Total count of configured masking rules (enabled or disabled).
   */
  public readonly ruleCount = input.required<number>();

  /**
   * Count of enabled masking rules ready to be exported.
   */
  public readonly enabledRuleCount = input.required<number>();

  /**
   * Whether a masked .khi export and download operation is in progress.
   */
  public readonly isExporting = input.required<boolean>();

  /**
   * Live determinate progress details while generating and downloading a masked .khi archive.
   */
  public readonly exportProgress = input<MaskExportProgressViewModel | null>(
    null,
  );

  /**
   * Optional status message describing the latest export or search operation.
   */
  public readonly statusMessage = input<string>('');

  /**
   * Optional error message when string listing or export fails.
   */
  public readonly errorMessage = input<string>('');

  /**
   * Emits when the user clears all configured rules.
   */
  public readonly clearAllRules = output<void>();

  /**
   * Emits when the user triggers masked .khi generation and download.
   */
  public readonly exportMaskedKhi = output<void>();

  /**
   * Whether the Clear All button can be triggered.
   */
  protected readonly canClearAll = computed<boolean>(
    () => this.ruleCount() > 0 && !this.isExporting(),
  );

  /**
   * Whether the export button can be triggered.
   */
  protected readonly canExport = computed<boolean>(
    () => this.enabledRuleCount() > 0 && !this.isExporting(),
  );

  /**
   * Rounded overall export progress percentage (0 - 100) displayed during masked .khi export.
   */
  protected readonly roundedExportProgress = computed<number>(() => {
    const progress = this.exportProgress();
    if (!progress) {
      return 0;
    }
    return Math.min(100, Math.max(0, Math.round(progress.progressPercentage)));
  });

  /**
   * Material icon name representing the active masked .khi export phase.
   */
  protected readonly exportPhaseIcon = computed<string>(() =>
    this.exportProgress()?.phase === MaskExportPhase.DownloadingFile
      ? 'downloading'
      : 'archive',
  );

  /**
   * Label displayed inside the primary Export Masked .khi button.
   */
  protected readonly exportButtonLabel = computed<string>(() => {
    if (this.isExporting()) {
      return this.exportProgress()
        ? `Exporting... (${this.roundedExportProgress()}%)`
        : 'Exporting...';
    }
    return `Export Masked .khi (${this.enabledRuleCount()})`;
  });
}
