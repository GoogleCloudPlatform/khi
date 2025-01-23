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

import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import {
  PopupAnswerResponse,
  PopupAnswerValidationResult,
  PopupFormRequest,
} from 'src/app/common/schema/api-types';

/**
 * The injection token to get the actual PopupManager implementation.
 */
export const POPUP_MANAGER = new InjectionToken<PopupManager>('POPUP_MANAGER');

export interface PopupManager {
  /**
   * Get the observable stream to monitor popup form requests.
   */
  requests(): Observable<PopupFormRequestWithClient>;
}

export interface PopupFormRequestWithClient extends PopupFormRequest {
  client: PopupClient;
}

export interface PopupClient {
  /**
   * Validate if the content is valid or not.
   * @param data the data to verify as the response of popup request
   */
  validate(data: PopupAnswerResponse): Observable<PopupAnswerValidationResult>;

  /**
   * Send the answer for the popup request.
   * @param data
   */
  answer(data: PopupAnswerResponse): Observable<void>;
}
