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

import { Observable } from 'rxjs';
import {
  GetConfigResponse,
  GetInspectionTaskFeatureResponse,
  GetInspectionTasksResponse,
  GetInspectionTypesResponse,
  InspectionDryRunRequest,
  InspectionDryRunResponse,
  InspectionMetadataOfRunResult,
  InspectionRunRequest,
  PopupAnswerResponse,
  PopupAnswerValidationResult,
  PopupFormRequest,
} from '../../common/schema/api-types';
import { InspectionTaskClient } from './backend-api.service';
import { InjectionToken } from '@angular/core';
import { UploadToken } from 'src/app/common/schema/form-types';
import { HttpEvent } from '@angular/common/http';

/**
 * A function type to report the progress of download.
 */
export type DownloadProgressReporter = (doneBytes: number) => void;

export const BACKEND_API = new InjectionToken<BackendAPI>('BACKEND_API');

export interface BackendAPI {
  /**
   * Get configuration applied on this frontend.
   * Expected called endpoint: GET /api/v2/config
   */
  getConfig(): Observable<GetConfigResponse>;
  /**
   * Get the list of inspection types.
   * Expected called endpoint: GET /api/v2/inspection/types
   */
  getInspectionTypes(): Observable<GetInspectionTypesResponse>;

  /**
   * List the status of tasks.
   * Expected called endpoint: GET /api/v2/inspection/tasks
   */
  getTaskStatuses(): Observable<GetInspectionTasksResponse>;

  /**
   * Create an inspection.
   * Expected called endpoint: POST /api/v2/inspection/types/<inspection-type>
   *
   * This function will return a http client for operating the generated task instead of returning the API response directly.
   *
   * @param inspectionTypeId the type of inspection id listed in the result of `getInspectionTypes()`
   */
  createInspection(inspectionTypeId: string): Observable<InspectionTaskClient>;

  /**
   * List the features selectable of this task.
   * Expected called endpoint: GET /api/v2/inspection/tasks/<task-id>/features
   *
   * @param taskId inspection task ID to list the feature
   */
  getFeatureList(taskId: string): Observable<GetInspectionTaskFeatureResponse>;

  /**
   * Set the selected features of this task.
   * Expected called endpoint: PUT /api/v2/inspection/tasks/<task-id>/features
   *
   * @param taskId inspection task ID to set the selected feature
   * @param featureIds list of feature Ids
   */
  setEnabledFeatures(taskId: string, featureIds: string[]): Observable<void>;

  /**
   * Get the metadata of an inspection with taskId.
   * Expected called endpoint: GET /api/v2/inspection/tasks/<task-id>/metadata
   *
   * @param taskId inspection task ID to download the metadata
   */
  getInspectionMetadata(
    taskId: string,
  ): Observable<InspectionMetadataOfRunResult>;

  /**
   * Request running a task.
   * Expected called endpoint: POST /api/v2/inspection/tasks/<task-id>/run
   *
   * @param taskId inspection taskId to run
   * @param request parameter of the task
   */
  runTask(taskId: string, request: InspectionRunRequest): Observable<void>;

  /**
   * Request dry run a task.
   * Expected called endpoint: POST /api/v2/inspection/tasks/<task-id>/dryrun
   * @param taskId inspection taskId to dryrun
   * @param request parameter of the task
   */
  dryRunTask(
    taskId: string,
    request: InspectionDryRunRequest,
  ): Observable<InspectionDryRunResponse>;

  /**
   * Get the inspection data with taskId.
   * Expected called endpoint: GET /api/v2/inspection/tasks/<task-id>/data
   *
   * @param taskId inspection task ID to download the result.
   * @param reporter task progress reporter.
   */
  getInspectionData(
    taskId: string,
    reporter: DownloadProgressReporter,
  ): Observable<Blob | null>;

  /**
   * Cancel the inspection task.
   * Expected called endpoint: POST /api/v2/inspection/tasks/<task-id>/cancel
   *
   * @param taskId inspection task ID to cancel
   */
  cancelInspection(taskId: string): Observable<void>;

  /**
   * Get the current popup request.
   * Expected called endpoint: GET /api/v2/popup
   */
  getPopup(): Observable<PopupFormRequest | null>;

  /**
   * Validate the request for the current popup
   * Expected called endpoint: POST /api/v2/popup/validate
   */
  validatePopupAnswer(
    answer: PopupAnswerResponse,
  ): Observable<PopupAnswerValidationResult>;

  /**
   * Answer the current request for the popup
   * Expected called endpoint: POST /api/v2/popup/answer
   */
  answerPopup(answer: PopupAnswerResponse): Observable<void>;

  /**
   * Upload the file as the one bound to the token.
   */
  uploadFile(token: UploadToken, file: File): Observable<HttpEvent<unknown>>;
}
