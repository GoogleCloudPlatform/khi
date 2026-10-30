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
import { MatIconModule } from '@angular/material/icon';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatTabsModule } from '@angular/material/tabs';
import { MaskExportFooterComponent } from 'src/app/mask/components/mask-export-footer.component';
import { MaskIndexBannerComponent } from 'src/app/mask/components/mask-index-banner.component';
import { MaskRuleListComponent } from 'src/app/mask/components/mask-rule-list.component';
import { MaskStringPoolComponent } from 'src/app/mask/components/mask-string-pool.component';
import {
  CandidateToggleEvent,
  MaskCandidateItemViewModel,
  MaskExportProgressViewModel,
  MaskRuleItemViewModel,
  RuleEnabledChangeEvent,
  RuleReplacementChangeEvent,
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
    MatIconModule,
    MatSlideToggleModule,
    MatTabsModule,
    KHIIconRegistrationModule,
    MaskIndexBannerComponent,
    MaskStringPoolComponent,
    MaskRuleListComponent,
    MaskExportFooterComponent,
  ],
  templateUrl: './mask-layout.component.html',
  styleUrls: ['./mask-layout.component.scss'],
})
export class MaskLayoutComponent {
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
   * Label displayed on the Selected Masks tab header.
   */
  protected readonly selectedMasksTabLabel = computed<string>(
    () => `Selected Masks (${this.rules().length})`,
  );
}
