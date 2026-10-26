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

/**
 * Construction lifecycle phase of the on-demand string pool trigram index.
 */
export enum StringPoolIndexPhase {
  Idle = 'IDLE',
  Building = 'BUILDING',
  Ready = 'READY',
  Failed = 'FAILED',
}

/**
 * View model representing the current progress of on-demand string pool trigram indexing.
 */
export interface StringPoolIndexStatusViewModel {
  readonly phase: StringPoolIndexPhase;
  readonly progressPercentage: number;
  readonly message: string;
}

/**
 * View model representing a single candidate string from the Workbench string pool.
 */
export interface MaskCandidateItemViewModel {
  readonly value: string;
  readonly selected: boolean;
  readonly inRuleList: boolean;
}

/**
 * View model representing a configured string masking rule in the Selected Masks tab.
 */
export interface MaskRuleItemViewModel {
  readonly target: string;
  readonly replacement: string;
  readonly defaultReplacement: string;
  readonly isDeletion: boolean;
  readonly isDefaultReplacement: boolean;
  readonly enabled: boolean;
}

/**
 * Event payload emitted when a candidate string checkbox is toggled.
 */
export interface CandidateToggleEvent {
  readonly target: string;
  readonly checked: boolean;
}

/**
 * Event payload emitted when a configured mask rule's enabled flag changes.
 */
export interface RuleEnabledChangeEvent {
  readonly target: string;
  readonly enabled: boolean;
}

/**
 * Event payload emitted when a configured mask rule's replacement text changes.
 */
export interface RuleReplacementChangeEvent {
  readonly target: string;
  readonly replacement: string;
}

/**
 * Active execution stage of a masked .khi archive export and download operation.
 */
export enum MaskExportPhase {
  GeneratingArchive = 'GENERATING_ARCHIVE',
  DownloadingFile = 'DOWNLOADING_FILE',
}

/**
 * View model representing the current progress of generating and downloading a masked .khi archive.
 */
export interface MaskExportProgressViewModel {
  readonly phase: MaskExportPhase;
  readonly phaseLabel: string;
  readonly progressPercentage: number;
  readonly message: string;
}
