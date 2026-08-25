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

import {
  Injectable,
  InjectionToken,
  OnDestroy,
  inject,
  signal,
} from '@angular/core';
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
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { ServerStat } from 'src/app/generated/api/v1/server_status_pb';

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
 * BackendSyncServiceImpl provides resources by polling backend endpoints and streaming server status.
 */
@Injectable()
export class BackendSyncServiceImpl implements BackendSyncService, OnDestroy {
  private readonly backendApi = inject<BackendAPI>(BACKEND_API);
  private readonly connectClient = inject(ConnectClientService);
  private readonly abortController = new AbortController();

  /**
   * Signal to manage host server resource statistics.
   */
  private readonly serverStatSignal = signal<ServerStat | null>(null);

  /**
   * Signal of the current host server resource statistics.
   */
  readonly serverStat = this.serverStatSignal.asReadonly();

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

  constructor() {
    void this.startWatchingServerStat(this.abortController.signal);
  }

  ngOnDestroy(): void {
    this.abortController.abort();
  }

  /**
   * Subscribes to the server status stream, automatically reconnecting on 30s cycle termination.
   */
  private async startWatchingServerStat(
    abortSignal: AbortSignal,
  ): Promise<void> {
    while (!abortSignal.aborted) {
      try {
        const stream = this.connectClient.serverStatusClient.watchServerStat(
          {},
          { signal: abortSignal },
        );
        for await (const res of stream) {
          if (abortSignal.aborted) {
            return;
          }
          if (res.serverStat) {
            this.serverStatSignal.set(res.serverStat);
          }
          this.connectionStatusSignal.set(BackendConnectionStatus.Connected);
        }
        // Wait a brief moment before reconnecting on normal stream cycle expiration
        await new Promise((resolve) => setTimeout(resolve, 100));
      } catch (err) {
        if (abortSignal.aborted) {
          return;
        }
        console.warn('Failed in watchServerStat:', err);
        this.connectionStatusSignal.set(BackendConnectionStatus.Disconnected);
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
    }
  }

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
