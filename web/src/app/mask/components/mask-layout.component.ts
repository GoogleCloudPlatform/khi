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
import { Component, computed, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatTabsModule } from '@angular/material/tabs';
import { MatTooltipModule } from '@angular/material/tooltip';
import {
  CandidateToggleEvent,
  MaskCandidateItemViewModel,
  MaskExportPhase,
  MaskExportProgressViewModel,
  MaskRuleItemViewModel,
  RuleEnabledChangeEvent,
  RuleReplacementChangeEvent,
  StringPoolIndexPhase,
  StringPoolIndexStatusViewModel,
} from 'src/app/mask/types/mask-viewmodel';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Layout component for the dockable String Masking and Masked .khi Export pane.
 */
@Component({
  selector: 'khi-mask-layout',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatCheckboxModule,
    MatIconModule,
    MatProgressBarModule,
    MatSlideToggleModule,
    MatTabsModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './mask-layout.component.html',
  styleUrls: ['./mask-layout.component.scss'],
})
export class MaskLayoutComponent {
  protected readonly StringPoolIndexPhase = StringPoolIndexPhase;

  /**
   * Whether an active Workbench inspection session is loaded.
   */
  public readonly isWorkbenchActive = input.required<boolean>();

  /**
   * Current construction status of the on-demand string pool trigram index.
   */
  public readonly indexStatus =
    input.required<StringPoolIndexStatusViewModel>();

  /**
   * Whether live masking preview across KHI views is enabled.
   */
  public readonly livePreviewEnabled = input.required<boolean>();

  /**
   * Active search query filtering candidate strings in the string pool.
   */
  public readonly searchQuery = input.required<string>();

  /**
   * Inline RE2 syntax validation error message for the active search query.
   */
  public readonly searchRegexError = input<string>('');

  /**
   * Candidate strings currently loaded from the Workbench string pool.
   */
  public readonly candidates =
    input.required<readonly MaskCandidateItemViewModel[]>();

  /**
   * Total count of unique strings matching the search query on the backend.
   */
  public readonly totalCandidateCount = input.required<number>();

  /**
   * Whether candidate strings are currently being fetched.
   */
  public readonly isLoadingCandidates = input.required<boolean>();

  /**
   * Configured masking rules displayed in the Selected Masks tab.
   */
  public readonly rules = input.required<readonly MaskRuleItemViewModel[]>();

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
   * Emits when the live preview toggle changes.
   */
  public readonly livePreviewChange = output<boolean>();

  /**
   * Emits when the candidate search query text changes.
   */
  public readonly searchQueryChange = output<string>();

  /**
   * Emits when the user selects all currently displayed candidate strings in bulk.
   */
  public readonly selectShownCandidates = output<void>();

  /**
   * Emits when the user requests the next page of candidate strings.
   */
  public readonly loadMoreCandidates = output<void>();

  /**
   * Emits when a candidate string checkbox is toggled.
   */
  public readonly candidateToggle = output<CandidateToggleEvent>();

  /**
   * Emits when the user submits a custom substring to mask.
   */
  public readonly addCustomSubstring = output<string>();

  /**
   * Emits when a rule's enabled checkbox changes.
   */
  public readonly ruleEnabledChange = output<RuleEnabledChangeEvent>();

  /**
   * Emits when a rule's replacement string changes.
   */
  public readonly ruleReplacementChange = output<RuleReplacementChangeEvent>();

  /**
   * Emits when the user resets a rule's replacement back to its assigned `<masked_N>` placeholder.
   */
  public readonly ruleResetReplacement = output<string>();

  /**
   * Emits when a rule is removed from the rule list.
   */
  public readonly ruleRemove = output<string>();

  /**
   * Emits when the user clears all configured rules.
   */
  public readonly clearAllRules = output<void>();

  /**
   * Emits when the user triggers masked .khi generation and download.
   */
  public readonly exportMaskedKhi = output<void>();

  /**
   * Local draft text for adding an arbitrary substring rule.
   */
  protected readonly customSubstringDraft = signal<string>('');

  /**
   * Rounded percentage displayed in the string pool index progress banner.
   */
  protected readonly roundedIndexProgress = computed<number>(() =>
    Math.min(
      100,
      Math.max(0, Math.round(this.indexStatus().progressPercentage)),
    ),
  );

  /**
   * Whether additional pages of candidate strings remain to be loaded.
   */
  protected readonly hasMoreCandidates = computed<boolean>(
    () => this.candidates().length < this.totalCandidateCount(),
  );

  /**
   * Whether the custom substring input contains a non-empty value.
   */
  protected readonly canAddCustomSubstring = computed<boolean>(
    () => this.customSubstringDraft().length > 0,
  );

  /**
   * Whether the export button can be triggered.
   */
  protected readonly canExport = computed<boolean>(
    () =>
      this.isWorkbenchActive() &&
      this.enabledRuleCount() > 0 &&
      !this.isExporting(),
  );

  /**
   * Label displayed on the Selected Masks tab header.
   */
  protected readonly selectedMasksTabLabel = computed<string>(
    () => `Selected Masks (${this.rules().length})`,
  );

  /**
   * Whether an inline RE2 syntax validation error should be displayed below the search input.
   */
  protected readonly hasSearchRegexError = computed<boolean>(
    () => this.searchRegexError().length > 0,
  );

  /**
   * Count of currently displayed candidate strings that are not yet selected.
   */
  protected readonly unselectedShownCount = computed<number>(
    () => this.candidates().filter((c) => !c.selected).length,
  );

  /**
   * Whether the bulk "Select shown" button can be triggered.
   */
  protected readonly canSelectShown = computed<boolean>(
    () => this.unselectedShownCount() > 0 && !this.isLoadingCandidates(),
  );

  /**
   * Button label displaying how many unselected candidate strings will be added in bulk.
   */
  protected readonly selectShownLabel = computed<string>(() => {
    const count = this.unselectedShownCount();
    return count > 0 ? `Select shown (${count})` : 'Select shown';
  });

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

  protected onSearchInputChange(value: string): void {
    this.searchQueryChange.emit(value);
  }

  protected onClearSearch(): void {
    this.searchQueryChange.emit('');
  }

  protected onCustomSubstringInput(value: string): void {
    this.customSubstringDraft.set(value);
  }

  protected onSubmitCustomSubstring(): void {
    const target = this.customSubstringDraft();
    if (target.length === 0) {
      return;
    }
    this.addCustomSubstring.emit(target);
    this.customSubstringDraft.set('');
  }

  protected onCandidateCheckboxChange(value: string, checked: boolean): void {
    this.candidateToggle.emit({ target: value, checked });
  }

  protected onRuleEnabledToggle(target: string, enabled: boolean): void {
    this.ruleEnabledChange.emit({ target, enabled });
  }

  protected onRuleReplacementInput(target: string, replacement: string): void {
    this.ruleReplacementChange.emit({ target, replacement });
  }
}
