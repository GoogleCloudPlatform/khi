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

import { Inject, Injectable, InjectionToken } from '@angular/core';
import {
  Observable,
  exhaustMap,
  interval,
  retry,
  shareReplay,
  tap,
} from 'rxjs';
import { BACKEND_API, BackendAPI } from './backend-api-interface';
import { BackendConnectionService } from './backend-connection-interface';
import {
  GetInspectionTypesResponse,
  GetInspectionTasksResponse,
} from 'src/app/common/schema/api-types';

/**
 * Angular injection token for BackendConnectionService.
 */
export const BACKEND_CONNECTION = new InjectionToken<BackendConnectionService>(
  'BACKEND_CONNECTION',
);

/**
 * BackendConnectionService provides observables with polling backend endpoints.
 */
@Injectable()
export class BackendConnectionServiceImpl implements BackendConnectionService {
  /**
   * Interval to poll task progresses.
   */
  static readonly PROGRESS_POLLING_INTERVAL = 1000;

  /**
   * Interval to poll the list of inspection types.
   */
  static readonly LIST_INSPECTION_TYPES_RETRY_TIME = 1000;

  private inspectionTypesObservable = this.backendApi.getInspectionTypes().pipe(
    retry({
      delay: BackendConnectionServiceImpl.LIST_INSPECTION_TYPES_RETRY_TIME,
    }),
    shareReplay({
      bufferSize: 1,
      // refCount is explcitly false to prevent waiting the next poll when a new subscriber added when there is no subscriber registered.
      refCount: false,
    }),
  );

  private taskProgressObservable = interval(
    BackendConnectionServiceImpl.PROGRESS_POLLING_INTERVAL,
  ).pipe(
    exhaustMap(() => this.backendApi.getTaskStatuses()),
    tap({
      error: (err) => {
        console.warn(
          `Failed to refresh task progerss status:\n` + JSON.stringify(err),
        );
      },
    }),
    shareReplay({
      bufferSize: 1,
      // refCount is explcitly false to prevent waiting the next poll when a new subscriber added when there is no subscriber registered.
      refCount: false,
    }),
    retry(),
  );

  constructor(@Inject(BACKEND_API) private backendApi: BackendAPI) {}

  inspectionTypes(): Observable<GetInspectionTypesResponse> {
    return this.inspectionTypesObservable;
  }
  tasks(): Observable<GetInspectionTasksResponse> {
    return this.taskProgressObservable;
  }
}
