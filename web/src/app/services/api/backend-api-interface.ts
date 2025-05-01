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
  GetInspectionFeatureResponse,
  GetInspectionResponse,
  GetInspectionTypesResponse,
  InspectionDryRunRequest,
  InspectionDryRunResponse,
  InspectionMetadataOfRunResult,
  InspectionRunRequest,
  PopupAnswerResponse,
  PopupAnswerValidationResult,
  PopupFormRequest,
} from '../../common/schema/api-types';
import { InspectionClient } from './backend-api.service';
import { InjectionToken } from '@angular/core';
import { UploadToken } from 'src/app/common/schema/form-types';
import { HttpEvent } from '@angular/common/http';

/**
 * A function type to report the progress of download.
 * @param allBytes
 * @param doneBytes downloaded bytes for each calls
 */
export type DownloadProgressReporter = (
  allBytes: number,
  doneBytes: number,
) => void;

export const BACKEND_API = new InjectionToken<BackendAPI>('BACKEND_API');

export interface BackendAPI {
  /**
   * Get configuration applied on this frontend.
   * Expected called endpoint: GET /api/v3/config
   */
  getConfig(): Observable<GetConfigResponse>;
  /**
   * Get the list of inspection types.
   * Expected called endpoint: GET /api/v3/inspection/types
   */
  getInspectionTypes(): Observable<GetInspectionTypesResponse>;

  /**
   * List the status of tasks.
   * Expected called endpoint: GET /api/v3/inspection
   */
  getInspections(): Observable<GetInspectionResponse>;

  /**
   * Create an inspection.
   * Expected called endpoint: POST /api/v3/inspection/types/<inspection-type>
   *
   * This function will return a http client for operating the generated task instead of returning the API response directly.
   *
   * @param inspectionTypeId the type of inspection id listed in the result of `getInspectionTypes()`
   */
  createInspection(inspectionTypeId: string): Observable<InspectionClient>;

  /**
   * List the features selectable of this task.
   * Expected called endpoint: GET /api/v3/inspection/<inspection-id>/features
   *
   * @param inspectionID inspection ID to list the feature
   */
  getFeatureList(
    inspectionID: string,
  ): Observable<GetInspectionFeatureResponse>;

  /**
   * Set the selected features of this task.
   * Expected called endpoint: PUT /api/v3/inspection/<inspection-id>/features
   *
   * @param inspectionID inspection ID to set the selected feature
   * @param featureStatusMap Map of features mapped against true if enabled
   */
  setEnabledFeatures(
    inspectionID: string,
    featureStatusMap: { [key: string]: boolean },
  ): Observable<void>;

  /**
   * Get the metadata of an inspection with taskId.
   * Expected called endpoint: GET /api/v3/inspection/<inspection-id>/metadata
   *
   * @param inspectionID inspection ID to download the metadata
   */
  getInspectionMetadata(
    inspectionID: string,
  ): Observable<InspectionMetadataOfRunResult>;

  /**
   * Request running a task.
   * Expected called endpoint: POST /api/v3/inspection/<inspection-id>/run
   *
   * @param inspectionID inspection taskId to run
   * @param request parameter of the task
   */
  runInspection(
    inspectionID: string,
    request: InspectionRunRequest,
  ): Observable<void>;

  /**
   * Request dry run a task.
   * Expected called endpoint: POST /api/v3/inspection/<inspection-id>/dryrun
   * @param inspectionID inspection taskId to dryrun
   * @param request parameter of the task
   */
  dryRunInspection(
    inspectionID: string,
    request: InspectionDryRunRequest,
  ): Observable<InspectionDryRunResponse>;

  /**
   * Download the inspection data with specified inspectionID in parallel.
   * Expected called endpoint:
   *   GET /api/v3/inspection/<inspection-id>/metadata
   *   GET /api/v3/inspection/<inspection-id>/data
   *
   * @param inspectionID inspection ID to download the data.
   * @param reporter task progress reporter.
   */
  getInspectionData(
    inspectionID: string,
    reporter: DownloadProgressReporter,
  ): Observable<{ fileName: string; content: Blob }>;

  /**
   * Cancel the inspection task.
   * Expected called endpoint: POST /api/v3/inspection/<inspection-id>/cancel
   *
   * @param inspectionID inspection ID to cancel
   */
  cancelInspection(inspectionID: string): Observable<void>;

  /**
   * Get the current popup request.
   * Expected called endpoint: GET /api/v3/popup
   */
  getPopup(): Observable<PopupFormRequest | null>;

  /**
   * Validate the request for the current popup
   * Expected called endpoint: POST /api/v3/popup/validate
   */
  validatePopupAnswer(
    answer: PopupAnswerResponse,
  ): Observable<PopupAnswerValidationResult>;

  /**
   * Answer the current request for the popup
   * Expected called endpoint: POST /api/v3/popup/answer
   */
  answerPopup(answer: PopupAnswerResponse): Observable<void>;

  /**
   * Upload the file as the one bound to the token.
   */
  uploadFile(token: UploadToken, file: File): Observable<HttpEvent<unknown>>;
}
