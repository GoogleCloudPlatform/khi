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

import { GroupParameterFormField } from 'src/app/common/schema/form-types';
import {
  InspectionMetadataJobModeCommand,
  InspectionMetadataQuery,
} from 'src/app/common/schema/metadata-types';

/**
 * Defines the step indices within the new inspection wizard.
 */
export enum NewInspectionStepIndex {
  /**
   * Represents the inspection type selection step.
   */
  InspectionType = 0,
  /**
   * Represents the feature selection step.
   */
  FeatureSelection = 1,
  /**
   * Represents the parameter input step.
   */
  ParameterInput = 2,
}

/**
 * Indicates that an asynchronous operation or delay was cancelled.
 */
export class CancellationError extends Error {
  /**
   * Initializes a new CancellationError instance.
   *
   * @param message The cancellation error message.
   */
  constructor(message = 'Operation was cancelled') {
    super(message);
    this.name = 'CancellationError';
  }
}

/**
 * Represents the dialog result returned when closing the new inspection dialog.
 */
export interface NewInspectionDialogResult {
  /**
   * Indicates whether an inspection task was successfully started.
   */
  readonly inspectionTaskStarted: boolean;
}

/**
 * Represents severity levels based on estimated log volume.
 */
export enum TotalEstimatedLogsSeverity {
  /**
   * Indicates normal log volume severity.
   */
  Normal = 'normal',
  /**
   * Indicates warning log volume severity.
   */
  Warning = 'warning',
  /**
   * Indicates danger log volume severity.
   */
  Danger = 'danger',
}

/**
 * Represents a summary of total estimated log counts across all queries.
 */
export interface TotalEstimatedLogsSummary {
  /**
   * Stores the total number of known logs estimated so far.
   */
  readonly knownCount: number;
  /**
   * Indicates whether all queries have completed estimation.
   */
  readonly isComplete: boolean;
  /**
   * Indicates whether estimation is actively in progress.
   */
  readonly isEstimating: boolean;
  /**
   * Indicates whether any query has incomplete parameters.
   */
  readonly isIncomplete: boolean;
  /**
   * Provides formatted display text for estimated log count.
   */
  readonly displayText: string;
  /**
   * Holds the severity level associated with the estimated log volume.
   */
  readonly severity: TotalEstimatedLogsSeverity;
}

/**
 * Represents the view model required to render the parameter input step.
 */
export interface ParameterStepViewModel {
  /**
   * Holds the root group containing all form parameter fields.
   */
  readonly rootGroupForm: GroupParameterFormField;
  /**
   * Stores the list of inspection queries.
   */
  readonly queries: InspectionMetadataQuery[];
  /**
   * Holds optional job mode command metadata.
   */
  readonly job?: InspectionMetadataJobModeCommand;
  /**
   * Stores the total count of input fields across all parameter groups.
   */
  readonly fieldCount: number;
  /**
   * Holds the optional total estimated logs summary.
   */
  readonly totalEstimatedSummary?: TotalEstimatedLogsSummary;
}

/**
 * Represents the configuration data passed when opening the new inspection dialog.
 */
export interface NewInspectionDialogData {
  /**
   * Stores the initial inspection type ID to preselect.
   */
  readonly initialInspectionTypeId?: string;
  /**
   * Stores the initial feature IDs to enable.
   */
  readonly initialFeatureIds?: readonly string[];
  /**
   * Stores initial parameter values to prefill into the form.
   */
  readonly initialParameters?: Record<string, unknown>;
}
