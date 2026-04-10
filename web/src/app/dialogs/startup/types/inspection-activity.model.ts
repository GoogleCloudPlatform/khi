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

import { InspectionMetadataProgressPhase } from 'src/app/common/schema/metadata-types';

/**
 * Represents the view model for an inspection list item.
 */
export interface InspectionListItemViewModel {
  /** Unique identifier for the inspection. */
  id: string;
  /** Label indicating when the inspection was performed. */
  inspectionTimeLabel: string;
  /** The display name or title of the inspection. */
  label: string;
  /** The current phase of the inspection. */
  phase: InspectionMetadataProgressPhase;
  /** Total progress of the inspection. */
  totalProgress: ProgressBarViewModel;
  /** Detailed progress for sub-tasks. */
  progresses: ProgressBarViewModel[];
  /** Errors encountered during the inspection. */
  errors: ErrorViewModel[];
}

/**
 * Represents the progress bar state.
 */
export interface ProgressBarViewModel {
  /** Unique identifier for the task. */
  id: string;
  /** Label for the task. */
  label: string;
  /** Detailed status message. */
  message: string;
  /** Progress percentage (0-100). */
  percentage: number;
  /** Formatted percentage text (e.g., "50%"). */
  percentageLabel: string;
  /** Whether the progress is indeterminate. */
  indeterminate: boolean;
}

/**
 * Represents an error associated with an inspection.
 */
export interface ErrorViewModel {
  /** The error message. */
  message: string;
  /** Optional link for more details or action. */
  link: string;
}

/**
 * Request payload for changing an inspection title.
 */
export interface InspectionTitleChangeRequest {
  /** Unique identifier for the inspection. */
  id: string;
  /** The new title to set. */
  changeTo: string;
}
