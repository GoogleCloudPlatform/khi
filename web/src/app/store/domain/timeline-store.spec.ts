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

import {
  TimelineStore,
  TimelineDTO,
  RevisionDTO,
  EventDTO,
} from 'src/app/store/domain/timeline-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore, LogDTO } from 'src/app/store/domain/log-store';

describe('TimelineStore', () => {
  let internPool: InternPoolStore;
  let styleStore: StyleStore;
  let logStore: LogStore;
  let store: TimelineStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = InternPoolStore.create();
    styleStore = new StyleStore();
    logStore = LogStore.create(internPool, styleStore);
    store = TimelineStore.create(internPool, styleStore, logStore);

    styleStore.addTimelineTypes([
      {
        id: 1,
        label: 'type-a',
        description: 'desc',
        icon: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        typeChipBackgroundColor: mockColor,
        typeChipForegroundColor: mockColor,
        visible: true,
        sortPriority: 0,
        height: 1,
      },
    ]);

    styleStore.addSeverities([
      {
        id: 1,
        label: 'S1',
        shortLabel: 'S1',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 0,
      },
    ]);

    styleStore.addLogTypes([
      {
        id: 1,
        label: 'L1',
        description: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
      },
    ]);

    styleStore.addVerbs([
      {
        id: 1,
        label: 'V1',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        visible: true,
      },
    ]);

    styleStore.addRevisionStates([
      {
        id: 1,
        label: 'normal',
        icon: '',
        description: '',
        backgroundColor: mockColor,
        style: 0,
      },
    ]);
  });

  it('should successfully populate internal states on initialize', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-x' },
      { id: 2, value: 'principal-y' },
    ]);

    const logs = [
      { id: 1, ts: 10n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
    ];
    logStore.initialize(logs, 1);

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [100],
        eventIds: [200],
      },
    ];

    const rawRevisions: RevisionDTO[] = [
      {
        id: 100,
        logId: 1,
        changedTime: 123456n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
        resourceBodyStructId: 88,
      },
    ];

    const rawEvents: EventDTO[] = [
      {
        id: 200,
        logId: 1,
      },
    ];

    expect(() =>
      store.initialize(rawTimelines, 1, rawRevisions, 1, rawEvents, 1),
    ).not.toThrow();

    const t = store.getTimeline(10);
    expect(t.id).toBe(10);
    expect(t.name).toBe('timeline-x');

    const rev = t.revisions[0];
    expect(rev.id).toBe(100);
    expect(rev.structId).toBe(88);

    const all = store.timelines;
    expect(all.length).toBe(1);
    expect(all[0].id).toBe(10);
  });

  it('should correctly build severity index for timelines', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-x' },
      { id: 2, value: 'principal-y' },
    ]);

    // Severity 1: Info, Severity 2: Warning
    styleStore.addSeverities([
      {
        id: 1,
        label: 'Info',
        shortLabel: 'I',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 1,
      },
      {
        id: 2,
        label: 'Warning',
        shortLabel: 'W',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 2,
      },
    ]);

    const logs = [
      { id: 1, ts: 10n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 20n, logTypeId: 1, severityTypeId: 2, summaryStringId: 1 },
    ];
    logStore.initialize(logs, 2);

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [100],
        eventIds: [],
      },
      {
        id: 20,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [200],
      },
      {
        id: 30,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [],
      },
    ];

    const rawRevisions: RevisionDTO[] = [
      {
        id: 100,
        logId: 1,
        changedTime: 10n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
      },
    ];

    const rawEvents: EventDTO[] = [
      {
        id: 200,
        logId: 2,
      },
    ];

    store.initialize(rawTimelines, 3, rawRevisions, 1, rawEvents, 1);

    const s1 = styleStore.getSeverity(1);
    const s2 = styleStore.getSeverity(2);

    const t1 = store.getTimeline(10);
    expect(t1.hasSeverity(s1)).toBeTrue();
    expect(t1.hasSeverity(s2)).toBeFalse();

    const t2 = store.getTimeline(20);
    expect(t2.hasSeverity(s1)).toBeFalse();
    expect(t2.hasSeverity(s2)).toBeTrue();

    const t3 = store.getTimeline(30);
    expect(t3.hasSeverity(s1)).toBeFalse();
    expect(t3.hasSeverity(s2)).toBeFalse();

    // Invalid severity ID
    expect(
      t1.hasSeverity({
        id: 99,
        label: 'invalid',
        shortLabel: 'inv',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 99,
      }),
    ).toBeFalse();
  });

  it('should correctly check if timeline has severities', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-x' },
      { id: 2, value: 'principal-y' },
    ]);

    // Severity 1: Info (order 1), Severity 2: Warning (order 2), Severity 3: Error (order 3)
    styleStore.addSeverities([
      {
        id: 1,
        label: 'Info',
        shortLabel: 'I',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 1,
      },
      {
        id: 2,
        label: 'Warning',
        shortLabel: 'W',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 2,
      },
      {
        id: 3,
        label: 'Error',
        shortLabel: 'E',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        order: 3,
      },
    ]);

    const logs = [
      { id: 1, ts: 10n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 20n, logTypeId: 1, severityTypeId: 2, summaryStringId: 1 },
      { id: 3, ts: 30n, logTypeId: 1, severityTypeId: 3, summaryStringId: 1 },
    ];
    logStore.initialize(logs, 3);

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [100],
        eventIds: [],
      },
      {
        id: 20,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [200],
      },
      {
        id: 30,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [300],
      },
    ];

    const rawRevisions: RevisionDTO[] = [
      {
        id: 100,
        logId: 1,
        changedTime: 10n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
      },
    ];

    const rawEvents: EventDTO[] = [
      {
        id: 200,
        logId: 2,
      },
      {
        id: 300,
        logId: 3,
      },
    ];

    store.initialize(rawTimelines, 3, rawRevisions, 1, rawEvents, 2);

    const s1 = styleStore.getSeverity(1);
    const s2 = styleStore.getSeverity(2);
    const s3 = styleStore.getSeverity(3);

    const t1 = store.getTimeline(10);
    const t2 = store.getTimeline(20);
    const t3 = store.getTimeline(30);

    // Single severity checks
    expect(t1.hasSeverity(s1)).toBeTrue();
    expect(t1.hasSeverity(s2)).toBeFalse();

    // Multiple severity checks
    expect(t1.hasSeverity(s1, s2)).toBeTrue();
    expect(t2.hasSeverity(s2, s3)).toBeTrue();
    expect(t3.hasSeverity(s1, s2)).toBeFalse();
    expect(t3.hasSeverity(s3)).toBeTrue();

    // Empty severities list
    expect(t1.hasSeverity()).toBeFalse();
  });

  it('should correctly decode timeline traversal path', () => {
    internPool.addStrings([
      { id: 1, value: 'root' },
      { id: 2, value: 'child' },
    ]);

    const rawTimelines: TimelineDTO[] = [
      {
        id: 1,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 2,
        timelineTypeId: 1,
        nameStringId: 2,
        parentTimelineId: 1,
        revisionIds: [],
        eventIds: [],
      },
    ];

    store.initialize(rawTimelines, 2, [], 0, [], 0);

    const timeline = store.getTimeline(2);
    const computedPath = timeline.path;

    expect(computedPath.length).toBe(2);
    expect(computedPath[0].label).toBe('root');
    expect(computedPath[1].label).toBe('child');
  });

  it('should error when reading invalid timeline ID', () => {
    expect(() => store.getTimeline(999)).toThrowError(
      'Timeline ID 999 not found',
    );
  });

  it('should successfully map child timeline IDs on initialize', () => {
    internPool.addStrings([
      { id: 1, value: 'parent' },
      { id: 2, value: 'child' },
    ]);

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 20,
        timelineTypeId: 1,
        nameStringId: 2,
        parentTimelineId: 10,
        revisionIds: [],
        eventIds: [],
      },
    ];

    store.initialize(rawTimelines, 2, [], 0, [], 0);

    const childIds = store._getChildIdsForTimeline(10);
    expect(childIds.length).toBe(1);
    expect(childIds[0]).toBe(20);

    expect(store._getChildIdsForTimeline(20).length).toBe(0);
    expect(() => store._getChildIdsForTimeline(999)).toThrowError(
      'Timeline ID 999 not found',
    );
  });

  it('should restore from shared memory using fromSharedData and enforce readOnly guard', () => {
    internPool.addStrings([{ id: 1, value: 'timeline-1' }]);

    const timelines: TimelineDTO[] = [
      {
        id: 1,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [1],
        eventIds: [],
      },
    ];

    const revisions: RevisionDTO[] = [
      {
        id: 1,
        logId: 1,
        changedTime: 1000n,
        principalStringId: 1,
        verbTypeId: 1,
        stateTypeId: 1,
        resourceBodyStructId: 99,
      },
    ];

    // Mock logs initialization needed by severity indexing
    const logs: LogDTO[] = [
      { id: 1, ts: 1000n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
    ];
    logStore.initialize(logs, 1);

    store.initialize(timelines, 1, revisions, 1, [], 0);

    const sharedData = store.getSharedData();
    const restoredStore = TimelineStore.fromSharedData(
      internPool,
      styleStore,
      logStore,
      sharedData,
    );

    expect(restoredStore.timelines.length).toBe(1);
    const restoredTimeline = restoredStore.getTimeline(1);
    expect(restoredTimeline.id).toBe(1);

    const restoredRevisions = restoredTimeline.revisions;
    expect(restoredRevisions.length).toBe(1);
    expect(restoredRevisions[0].id).toBe(1);
    expect(restoredRevisions[0].structId).toBe(99);

    expect(() => {
      restoredStore.initialize(timelines, 1, revisions, 1, [], 0);
    }).toThrowError('Cannot write to a shared read-only TimelineStore');
  });

  describe('ArrayBuffer fallback when SharedArrayBuffer is unsupported', () => {
    let originalSharedArrayBuffer: typeof SharedArrayBuffer | undefined;

    beforeEach(() => {
      originalSharedArrayBuffer = SharedArrayBuffer;
      (
        globalThis as unknown as Record<
          string,
          typeof SharedArrayBuffer | undefined
        >
      )['SharedArrayBuffer'] = undefined;
    });

    afterEach(() => {
      (
        globalThis as unknown as Record<
          string,
          typeof SharedArrayBuffer | undefined
        >
      )['SharedArrayBuffer'] = originalSharedArrayBuffer;
    });

    it('should allocate ArrayBuffer instead of SharedArrayBuffer and perform operations successfully', () => {
      const fallbackStore = TimelineStore.create(
        internPool,
        styleStore,
        logStore,
      );
      const timelines: TimelineDTO[] = [
        {
          id: 1,
          timelineTypeId: 1,
          nameStringId: 1,
          parentTimelineId: 0,
          revisionIds: [],
          eventIds: [],
        },
      ];
      fallbackStore.initialize(timelines, 1, [], 0, [], 0);

      expect(fallbackStore.timelines.length).toBe(1);
      const timeline = fallbackStore.getTimeline(1);
      expect(timeline.id).toBe(1);

      const sharedData = fallbackStore.getSharedData();
      expect(sharedData.metadataSab instanceof ArrayBuffer).toBeTrue();

      const restoredStore = TimelineStore.fromSharedData(
        internPool,
        styleStore,
        logStore,
        sharedData,
      );
      expect(restoredStore.timelines.length).toBe(1);
      expect(restoredStore.getTimeline(1).id).toBe(1);
    });
  });
});
