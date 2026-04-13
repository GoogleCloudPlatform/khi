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

import { ResourceRef, Signal } from '@angular/core';
import {
  GetInspectionResponse,
  GetInspectionTypesResponse,
} from 'src/app/common/schema/api-types';

/**
 * Connection status to the backend.
 */
export enum BackendConnectionStatus {
  Connecting = 'connecting',
  Connected = 'connected',
  Disconnected = 'disconnected',
}

/**
 * BackendSyncService maintains the latest information from the backend.
 */
export interface BackendSyncService {
  /**
   * Current connection status to the backend.
   */
  readonly connectionStatus: Signal<BackendConnectionStatus>;

  /**
   * Monitored available task types on the backend.
   */
  readonly inspectionTypes: ResourceRef<GetInspectionTypesResponse>;

  /**
   * Monitored task lists on the backend.
   */
  readonly tasks: ResourceRef<GetInspectionResponse>;
}
