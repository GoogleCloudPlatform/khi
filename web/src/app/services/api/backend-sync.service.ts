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

import { Injectable, InjectionToken, inject, signal } from '@angular/core';
import { rxResource } from '@angular/core/rxjs-interop';
import { catchError, EMPTY, exhaustMap, retry, tap, timer } from 'rxjs';
import { BACKEND_API, BackendAPI } from './backend-api-interface';
import {
  BackendSyncService,
  BackendConnectionStatus,
} from './backend-sync-interface';
import {
  GetInspectionTypesResponse,
  GetInspectionResponse,
} from 'src/app/common/schema/api-types';

/**
 * Angular injection token for BackendSyncService.
 */
export const BACKEND_SYNC = new InjectionToken<BackendSyncService>(
  'BACKEND_SYNC',
);

/**
 * Interval to poll task progresses.
 */
export const PROGRESS_POLLING_INTERVAL = 1000;

/**
 * Interval to poll the list of inspection types.
 */
export const LIST_INSPECTION_TYPES_RETRY_TIME = 1000;

/**
 * BackendSyncServiceImpl provides resources by polling backend endpoints.
 */
@Injectable()
export class BackendSyncServiceImpl implements BackendSyncService {
  private readonly backendApi = inject<BackendAPI>(BACKEND_API);

  /**
   * Signal to manage the connection status internally.
   */
  private readonly connectionStatusSignal = signal<BackendConnectionStatus>(
    BackendConnectionStatus.Connecting,
  );

  /**
   * Signal of the current backend connection status.
   */
  readonly connectionStatus = this.connectionStatusSignal.asReadonly();

  /**
   * Resource for the list of available inspection types.
   */
  readonly inspectionTypes = rxResource<GetInspectionTypesResponse, void>({
    defaultValue: { types: [] },
    stream: () =>
      this.backendApi.getInspectionTypes().pipe(
        tap(this.getStatusUpdater('getInspectionTypes')),
        retry({
          delay: LIST_INSPECTION_TYPES_RETRY_TIME,
        }),
      ),
  });

  /**
   * Resource for the list of inspections and their tasks.
   */
  readonly tasks = rxResource<GetInspectionResponse, void>({
    defaultValue: {
      inspections: {},
      serverStat: { currentMemoryUsage: 0, totalMemory: 0 },
    },
    stream: () =>
      timer(0, PROGRESS_POLLING_INTERVAL).pipe(
        exhaustMap(() =>
          this.backendApi.getInspections().pipe(
            tap(this.getStatusUpdater('getInspections')),
            catchError(() => EMPTY),
          ),
        ),
      ),
  });

  private getStatusUpdater(context: string) {
    return {
      next: () =>
        this.connectionStatusSignal.set(BackendConnectionStatus.Connected),
      error: (err: unknown) => {
        console.warn(`Failed in ${context}:`, err);
        this.connectionStatusSignal.set(BackendConnectionStatus.Disconnected);
      },
    };
  }
}
