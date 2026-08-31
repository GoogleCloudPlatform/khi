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
import { signal, WritableSignal } from '@angular/core';
import { StructYamlPrefetchService } from 'src/app/services/struct-yaml-prefetch.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { Timeline, Revision, Event } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';

describe('StructYamlPrefetchService', () => {
  let service: StructYamlPrefetchService;
  let mockWorkbenchClient: jasmine.SpyObj<WorkbenchClientService>;
  let mockSelectedTimeline: WritableSignal<ReadonlyDomainElement<Timeline> | null>;
  let mockSelectedRevision: WritableSignal<ReadonlyDomainElement<Revision> | null>;
  let mockSelectedLog: WritableSignal<ReadonlyDomainElement<Log> | null>;
  let mockSelectedTimelinesWithChildren: WritableSignal<
    ReadonlyDomainElement<Timeline>[]
  >;
  let mockFilteredLogIds: WritableSignal<IdBitset>;
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
    mockWorkbenchClient = jasmine.createSpyObj('WorkbenchClientService', [
      'prefetchStructYAMLs',
    ]);
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
        StructYamlPrefetchService,
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

    service = TestBed.inject(StructYamlPrefetchService);
  });

  describe('prefetchTimeline', () => {
    it('should prefetch struct IDs from revisions and events sorted by timestamp up to limit', () => {
      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revLog1 = createMockLog(1, 0, 100n, 10);
      const revLog2 = createMockLog(2, 1, 300n, 20);
      const eventLog1 = createMockLog(3, 2, 200n, 30);

      const rev1 = createMockRevision(1, 0, 100n, 1000, revLog1, mockTimeline);
      const rev2 = createMockRevision(2, 1, 300n, 2000, revLog2, mockTimeline);
      const event1 = createMockEvent(1, 200n, eventLog1);

      const timelineWithItems = createMockTimeline(1, [rev1, rev2], [event1]);

      service.prefetchTimeline(timelineWithItems);

      // Chronological order:
      // 100n -> rev1 (structId 1000, log structId 10)
      // 200n -> event1 (log structId 30)
      // 300n -> rev2 (structId 2000, log structId 20)
      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledWith([
        1000, 10, 30, 2000, 20,
      ]);
    });

    it('should cap prefetched timeline struct IDs at PREFETCH_TIMELINE_LIMIT', () => {
      const mockTimeline = { id: 2 } as ReadonlyDomainElement<Timeline>;
      const revisions: ReadonlyDomainElement<Revision>[] = [];

      for (let i = 0; i < 60; i++) {
        const log = createMockLog(i + 1, i, BigInt(i * 10), 0);
        revisions.push(
          createMockRevision(
            i + 1,
            i,
            BigInt(i * 10),
            100 + i,
            log,
            mockTimeline,
          ),
        );
      }

      const largeTimeline = createMockTimeline(2, revisions, []);
      service.prefetchTimeline(largeTimeline);

      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledTimes(1);
      const callArgs =
        mockWorkbenchClient.prefetchStructYAMLs.calls.mostRecent()
          .args[0] as number[];
      expect(callArgs.length).toBe(
        StructYamlPrefetchService.PREFETCH_TIMELINE_LIMIT,
      );
      expect(callArgs[0]).toBe(100);
      expect(callArgs[49]).toBe(149);
    });

    it('should do nothing if timeline has no positive struct IDs', () => {
      const mockTimeline = { id: 3 } as ReadonlyDomainElement<Timeline>;
      const log = createMockLog(1, 0, 100n, 0);
      const rev = createMockRevision(1, 0, 100n, 0, log, mockTimeline);
      const emptyTimeline = createMockTimeline(3, [rev], []);

      service.prefetchTimeline(emptyTimeline);

      expect(mockWorkbenchClient.prefetchStructYAMLs).not.toHaveBeenCalled();
    });
  });

  describe('prefetchSurroundingRevisions', () => {
    it('should prefetch revisions surrounding the selected revision within bounds', () => {
      const mockTimeline = {
        id: 10,
        revisions: [] as ReadonlyDomainElement<Revision>[],
      } as unknown as ReadonlyDomainElement<Timeline>;

      for (let i = 0; i < 50; i++) {
        const log = createMockLog(i + 1, i, BigInt(i * 10), 500 + i);
        const rev = createMockRevision(
          i + 1,
          i,
          BigInt(i * 10),
          1000 + i,
          log,
          mockTimeline,
        );
        (mockTimeline.revisions as ReadonlyDomainElement<Revision>[]).push(rev);
      }

      // Select revision at index 25
      const selectedRev = mockTimeline.revisions[25];
      service.prefetchSurroundingRevisions(selectedRev);

      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledTimes(1);
      const callArgs =
        mockWorkbenchClient.prefetchStructYAMLs.calls.mostRecent()
          .args[0] as number[];

      // Index 25 - 20 = index 5, index 25 + 20 = index 45. (41 revisions * 2 struct IDs = 82 IDs)
      expect(callArgs).toContain(1005);
      expect(callArgs).toContain(1025);
      expect(callArgs).toContain(1045);
      expect(callArgs).not.toContain(1004);
      expect(callArgs).not.toContain(1046);
    });

    it('should handle revision at beginning of timeline (index 0)', () => {
      const mockTimeline = {
        id: 11,
        revisions: [] as ReadonlyDomainElement<Revision>[],
      } as unknown as ReadonlyDomainElement<Timeline>;

      for (let i = 0; i < 30; i++) {
        const log = createMockLog(i + 1, i, BigInt(i * 10), 0);
        const rev = createMockRevision(
          i + 1,
          i,
          BigInt(i * 10),
          100 + i,
          log,
          mockTimeline,
        );
        (mockTimeline.revisions as ReadonlyDomainElement<Revision>[]).push(rev);
      }

      const firstRev = mockTimeline.revisions[0];
      service.prefetchSurroundingRevisions(firstRev);

      const callArgs =
        mockWorkbenchClient.prefetchStructYAMLs.calls.mostRecent()
          .args[0] as number[];
      // Should include indices 0 to 20
      expect(callArgs.length).toBe(21);
      expect(callArgs[0]).toBe(100);
      expect(callArgs[20]).toBe(120);
    });
  });

  describe('prefetchSurroundingLogs', () => {
    it('should prefetch surrounding logs in LogStore when no timeline is selected', () => {
      mockLogStore.count = 50;
      mockFilteredLogIds.set(IdBitset.fromSequential(50));

      // Select log at index 25 (id: 26, structId: 225)
      const selectedLog = createMockLog(26, 25, 250n, 225);
      service.prefetchSurroundingLogs(selectedLog);

      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledTimes(1);
      const callArgs =
        mockWorkbenchClient.prefetchStructYAMLs.calls.mostRecent()
          .args[0] as number[];

      // Index 25 - 20 = 5 (structId 205) to index 45 (structId 245)
      expect(callArgs).toContain(205);
      expect(callArgs).toContain(225);
      expect(callArgs).toContain(245);
      expect(callArgs).not.toContain(204);
      expect(callArgs).not.toContain(246);
    });

    it('should prefetch surrounding items on selected timeline when timeline is selected', () => {
      const mockTimeline = { id: 1 } as ReadonlyDomainElement<Timeline>;
      const revisions: ReadonlyDomainElement<Revision>[] = [];
      for (let i = 0; i < 30; i++) {
        const log = createMockLog(i + 1, i, BigInt(i * 10), 0);
        revisions.push(
          createMockRevision(
            i + 1,
            i,
            BigInt(i * 10),
            300 + i,
            log,
            mockTimeline,
          ),
        );
      }
      const timeline = createMockTimeline(1, revisions);
      mockSelectedTimelinesWithChildren.set([timeline]);
      mockFilteredLogIds.set(IdBitset.fromSequential(30));

      // Select log at index 15
      const selectedLog = createMockLog(16, 15, 150n, 0);
      service.prefetchSurroundingLogs(selectedLog);

      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledTimes(1);
      const callArgs =
        mockWorkbenchClient.prefetchStructYAMLs.calls.mostRecent()
          .args[0] as number[];

      // Surrounding revisions around index 15 (radius 20): from index 0 to 29 (structIds 300 to 329)
      expect(callArgs).toContain(300);
      expect(callArgs).toContain(315);
      expect(callArgs).toContain(329);
    });
  });

  describe('reactive selection effects', () => {
    it('should trigger prefetchTimeline when selectedTimeline signal updates', async () => {
      const revLog = createMockLog(1, 0, 100n, 10);
      const rev = createMockRevision(1, 0, 100n, 100, revLog, {
        id: 1,
      } as ReadonlyDomainElement<Timeline>);
      const timeline = createMockTimeline(1, [rev]);

      mockSelectedTimeline.set(timeline);
      TestBed.flushEffects();

      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledWith([
        100, 10,
      ]);
    });

    it('should trigger prefetchSurroundingRevisions when selectedRevision signal updates', async () => {
      const mockTimeline = {
        id: 5,
        revisions: [] as ReadonlyDomainElement<Revision>[],
      } as unknown as ReadonlyDomainElement<Timeline>;
      const revLog = createMockLog(1, 0, 100n, 0);
      const rev = createMockRevision(1, 0, 100n, 500, revLog, mockTimeline);
      (mockTimeline.revisions as ReadonlyDomainElement<Revision>[]).push(rev);

      mockSelectedRevision.set(rev);
      TestBed.flushEffects();

      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledWith([
        500,
      ]);
    });

    it('should trigger prefetchSurroundingLogs when selectedLog signal updates', async () => {
      const log = createMockLog(1, 0, 100n, 888);
      mockLogStore.count = 1;
      mockLogStore.getLogIdByIndex = () => 1;
      mockLogStore.getBodyStructId = () => 888;
      mockFilteredLogIds.set(IdBitset.fromAll([log.id]));

      mockSelectedLog.set(log);
      TestBed.flushEffects();

      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledWith([
        888,
      ]);
    });

    it('should re-trigger prefetch when selection resets to null then re-selects the same timeline', async () => {
      const revLog = createMockLog(1, 0, 100n, 10);
      const rev = createMockRevision(1, 0, 100n, 100, revLog, {
        id: 1,
      } as ReadonlyDomainElement<Timeline>);
      const timeline = createMockTimeline(1, [rev]);

      mockSelectedTimeline.set(timeline);
      TestBed.flushEffects();
      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledTimes(1);

      mockSelectedTimeline.set(null);
      TestBed.flushEffects();

      mockSelectedTimeline.set(timeline);
      TestBed.flushEffects();
      expect(mockWorkbenchClient.prefetchStructYAMLs).toHaveBeenCalledTimes(2);
    });
  });
});
