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

import { TestBed, fakeAsync, tick } from '@angular/core/testing';
import {
  BackendSyncServiceImpl,
  LIST_INSPECTION_TYPES_RETRY_TIME,
} from './backend-sync.service';
import { BACKEND_API, BackendAPI } from './backend-api-interface';
import { defer, of, throwError } from 'rxjs';
import { BackendConnectionStatus } from './backend-sync-interface';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { ServerStat } from 'src/app/generated/api/v1/server_status_pb';
import { InspectionListItem } from 'src/app/generated/api/v1/inspection_pb';

describe('BackendSyncService', () => {
  let service: BackendSyncServiceImpl;
  let mockBackendApi: jasmine.SpyObj<BackendAPI>;
  let mockConnectClient: {
    serverStatusClient: {
      watchServerStat: jasmine.Spy;
    };
    inspectionClient: {
      watchInspections: jasmine.Spy;
    };
  };

  beforeEach(() => {
    mockBackendApi = jasmine.createSpyObj('BackendAPI', ['getInspectionTypes']);
    mockBackendApi.getInspectionTypes.and.returnValue(of({ types: [] }));

    mockConnectClient = {
      serverStatusClient: {
        watchServerStat: jasmine
          .createSpy('watchServerStat')
          .and.callFake((_req: unknown, opts?: { signal?: AbortSignal }) => {
            return (async function* () {
              yield {
                serverStat: {
                  currentMemoryUsage: 1024n * 1024n * 50n,
                  totalMemory: 1024n * 1024n * 1024n * 16n,
                  cpuUsagePercentage: 20.5,
                } as ServerStat,
              };
              if (opts?.signal) {
                await new Promise<void>((resolve) => {
                  opts.signal!.addEventListener('abort', () => resolve(), {
                    once: true,
                  });
                });
              }
            })();
          }),
      },
      inspectionClient: {
        watchInspections: jasmine
          .createSpy('watchInspections')
          .and.callFake((_req: unknown, opts?: { signal?: AbortSignal }) => {
            return (async function* () {
              yield {
                inspections: [
                  {
                    id: 'insp-1',
                  } as InspectionListItem,
                ],
              };
              if (opts?.signal) {
                await new Promise<void>((resolve) => {
                  opts.signal!.addEventListener('abort', () => resolve(), {
                    once: true,
                  });
                });
              }
            })();
          }),
      },
    };

    TestBed.configureTestingModule({
      providers: [
        BackendSyncServiceImpl,
        { provide: BACKEND_API, useValue: mockBackendApi },
        { provide: ConnectClientService, useValue: mockConnectClient },
      ],
    });
  });

  afterEach(() => {
    service?.ngOnDestroy();
  });

  it('should be created', () => {
    service = TestBed.inject(BackendSyncServiceImpl);
    expect(service).toBeTruthy();
  });

  it('should have initial connection status as Connecting', () => {
    service = TestBed.inject(BackendSyncServiceImpl);
    expect(service.connectionStatus()).toBe(BackendConnectionStatus.Connecting);
  });

  it('should become Connected when inspectionTypes succeeds', fakeAsync(() => {
    service = TestBed.inject(BackendSyncServiceImpl);
    service.inspectionTypes.value();
    tick();
    expect(service.connectionStatus()).toBe(BackendConnectionStatus.Connected);
  }));

  it('should become Disconnected when getInspectionTypes fails', fakeAsync(() => {
    mockBackendApi.getInspectionTypes.and.returnValue(
      throwError(() => new Error('API Error')),
    );

    service = TestBed.inject(BackendSyncServiceImpl);
    service.inspectionTypes.value();
    tick();

    expect(service.connectionStatus()).toBe(
      BackendConnectionStatus.Disconnected,
    );
  }));

  it('should retry getInspectionTypes on failure', fakeAsync(() => {
    let callCount = 0;
    mockBackendApi.getInspectionTypes.and.returnValue(
      defer(() => {
        callCount++;
        if (callCount === 1) {
          return throwError(() => new Error('API Error'));
        }
        return of({ types: [] });
      }),
    );

    service = TestBed.inject(BackendSyncServiceImpl);
    service.inspectionTypes.value();
    tick(); // First call fails

    expect(callCount).toBe(1);
    expect(service.connectionStatus()).toBe(
      BackendConnectionStatus.Disconnected,
    );

    tick(LIST_INSPECTION_TYPES_RETRY_TIME); // Wait for retry

    expect(callCount).toBe(2);
    expect(service.connectionStatus()).toBe(BackendConnectionStatus.Connected);
  }));

  it('should update inspections from watchInspections', async () => {
    service = TestBed.inject(BackendSyncServiceImpl);

    // Wait microtasks for async generator
    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(service.inspections()).toEqual([
      jasmine.objectContaining({
        id: 'insp-1',
      }),
    ]);
  });

  it('should update serverStat from watchServerStat', async () => {
    service = TestBed.inject(BackendSyncServiceImpl);

    // Wait microtasks for async generator
    await new Promise((resolve) => setTimeout(resolve, 10));

    expect(service.serverStat()).toEqual(
      jasmine.objectContaining({
        currentMemoryUsage: 1024n * 1024n * 50n,
        totalMemory: 1024n * 1024n * 1024n * 16n,
        cpuUsagePercentage: 20.5,
      }),
    );
  });
});
