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

import { TestBed } from '@angular/core/testing';
import { computed, signal, WritableSignal } from '@angular/core';
import { WorkbenchPrefetchService } from 'src/app/services/workbench-prefetch.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { Timeline, Revision, Event } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';

describe('WorkbenchPrefetchService', () => {
  let service: WorkbenchPrefetchService;
  let mockWorkbenchClient: jasmine.SpyObj<WorkbenchClientService>;
  let mockSelectedTimeline: WritableSignal<ReadonlyDomainElement<Timeline> | null>;
  let mockSelectedRevision: WritableSignal<ReadonlyDomainElement<Revision> | null>;
  let mockSelectedLog: WritableSignal<ReadonlyDomainElement<Log> | null>;
  let mockSelectedTimelinesWithChildren: WritableSignal<
    ReadonlyDomainElement<Timeline>[]
  >;
  let mockFilteredLogIds: WritableSignal<IdBitset>;
  let mockActiveWorkbenchId: WritableSignal<string | null>;
  let mockLogStore: {
    count: number;
    getLogIdByIndex: (idx: number) => number;
    getBodyStructId: (id: number) => number;
  };

  function createMockLog(
    id: number,
    logIndex: number,
    timestamp: bigint,
    structId = 0,
  ): ReadonlyDomainElement<Log> {
    return {
      id,
      logIndex,
      timestamp,
      structId,
    } as unknown as ReadonlyDomainElement<Log>;
  }

  function createMockRevision(
    id: number,
    index: number,
    changedTime: bigint,
    structId: number,
    log: ReadonlyDomainElement<Log>,
    timeline: ReadonlyDomainElement<Timeline>,
  ): ReadonlyDomainElement<Revision> {
    return {
      id,
      index,
      changedTime,
      structId,
      logId: log.id,
      logIndex: log.logIndex,
      log,
      timeline,
    } as unknown as ReadonlyDomainElement<Revision>;
  }

  function createMockEvent(
    id: number,
    timestamp: bigint,
    log: ReadonlyDomainElement<Log>,
  ): ReadonlyDomainElement<Event> {
    return {
      id,
      timestamp,
      logId: log.id,
      logIndex: log.logIndex,
      log,
    } as unknown as ReadonlyDomainElement<Event>;
  }

  function createMockTimeline(
    id: number,
    revisions: ReadonlyDomainElement<Revision>[] = [],
    events: ReadonlyDomainElement<Event>[] = [],
  ): ReadonlyDomainElement<Timeline> {
    return {
      id,
      revisions,
      events,
    } as unknown as ReadonlyDomainElement<Timeline>;
  }

  beforeEach(() => {
    mockActiveWorkbenchId = signal<string | null>('wb-test-1');
    mockWorkbenchClient = jasmine.createSpyObj(
      'WorkbenchClientService',
      ['readStructYAMLs', 'getTimelineIdsForLogs'],
      {
        activeWorkbenchId: mockActiveWorkbenchId.asReadonly(),
        isWorkbenchActive: computed(() => mockActiveWorkbenchId() !== null),
      },
    );
    mockWorkbenchClient.readStructYAMLs.and.resolveTo(new Map());
    mockWorkbenchClient.getTimelineIdsForLogs.and.resolveTo(new Map());

    mockSelectedTimeline = signal<ReadonlyDomainElement<Timeline> | null>(null);
    mockSelectedRevision = signal<ReadonlyDomainElement<Revision> | null>(null);
    mockSelectedLog = signal<ReadonlyDomainElement<Log> | null>(null);
    mockSelectedTimelinesWithChildren = signal<
      ReadonlyDomainElement<Timeline>[]
    >([]);
    mockFilteredLogIds = signal<IdBitset>(IdBitset.createEmpty());
    mockLogStore = {
      count: 0,
      getLogIdByIndex: (idx: number) => idx + 1,
      getBodyStructId: (id: number) => 200 + (id - 1),
    };

    TestBed.configureTestingModule({
      providers: [
        WorkbenchPrefetchService,
        {
          provide: WorkbenchClientService,
          useValue: mockWorkbenchClient,
        },
        {
          provide: SelectionManager,
          useValue: {
            selectedTimeline: mockSelectedTimeline,
            selectedRevision: mockSelectedRevision,
            selectedLog: mockSelectedLog,
            selectedTimelinesWithChildren: mockSelectedTimelinesWithChildren,
          },
        },
        {
          provide: InspectionDataStore,
          useValue: {
            inspectionData: signal({
              logStore: mockLogStore,
            }),
            timelineView: signal({
              filteredLogIds: mockFilteredLogIds,
            }),
          },
        },
      ],
    });

    service = TestBed.inject(WorkbenchPrefetchService);
  });

  describe('prefetchTimeline', () => {
    it('should prefetch struct IDs and log IDs from revisions and events sorted by timestamp up to limit', () => {
      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revLog1 = createMockLog(1, 0, 100n, 10);
      const revLog2 = createMockLog(2, 1, 300n, 20);
      const eventLog1 = createMockLog(3, 2, 200n, 30);

      const rev1 = createMockRevision(1, 0, 100n, 101, revLog1, mockTimeline);
      const rev2 = createMockRevision(2, 1, 300n, 102, revLog2, mockTimeline);
      const event1 = createMockEvent(1, 200n, eventLog1);

      const timelineWithItems = createMockTimeline(1, [rev1, rev2], [event1]);

      service.prefetchTimeline(timelineWithItems);

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalledWith([
        101, 10, 30, 102, 20,
      ]);
      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalledWith([
        1, 3, 2,
      ]);
    });

    it('should cap prefetched timeline struct IDs and log IDs at PREFETCH_TIMELINE_LIMIT', () => {
      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revisions: ReadonlyDomainElement<Revision>[] = [];

      for (let i = 0; i < 60; i++) {
        const log = createMockLog(i + 1, i, BigInt(i * 10), i + 1);
        revisions.push(
          createMockRevision(
            i + 1,
            i,
            BigInt(i * 10),
            1000 + i,
            log,
            mockTimeline,
          ),
        );
      }

      const largeTimeline = createMockTimeline(1, revisions, []);
      service.prefetchTimeline(largeTimeline);

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalledTimes(1);
      const passedStructIds =
        mockWorkbenchClient.readStructYAMLs.calls.mostRecent().args[0];
      expect(passedStructIds.length).toBe(
        WorkbenchPrefetchService.PREFETCH_TIMELINE_LIMIT,
      );

      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalledTimes(
        1,
      );
      const passedLogIds =
        mockWorkbenchClient.getTimelineIdsForLogs.calls.mostRecent().args[0];
      expect(passedLogIds.length).toBe(
        WorkbenchPrefetchService.PREFETCH_TIMELINE_LIMIT,
      );
    });

    it('should not dispatch prefetch calls for an empty timeline', () => {
      const emptyTimeline = createMockTimeline(1, [], []);
      service.prefetchTimeline(emptyTimeline);

      expect(mockWorkbenchClient.readStructYAMLs).not.toHaveBeenCalled();
      expect(mockWorkbenchClient.getTimelineIdsForLogs).not.toHaveBeenCalled();
    });
  });

  describe('prefetchSurroundingRevisions', () => {
    it('should prefetch revisions and logs surrounding the selected revision within bounds', () => {
      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revisions: ReadonlyDomainElement<Revision>[] = [];

      for (let i = 0; i < 50; i++) {
        const log = createMockLog(i + 1, i, BigInt(i * 10), i + 1);
        revisions.push(
          createMockRevision(
            i + 1,
            i,
            BigInt(i * 10),
            500 + i,
            log,
            mockTimeline,
          ),
        );
      }

      const timelineWithManyRevs = createMockTimeline(1, revisions, []);
      revisions.forEach((rev) => {
        (rev as { timeline: ReadonlyDomainElement<Timeline> }).timeline =
          timelineWithManyRevs;
      });

      const selectedRev = revisions[25];
      service.prefetchSurroundingRevisions(selectedRev);

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalledTimes(1);
      const structIds =
        mockWorkbenchClient.readStructYAMLs.calls.mostRecent().args[0];
      expect(structIds.length).toBe(41 * 2); // 41 revisions with 2 struct IDs each

      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalledTimes(
        1,
      );
      const logIds =
        mockWorkbenchClient.getTimelineIdsForLogs.calls.mostRecent().args[0];
      expect(logIds.length).toBe(41);
    });
  });

  describe('prefetchSurroundingLogs', () => {
    it('should prefetch surrounding logs in LogStore when no timeline is selected', () => {
      mockLogStore.count = 50;
      mockFilteredLogIds.set(IdBitset.fromSequential(50));

      const selectedLog = createMockLog(10, 9, 1000n, 300);
      service.prefetchSurroundingLogs(selectedLog);

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalledTimes(1);
      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalledTimes(
        1,
      );

      const passedLogIds =
        mockWorkbenchClient.getTimelineIdsForLogs.calls.mostRecent().args[0];
      expect(passedLogIds).toContain(10);
    });

    it('should prefetch surrounding items on selected timeline when timeline is selected', () => {
      mockLogStore.count = 100;
      mockFilteredLogIds.set(IdBitset.fromSequential(100));

      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revLog = createMockLog(5, 4, 100n, 10);
      const rev = createMockRevision(1, 0, 100n, 50, revLog, mockTimeline);
      const timeline = createMockTimeline(1, [rev], []);

      mockSelectedTimelinesWithChildren.set([timeline]);

      const selectedLog = createMockLog(5, 4, 100n, 10);
      service.prefetchSurroundingLogs(selectedLog);

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalledTimes(1);
      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalledTimes(
        1,
      );
    });
  });

  describe('signal effects', () => {
    it('should trigger prefetchTimeline when selectedTimeline signal updates', async () => {
      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revLog = createMockLog(1, 0, 100n, 10);
      const rev = createMockRevision(1, 0, 100n, 101, revLog, mockTimeline);
      const timelineWithItems = createMockTimeline(1, [rev]);

      mockSelectedTimeline.set(timelineWithItems);
      TestBed.flushEffects();

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalledWith([
        101, 10,
      ]);
      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalledWith([
        1,
      ]);
    });

    it('should trigger prefetchSurroundingRevisions when selectedRevision signal updates', async () => {
      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revLog = createMockLog(1, 0, 100n, 10);
      const rev = createMockRevision(1, 0, 100n, 101, revLog, mockTimeline);
      const timeline = createMockTimeline(1, [rev]);
      (rev as { timeline: ReadonlyDomainElement<Timeline> }).timeline =
        timeline;

      mockSelectedRevision.set(rev);
      TestBed.flushEffects();

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalledWith([
        101, 10,
      ]);
      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalledWith([
        1,
      ]);
    });

    it('should trigger prefetchSurroundingLogs when selectedLog signal updates', async () => {
      mockLogStore.count = 10;
      mockFilteredLogIds.set(IdBitset.fromSequential(10));

      const log = createMockLog(1, 0, 100n, 50);
      mockSelectedLog.set(log);
      TestBed.flushEffects();

      expect(mockWorkbenchClient.readStructYAMLs).toHaveBeenCalled();
      expect(mockWorkbenchClient.getTimelineIdsForLogs).toHaveBeenCalled();
    });
  });
});
