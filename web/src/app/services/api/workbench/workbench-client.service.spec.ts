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
  WatchIndexProgressResponse_IndexState,
} from 'src/app/generated/api/v1/workbench_pb';
import { GetArchitectureGraphResponseSchema } from 'src/app/generated/api/v1/architecture_graph_pb';
import { SparseBitsetSchema } from 'src/app/generated/api/v1/sparse_bitset_pb';
import { create } from '@bufbuild/protobuf';
import { ConnectError, Code } from '@connectrpc/connect';
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
        watchIndexProgress: jasmine.createSpy('watchIndexProgress'),
        heartbeatWorkbench: jasmine.createSpy('heartbeatWorkbench'),
        readStructYAML: jasmine.createSpy('readStructYAML'),
        filterTimeline: jasmine.createSpy('filterTimeline'),
        getArchitectureGraph: jasmine.createSpy('getArchitectureGraph'),
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

  it('should watch index progress and update index state signals', async () => {
    async function* mockIndexStream() {
      yield {
        state: WatchIndexProgressResponse_IndexState.BUILDING,
        progressPercentage: 40,
        message: 'Building index...',
      };
      yield {
        state: WatchIndexProgressResponse_IndexState.READY,
        progressPercentage: 100,
        message: 'Search index ready.',
      };
    }

    (
      mockConnectClient.workbenchClient.watchIndexProgress as jasmine.Spy
    ).and.returnValue(mockIndexStream());

    expect(service.indexState()).toBe(
      WatchIndexProgressResponse_IndexState.UNSPECIFIED,
    );
    expect(service.isIndexBuilding()).toBeFalse();
    expect(service.isIndexReady()).toBeFalse();

    await service.watchIndexProgress('test-workbench-1');

    expect(
      mockConnectClient.workbenchClient.watchIndexProgress,
    ).toHaveBeenCalledWith(
      { workbenchId: 'test-workbench-1' },
      jasmine.any(Object),
    );
    expect(service.indexState()).toBe(
      WatchIndexProgressResponse_IndexState.READY,
    );
    expect(service.indexProgressPercentage()).toBe(100);
    expect(service.indexMessage()).toBe('Search index ready.');
    expect(service.isIndexReady()).toBeTrue();
    expect(service.isIndexBuilding()).toBeFalse();
  });

  it('should reconnect on 30s stream cycle termination until index reaches READY', async () => {
    let invocationCount = 0;
    (
      mockConnectClient.workbenchClient.watchIndexProgress as jasmine.Spy
    ).and.callFake(() => {
      invocationCount++;
      if (invocationCount === 1) {
        // First 30s cycle stream: ends while still BUILDING
        return (async function* () {
          yield {
            state: WatchIndexProgressResponse_IndexState.BUILDING,
            progressPercentage: 50,
            message: 'Building trigrams...',
          };
        })();
      }
      // Reconnected second cycle: reaches READY
      return (async function* () {
        yield {
          state: WatchIndexProgressResponse_IndexState.READY,
          progressPercentage: 100,
          message: 'Index complete.',
        };
      })();
    });

    await service.watchIndexProgress('usr-1-session-0');

    expect(invocationCount).toBe(2);
    expect(service.indexState()).toBe(
      WatchIndexProgressResponse_IndexState.READY,
    );
    expect(service.isIndexReady()).toBeTrue();
  });

  it('should abort active index progress stream on closeWorkbench', async () => {
    let capturedSignal: AbortSignal | undefined;
    async function* mockIndexStream() {
      yield {
        state: WatchIndexProgressResponse_IndexState.BUILDING,
        progressPercentage: 50,
        message: 'Building...',
      };
    }
    (
      mockConnectClient.workbenchClient.watchIndexProgress as jasmine.Spy
    ).and.callFake((_req: unknown, opts: { signal?: AbortSignal }) => {
      capturedSignal = opts?.signal;
      return mockIndexStream();
    });

    async function* mockOpenStream() {
      yield {
        stage: OpenWorkbenchResponse_Stage.READY,
        progressPercentage: 100,
        message: 'Ready',
        workbenchId: 'wb-1',
      };
    }
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(mockOpenStream());
    (
      mockConnectClient.workbenchClient.closeWorkbench as jasmine.Spy
    ).and.returnValue(Promise.resolve({ closed: true }));

    await service.openWorkbench('session-1', 'insp-1');
    expect(capturedSignal).toBeDefined();
    expect(capturedSignal?.aborted).toBeFalse();

    await service.closeWorkbench('wb-1');
    expect(capturedSignal?.aborted).toBeTrue();
  });

  it('should mark workbench as expired when heartbeat returns inactive', async () => {
    (
      mockConnectClient.workbenchClient.heartbeatWorkbench as jasmine.Spy
    ).and.returnValue(Promise.resolve({ active: false }));

    const isActive = await service.heartbeat('wb-1');

    expect(isActive).toBeFalse();
    expect(service.isWorkbenchExpired()).toBeTrue();
  });

  it('should mark workbench as expired when heartbeat throws ConnectError Code.NotFound', async () => {
    const notFoundErr = new ConnectError('session expired', Code.NotFound);
    (
      mockConnectClient.workbenchClient.heartbeatWorkbench as jasmine.Spy
    ).and.returnValue(Promise.reject(notFoundErr));

    const isActive = await service.heartbeat('wb-1');

    expect(isActive).toBeFalse();
    expect(service.isWorkbenchExpired()).toBeTrue();
  });

  it('should mark workbench as expired when readStructYAML throws ConnectError Code.NotFound', async () => {
    async function* mockOpenStream() {
      yield {
        stage: OpenWorkbenchResponse_Stage.READY,
        progressPercentage: 100,
        message: 'Ready',
        workbenchId: 'wb-active',
      };
    }
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(mockOpenStream());
    const notFoundErr = new ConnectError('not found', Code.NotFound);
    (
      mockConnectClient.workbenchClient.readStructYAML as jasmine.Spy
    ).and.returnValue(Promise.reject(notFoundErr));

    await service.openWorkbench('s-1', 'i-1');
    expect(service.isWorkbenchExpired()).toBeFalse();

    await expectAsync(service.readStructYAML(10)).toBeRejectedWith(notFoundErr);
    expect(service.isWorkbenchExpired()).toBeTrue();
  });

  it('should mark workbench as expired when filterTimeline throws ConnectError Code.NotFound', async () => {
    async function* mockOpenStream() {
      yield {
        stage: OpenWorkbenchResponse_Stage.READY,
        progressPercentage: 100,
        message: 'Ready',
        workbenchId: 'wb-active',
      };
    }
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(mockOpenStream());
    const notFoundErr = new ConnectError('not found', Code.NotFound);
    (
      mockConnectClient.workbenchClient.filterTimeline as jasmine.Spy
    ).and.returnValue(
      (async function* () {
        throw notFoundErr;
      })(),
    );

    await service.openWorkbench('s-1', 'i-1');
    expect(service.isWorkbenchExpired()).toBeFalse();

    await expectAsync(service.filterTimeline({})).toBeRejectedWith(notFoundErr);
    expect(service.isWorkbenchExpired()).toBeTrue();
  });

  it('should reopen workbench using stored session and inspection IDs and reset expired flag', async () => {
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.callFake(() => {
      return (async function* () {
        yield {
          stage: OpenWorkbenchResponse_Stage.READY,
          progressPercentage: 100,
          message: 'Ready',
          workbenchId: 'wb-reopened',
        };
      })();
    });

    await service.openWorkbench('my-session', 'my-inspection');
    (
      service as unknown as {
        isWorkbenchExpiredSignal: { set: (v: boolean) => void };
      }
    ).isWorkbenchExpiredSignal.set(true);
    expect(service.isWorkbenchExpired()).toBeTrue();

    const reopenedId = await service.reopenWorkbench();

    expect(reopenedId).toBe('wb-reopened');
    expect(service.activeWorkbenchId()).toBe('wb-reopened');
    expect(service.isWorkbenchExpired()).toBeFalse();
  });

  it('should deduplicate concurrent reopenWorkbench invocations', async () => {
    let callCount = 0;
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.callFake(() => {
      callCount++;
      return (async function* () {
        yield {
          stage: OpenWorkbenchResponse_Stage.READY,
          progressPercentage: 100,
          message: 'Ready',
          workbenchId: 'wb-reopened',
        };
      })();
    });

    await service.openWorkbench('my-session', 'my-inspection');

    const [id1, id2] = await Promise.all([
      service.reopenWorkbench(),
      service.reopenWorkbench(),
    ]);

    expect(id1).toBe('wb-reopened');
    expect(id2).toBe('wb-reopened');
    expect(callCount).toBe(2); // 1 for initial open, 1 for deduplicated reopen
  });

  it('should throw error when reopenWorkbench is called without prior openWorkbench', async () => {
    await expectAsync(service.reopenWorkbench()).toBeRejectedWithError(
      'No active or previous inspection session available to reopen.',
    );
  });

  it('should reset session IDs and expiration signals on closeWorkbench', async () => {
    async function* mockOpenStream() {
      yield {
        stage: OpenWorkbenchResponse_Stage.READY,
        progressPercentage: 100,
        message: 'Ready',
        workbenchId: 'wb-1',
      };
    }
    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(mockOpenStream());
    (
      mockConnectClient.workbenchClient.closeWorkbench as jasmine.Spy
    ).and.returnValue(Promise.resolve({ closed: true }));

    await service.openWorkbench('session-1', 'insp-1');
    await service.closeWorkbench('wb-1');

    await expectAsync(service.reopenWorkbench()).toBeRejectedWithError(
      'No active or previous inspection session available to reopen.',
    );
    expect(service.isWorkbenchExpired()).toBeFalse();
    expect(service.isReopening()).toBeFalse();
  });

  it('should not restart or trigger heartbeat on visibilitychange when session is expired', async () => {
    (
      mockConnectClient.workbenchClient.heartbeatWorkbench as jasmine.Spy
    ).and.returnValue(Promise.resolve({ active: false }));

    await service.heartbeat('wb-expired');
    expect(service.isWorkbenchExpired()).toBeTrue();

    const heartbeatSpy = mockConnectClient.workbenchClient
      .heartbeatWorkbench as jasmine.Spy;
    heartbeatSpy.calls.reset();

    // Dispatch visibilitychange when document is visible
    document.dispatchEvent(new Event('visibilitychange'));

    expect(heartbeatSpy).not.toHaveBeenCalled();
  });

  it('should close new workbench and not activate it if session was closed while openWorkbench was in flight', async () => {
    let completeStream!: () => void;
    const streamPromise = new Promise<void>((resolve) => {
      completeStream = resolve;
    });

    async function* delayedOpenStream() {
      await streamPromise;
      yield {
        stage: OpenWorkbenchResponse_Stage.READY,
        progressPercentage: 100,
        message: 'Ready',
        workbenchId: 'wb-orphaned',
      };
    }

    (
      mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
    ).and.returnValue(delayedOpenStream());
    (
      mockConnectClient.workbenchClient.closeWorkbench as jasmine.Spy
    ).and.returnValue(Promise.resolve({ closed: true }));

    const openPromise = service.openWorkbench('s-1', 'i-1');

    // While open is pending, user returns to startup / closes workbench
    await service.closeWorkbench();

    // Now open response arrives
    completeStream();
    const result = await openPromise;

    expect(result).toBeUndefined();
    expect(service.activeWorkbenchId()).toBeNull();
    expect(
      mockConnectClient.workbenchClient.closeWorkbench,
    ).toHaveBeenCalledWith({
      workbenchId: 'wb-orphaned',
    });
  });

  describe('getArchitectureGraph', () => {
    it('should throw error when no active workbench session', async () => {
      await expectAsync(
        service.getArchitectureGraph(1000n),
      ).toBeRejectedWithError('No active Workbench session found.');
    });

    it('should call getArchitectureGraph RPC on active workbench', async () => {
      async function* mockOpenStream() {
        yield {
          stage: OpenWorkbenchResponse_Stage.READY,
          progressPercentage: 100,
          message: 'Ready',
          workbenchId: 'wb-1',
        };
      }
      (
        mockConnectClient.workbenchClient.openWorkbench as jasmine.Spy
      ).and.returnValue(mockOpenStream());
      await service.openWorkbench('session-1', 'insp-1');

      const expectedResponse = create(GetArchitectureGraphResponseSchema, {
        timestampNs: 1000n,
      });
      (
        mockConnectClient.workbenchClient.getArchitectureGraph as jasmine.Spy
      ).and.returnValue(Promise.resolve(expectedResponse));

      const sparseBitset = create(SparseBitsetSchema, {
        indices: [0],
        masks: [1],
      });
      const result = await service.getArchitectureGraph(
        1000n,
        sparseBitset,
        180,
      );

      expect(result).toBe(expectedResponse);
      expect(
        mockConnectClient.workbenchClient.getArchitectureGraph,
      ).toHaveBeenCalledWith(
        {
          workbenchId: 'wb-1',
          timestampNs: 1000n,
          timelineBitset: sparseBitset,
          deletionThresholdSeconds: 180,
        },
        { signal: undefined },
      );
    });
  });
});
