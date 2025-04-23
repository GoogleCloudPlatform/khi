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
  GetInspectionResponse,
  GetInspectionTypesResponse,
} from 'src/app/common/schema/api-types';

/**
 * BackendConnectionService communicates the backend continuously and emit the latest information.
 */
export interface BackendConnectionService {
  /**
   * Return an observable to monitor the available task types on tge backend.
   */
  inspectionTypes(): Observable<GetInspectionTypesResponse>;

  /**
   * Return an observable to monitor the task lists on the backend.
   */
  tasks(): Observable<GetInspectionResponse>;
}
