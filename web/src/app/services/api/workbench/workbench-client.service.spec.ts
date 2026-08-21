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
  FilterResultMode,
  OpenWorkbenchResponse_Stage,
  SparseBitsetSchema,
} from 'src/app/generated/api/v1/workbench_pb';
import { create } from '@bufbuild/protobuf';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { UserIdentityService } from 'src/app/services/api/workbench/user-identity.service';
import {
  WorkbenchClientService,
  WorkbenchOpenProgressCallback,
} from 'src/app/services/api/workbench/workbench-client.service';

describe('WorkbenchClientService', () => {
  let service: WorkbenchClientService;
  let mockConnectClient: jasmine.SpyObj<ConnectClientService>;
  let mockUserIdentity: { readonly userId: string };

  beforeEach(() => {
    mockUserIdentity = { userId: 'usr-1' };
    mockConnectClient = jasmine.createSpyObj('ConnectClientService', [], {
      workbenchClient: {
        openWorkbench: jasmine.createSpy('openWorkbench'),
        heartbeatWorkbench: jasmine.createSpy('heartbeatWorkbench'),
        readStructYAML: jasmine.createSpy('readStructYAML'),
        filterTimeline: jasmine.createSpy('filterTimeline'),
        closeWorkbench: jasmine.createSpy('closeWorkbench'),
      },
    });

    TestBed.configureTestingModule({
      providers: [
        WorkbenchClientService,
        { provide: ConnectClientService, useValue: mockConnectClient },
        { provide: UserIdentityService, useValue: mockUserIdentity },
      ],
    });
    service = TestBed.inject(WorkbenchClientService);
  });

  afterEach(() => {
    service.ngOnDestroy();
  });

  it('should be created and have inactive initial state', () => {
    expect(service).toBeTruthy();
    expect(service.activeWorkbenchId()).toBeNull();
    expect(service.isWorkbenchActive()).toBeFalse();
  });

  it('should stream progress events and set active workbench on openWorkbench', async () => {
    async function* mockStream() {
      yield {
        $typeName: 'api.v1.OpenWorkbenchResponse' as const,
        stage: OpenWorkbenchResponse_Stage.READING_FILE,
        progressPercentage: 10,
        message: 'Reading...',
      };
      yield {
        $typeName: 'api.v1.OpenWorkbenchResponse' as const,
        stage: OpenWorkbenchResponse_Stage.READY,
        progressPercentage: 100,
        message: 'Ready!',
        workbenchId: 'usr-1-session-0',
      };
    }

    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(mockStream());

    const progressUpdates: {
      stage: OpenWorkbenchResponse_Stage;
      pct: number;
    }[] = [];
    const onProgress: WorkbenchOpenProgressCallback = (_msg, pct, stage) => {
      progressUpdates.push({ stage, pct });
    };

    const workbenchId = await service.openWorkbench(
      'session-0',
      'inspection-1',
      onProgress,
    );

    expect(workbenchId).toBe('usr-1-session-0');
    expect(service.activeWorkbenchId()).toBe('usr-1-session-0');
    expect(service.isWorkbenchActive()).toBeTrue();
    expect(progressUpdates.length).toBe(2);
    expect(progressUpdates[0].stage).toBe(
      OpenWorkbenchResponse_Stage.READING_FILE,
    );
    expect(progressUpdates[1].stage).toBe(OpenWorkbenchResponse_Stage.READY);
  });

  it('should invoke heartbeat periodically after workbench is opened', fakeAsync(() => {
    async function* mockStream() {
      yield {
        $typeName: 'api.v1.OpenWorkbenchResponse' as const,
        stage: OpenWorkbenchResponse_Stage.READY,
        progressPercentage: 100,
        message: 'Ready!',
        workbenchId: 'usr-1-session-0',
      };
    }

    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(mockStream());
    (
      mockConnectClient.workbenchClient.heartbeatWorkbench as jasmine.Spy
    ).and.returnValue(Promise.resolve({ active: true }));

    service.openWorkbench('session-0', 'inspection-1');
    tick();

    expect(
      mockConnectClient.workbenchClient.heartbeatWorkbench,
    ).not.toHaveBeenCalled();

    // Advance 15 seconds
    tick(15000);
    expect(
      mockConnectClient.workbenchClient.heartbeatWorkbench,
    ).toHaveBeenCalledWith({
      workbenchId: 'usr-1-session-0',
    });

    // Advance another 15 seconds
    tick(15000);
    expect(
      mockConnectClient.workbenchClient.heartbeatWorkbench,
    ).toHaveBeenCalledTimes(2);
  }));

  it('should close workbench and clear active state on closeWorkbench', async () => {
    (
      mockConnectClient.workbenchClient.closeWorkbench as jasmine.Spy
    ).and.returnValue(Promise.resolve({ closed: true }));

    await service.closeWorkbench('usr-1-session-0');

    expect(
      mockConnectClient.workbenchClient.closeWorkbench,
    ).toHaveBeenCalledWith({
      workbenchId: 'usr-1-session-0',
    });
    expect(service.activeWorkbenchId()).toBeNull();
    expect(service.isWorkbenchActive()).toBeFalse();
  });

  it('should return empty string without RPC call if structId is 0 or negative', async () => {
    const yaml = await service.readStructYAML(0);
    expect(yaml).toBe('');
    expect(
      mockConnectClient.workbenchClient.readStructYAML,
    ).not.toHaveBeenCalled();
  });

  it('should throw error on readStructYAML if no workbench session is active', async () => {
    await expectAsync(service.readStructYAML(42)).toBeRejectedWithError(
      'No active Workbench session found.',
    );
  });

  it('should call readStructYAML on backend client with active workbench ID and return yaml string', async () => {
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(
      (async function* () {
        yield {
          stage: OpenWorkbenchResponse_Stage.READY,
          progressPercentage: 100,
          message: 'Ready',
          workbenchId: 'usr-1-session-0',
        };
      })(),
    );
    (
      mockConnectClient.workbenchClient.readStructYAML as jasmine.Spy
    ).and.returnValue(Promise.resolve({ yaml: 'message: hello\n' }));

    await service.openWorkbench('session-0', 'inspection-1');

    const yaml = await service.readStructYAML(42);

    expect(
      mockConnectClient.workbenchClient.readStructYAML,
    ).toHaveBeenCalledWith({
      workbenchId: 'usr-1-session-0',
      structId: 42,
    });
    expect(yaml).toBe('message: hello\n');
  });

  it('should cache readStructYAML responses and avoid duplicate RPC calls', async () => {
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(
      (async function* () {
        yield {
          stage: OpenWorkbenchResponse_Stage.READY,
          progressPercentage: 100,
          message: 'Ready',
          workbenchId: 'usr-1-session-0',
        };
      })(),
    );
    (
      mockConnectClient.workbenchClient.readStructYAML as jasmine.Spy
    ).and.returnValue(Promise.resolve({ yaml: 'message: cached\n' }));

    await service.openWorkbench('session-0', 'inspection-1');

    const yaml1 = await service.readStructYAML(100);
    const yaml2 = await service.readStructYAML(100);

    expect(yaml1).toBe('message: cached\n');
    expect(yaml2).toBe('message: cached\n');
    expect(
      mockConnectClient.workbenchClient.readStructYAML,
    ).toHaveBeenCalledTimes(1);
  });

  it('should stream progress and return final result on filterTimeline', async () => {
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(
      (async function* () {
        yield {
          stage: OpenWorkbenchResponse_Stage.READY,
          progressPercentage: 100,
          message: 'Ready',
          workbenchId: 'usr-1-session-0',
        };
      })(),
    );

    async function* mockFilterStream() {
      yield {
        payload: {
          case: 'progress' as const,
          value: {
            stageName: 'Timeline CEL filter',
            current: 10,
            total: 100,
          },
        },
      };
      yield {
        payload: {
          case: 'result' as const,
          value: {
            timelineMode: FilterResultMode.INCLUDE,
            timelineBitset: create(SparseBitsetSchema, {
              indices: [0],
              masks: [0xe], // 1, 2, 3
            }),
            logMode: FilterResultMode.INCLUDE,
            logBitset: create(SparseBitsetSchema, {
              indices: [0],
              masks: [(1 << 10) | (1 << 20)],
            }),
          },
        },
      };
    }

    (
      mockConnectClient.workbenchClient.filterTimeline as jasmine.Spy
    ).and.returnValue(mockFilterStream());

    await service.openWorkbench('session-0', 'inspection-1');

    const progressReports: { stage: string; current: number; total: number }[] =
      [];
    const result = await service.filterTimeline(
      {
        timelineQuery: 'name == "pod-a"',
        excludeNoLogs: true,
      },
      (stage, current, total) => {
        progressReports.push({ stage, current, total });
      },
    );

    expect(
      mockConnectClient.workbenchClient.filterTimeline,
    ).toHaveBeenCalledWith(
      {
        workbenchId: 'usr-1-session-0',
        timelineQuery: 'name == "pod-a"',
        timelineExclusionQuery: '',
        logQuery: '',
        excludeNoLogs: true,
      },
      jasmine.any(Object),
    );

    expect(progressReports.length).toBe(1);
    expect(progressReports[0]).toEqual({
      stage: 'Timeline CEL filter',
      current: 10,
      total: 100,
    });
    expect(result.timelineMode).toBe(FilterResultMode.INCLUDE);
    expect(result.timelineBitset?.indices).toEqual([0]);
    expect(result.timelineBitset?.masks).toEqual([0xe]);
    expect(result.logMode).toBe(FilterResultMode.INCLUDE);
    expect(result.logBitset?.indices).toEqual([0]);
    expect(result.logBitset?.masks).toEqual([(1 << 10) | (1 << 20)]);
  });
});
