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
  PROGRESS_POLLING_INTERVAL,
  LIST_INSPECTION_TYPES_RETRY_TIME,
} from './backend-sync.service';
import { BACKEND_API, BackendAPI } from './backend-api-interface';
import { defer, of, throwError, EMPTY } from 'rxjs';
import { BackendConnectionStatus } from './backend-sync-interface';

describe('BackendSyncService', () => {
  let service: BackendSyncServiceImpl;
  let mockBackendApi: jasmine.SpyObj<BackendAPI>;

  beforeEach(() => {
    mockBackendApi = jasmine.createSpyObj('BackendAPI', [
      'getInspectionTypes',
      'getInspections',
    ]);
    mockBackendApi.getInspectionTypes.and.returnValue(of({ types: [] }));
    mockBackendApi.getInspections.and.returnValue(
      of({
        inspections: {},
        serverStat: { currentMemoryUsage: 0, totalMemory: 0 },
      }),
    );

    TestBed.configureTestingModule({
      providers: [
        BackendSyncServiceImpl,
        { provide: BACKEND_API, useValue: mockBackendApi },
      ],
    });
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

  it('should become Connected when tasks succeeds', fakeAsync(() => {
    service = TestBed.inject(BackendSyncServiceImpl);
    service.tasks.value();
    tick(PROGRESS_POLLING_INTERVAL);
    expect(service.connectionStatus()).toBe(BackendConnectionStatus.Connected);
  }));

  it('should become Disconnected when getInspectionTypes fails', fakeAsync(() => {
    mockBackendApi.getInspectionTypes.and.returnValue(
      throwError(() => new Error('API Error')),
    );
    mockBackendApi.getInspections.and.returnValue(EMPTY);

    service = TestBed.inject(BackendSyncServiceImpl);
    service.inspectionTypes.value();
    tick();

    expect(service.connectionStatus()).toBe(
      BackendConnectionStatus.Disconnected,
    );
  }));

  it('should become Disconnected when getInspections fails', fakeAsync(() => {
    mockBackendApi.getInspections.and.returnValue(
      throwError(() => new Error('API Error')),
    );

    service = TestBed.inject(BackendSyncServiceImpl);
    service.tasks.value();
    tick(PROGRESS_POLLING_INTERVAL);

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
    mockBackendApi.getInspections.and.returnValue(EMPTY);

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

  it('should recover and become Connected when tasks succeeds after failure', fakeAsync(() => {
    let callCount = 0;
    mockBackendApi.getInspections.and.returnValue(
      defer(() => {
        callCount++;
        if (callCount === 1) {
          return throwError(() => new Error('API Error'));
        }
        return of({
          inspections: {},
          serverStat: { currentMemoryUsage: 0, totalMemory: 0 },
        });
      }),
    );
    mockBackendApi.getInspectionTypes.and.returnValue(EMPTY);

    service = TestBed.inject(BackendSyncServiceImpl);
    service.tasks.value();

    tick(); // Process timer(0)

    // First call happens immediately with timer(0) and fails
    expect(service.connectionStatus()).toBe(
      BackendConnectionStatus.Disconnected,
    );

    // Wait for the next polling interval
    tick(PROGRESS_POLLING_INTERVAL);

    // Second call succeeds
    expect(service.connectionStatus()).toBe(BackendConnectionStatus.Connected);
  }));
});
