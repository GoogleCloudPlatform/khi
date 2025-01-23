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

/**
 * api-types.ts
 * Defines the API schemas used between KHI backend and frontend
 */

import {
  InspectionMetadataErrorSet,
  InspectionMetadataFormField,
  InspectionMetadataHeader,
  InspectionMetadataLog,
  InspectionMetadataPlan,
  InspectionMetadataProgress,
  InspectionMetadataQuery,
} from './metadata-types';

/**
 * Representing a type of inspection. This usually represents a cluster type(e.g GKE, Cloud Composer ...etc).
 */
export interface InspectionType {
  /**
   * Unique ID of this inspection type.
   */
  id: string;

  /**
   * Name of this inspection type. (e.g. Google Kubernetes Engine ...etc) .
   */
  name: string;

  /**
   * Description of this inspection type.
   */
  description: string;

  /**
   * Address pointing icon image.
   */
  icon: string;
}

/**
 * The response schema of GET /api/v2/inspection/types .
 */
export interface GetInspectionTypesResponse {
  /**
   * List of types supporting on this environment.
   */
  types: InspectionType[];
}

/**
 * The response schema of POST /api/v2/inspection/types/<InspectionType.id> .
 */
export interface CreateInspectionTaskResponse {
  /**
   * ID of the inspection task created.
   */
  inspectionId: string;
}

/**
 * Representing  a feature of inspection. This usually represents a log type(e.g. Kubernetes Audit Log, Kubernetes Event log ...etc).
 */
export interface InspectionFeature {
  /**
   * Unique ID of this inspection feature.
   */
  id: string;

  /**
   * Label of this inspection feature. Label must be a short descriptive name for the feature.
   */
  label: string;

  /**
   * Description of this inspection feature.
   */
  description: string;

  /**
   * Whether if this feature is turned on or not.
   */
  enabled: boolean;
}

/**
 * Response schema of GET /api/v2/inspection/tasks/<task-id>/features .
 */
export interface GetInspectionTaskFeatureResponse {
  /**
   * List of features for the inspection task.
   */
  features: InspectionFeature[];
}

/**
 * Request schema of PUT /api/v2/inspection/tasks/<task-id>/features .
 */
export interface PutInspectionTaskFeatureRequest {
  /**
   * List of IDs to be enabled.
   */
  features: string[];
}

/**
 * Response schema of POST /api/v2/inspection/tasks/<inspection task id>/dryrun .
 */
export type InspectionDryRunResponse = {
  /**
   * Metadata of the dryrun result.
   * Metadata in KHI inspection context is that the data generated along with executing the inspection task graph.
   * It usually contains the validation error or the other field but not containing the main inspection main data.
   */
  metadata: InspectionMetadataInDryrun;
};

/**
 * Representing a set of parameters given to the inspection task graph.
 */
type InspectionTaskGraphArgument = { [key: string]: unknown };

/**
 * Request schema of POST /api/v2/inspection/tasks/<inspection task id>/dryrun .
 */
export type InspectionDryRunRequest = InspectionTaskGraphArgument;

/**
 * Request schema of POST /api/v/inspection/tasks/<inspection task id>/run .
 */
export type InspectionRunRequest = InspectionTaskGraphArgument;

/**
 * Set of metadata generated for a task not having run yet.
 */
export type InspectionMetadataInDryrun = {
  /**
   * List of form fields to be filled to run this inspection task.
   */
  form: InspectionMetadataFormField[];

  /**
   * List of queries to be run with this inspection task.
   */
  query: InspectionMetadataQuery[];

  /**
   * The inspection task graph to be run with inspection task.
   */
  plan: InspectionMetadataPlan;
};

/**
 * Set of metadata generated for tasks in the task list.
 */
export type InspectionMetadataInTaskList = {
  /**
   * Current progress of this inspection task.
   */
  progress: InspectionMetadataProgress;
  /**
   * Summary of this inspection task like name, data size ...etc.
   */
  header: InspectionMetadataHeader;
  /**
   * Set of error logs for this inspection task.
   */
  error: InspectionMetadataErrorSet;
};

/**
 * Set of metadata generated for tasks completed.
 */
export type InspectionMetadataOfRunResult = {
  /**
   * Summary of this inspection task like name, data size ...etc.
   */
  header: InspectionMetadataHeader;
  /**
   * List of queries having run with this inspection task.
   */
  query: InspectionMetadataQuery[];
  /**
   * The inspection task graph having run with inspection task.
   */
  plan: InspectionMetadataPlan;
  /**
   * The logs generated from the inspection task itself.
   */
  log: InspectionMetadataLog[];

  /**
   * Set of error logs for this inspection task.
   */
  error: InspectionMetadataErrorSet;
};

/**
 * Response schema of /api/v2/inspection/tasks .
 */
export type GetInspectionTasksResponse = {
  tasks: {
    [taskId: string]: InspectionMetadataInTaskList;
  };
  serverStat: {
    totalMemoryAvailable: number;
  };
};

export type PopupFormType = 'text' | 'popup_redirect';

/**
 * PopupFormRequest is a type returned on the endpoint GET /api/v2/popup.
 * Note this request is from backend with polling. Thus this is also a response in HTTP.
 */
export interface PopupFormRequest {
  id: string;
  title: string;
  type: PopupFormType;
  description: string;
  placeholder: string;
  options: {
    /**
     * The redirect target. This option is valid only when the type is `popup_redirect`.
     */
    redirectTo?: string;
    [key: string]: string | undefined;
  };
}

/**
 * PopupAnswerResponse is a type replied to the server with the endpoint POST /api/v2/popup/answer or POST /api/v2/popup/validate
 */
export interface PopupAnswerResponse {
  id: string;
  value: string;
}

/**
 * PopupValidationResult is a type returned from server on POST /api/v2/popup/validate
 */
export interface PopupAnswerValidationResult {
  id: string;
  validationError: string;
}
