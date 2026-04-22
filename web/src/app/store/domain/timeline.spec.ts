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
import { create } from '@bufbuild/protobuf';
import {
  InternedStructSchema,
  InternedValueSchema,
} from 'src/app/generated/khifile/shared_pb';

describe('Timeline lazy adapter models', () => {
  let internPool: InternPoolStore;
  let styleStore: StyleStore;
  let logStore: LogStore;
  let timelineStore: TimelineStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = new InternPoolStore();
    styleStore = new StyleStore();
    logStore = new LogStore(internPool, styleStore);
    timelineStore = new TimelineStore(internPool, styleStore, logStore);

    styleStore.addTimelineTypes([
      {
        id: 1,
        label: 'type-a',
        description: 'desc',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        visible: true,
        sortPriority: 0,
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

  it('should yield populated Revision and Event lazy data', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-label' },
      { id: 2, value: 'user-name' },
      { id: 3, value: 'log-summary' },
    ]);

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
        changedTime: 1234567890n,
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

    const rawLogs: LogDTO[] = [
      { id: 1, ts: 100n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
      { id: 2, ts: 200n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
    ];

    logStore.initialize(rawLogs, 2);
    timelineStore.initialize(rawTimelines, 1, rawRevisions, 1, rawEvents, 1);

    const timeline = timelineStore.getTimeline(10);
    expect(timeline.id).toBe(10);
    expect(timeline.name).toBe('timeline-label');

    const revs = timeline.revisions;
    expect(revs.length).toBe(1);
    expect(revs[0].id).toBe(100);
    expect(revs[0].changedTime).toBe(1234567890n);
    expect(revs[0].legacyChangedTimeMs).toBe(1234.56789);
    expect(revs[0].principal).toBe('user-name');
    expect(revs[0].log.id).toBe(1);
    expect(revs[0].log.timestamp).toBe(100n);
    expect(revs[0].log.legacyTimestampMs).toBe(0.0001);

    const events = timeline.events;
    expect(events.length).toBe(1);
    expect(events[0].id).toBe(200);
    expect(events[0].log.id).toBe(2);
    expect(events[0].log.timestamp).toBe(200n);
    expect(events[0].log.legacyTimestampMs).toBe(0.0002);
  });

  it('should traverse parent and child relationships via generator and accessors', () => {
    internPool.addStrings([
      { id: 1, value: 'parent-tl' },
      { id: 2, value: 'child-tl' },
      { id: 3, value: 'grandchild-tl' },
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
      {
        id: 30,
        timelineTypeId: 1,
        nameStringId: 3,
        parentTimelineId: 20,
        revisionIds: [],
        eventIds: [],
      },
    ];

    timelineStore.initialize(rawTimelines, 3, [], 0, [], 0);

    const parent = timelineStore.getTimeline(10);
    const child = timelineStore.getTimeline(20);
    const grandchild = timelineStore.getTimeline(30);

    expect(parent.parent()).toBeNull();
    expect(child.parent()?.id).toBe(10);
    expect(grandchild.parent()?.id).toBe(20);

    expect(parent.layer).toBe(0);
    expect(child.layer).toBe(1);
    expect(grandchild.layer).toBe(2);

    expect(parent.childrenCount).toBe(1);
    expect(child.childrenCount).toBe(1);
    expect(grandchild.childrenCount).toBe(0);

    const childrenIter = parent.children();
    const firstChild = childrenIter.next().value;
    expect(firstChild?.id).toBe(20);
    expect(childrenIter.next().done).toBe(true);
  });

  it('should decode revision body and bodyYAML correctly', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-label' },
      { id: 2, value: 'user-name' },
      { id: 3, value: 'log-summary' },
      { id: 10, value: 'user' },
      { id: 11, value: 'status' },
      { id: 12, value: 'alice' },
    ]);

    internPool.addFieldPathSets([{ id: 1, fieldPathStringIds: [10, 11] }]);

    const struct = create(InternedStructSchema, {
      fieldPathSetId: 1,
      values: [
        create(InternedValueSchema, {
          kind: { case: 'stringValue', value: 12 },
        }),
        create(InternedValueSchema, {
          kind: { case: 'int64Value', value: 42n },
        }),
      ],
    });

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [100, 101],
        eventIds: [],
      },
    ];

    const rawRevisions: RevisionDTO[] = [
      {
        id: 100,
        logId: 1,
        changedTime: 1234567890n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
        body: struct,
      },
      {
        id: 101,
        logId: 2,
        changedTime: 1234567890n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
      },
    ];

    const rawLogs: LogDTO[] = [
      { id: 1, ts: 100n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
      { id: 2, ts: 200n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
    ];

    logStore.initialize(rawLogs, 2);
    timelineStore.initialize(rawTimelines, 1, rawRevisions, 2, [], 0);

    const timeline = timelineStore.getTimeline(10);
    const revs = timeline.revisions;

    expect(revs[0].body).toEqual({
      user: 'alice',
      status: 42,
    });
    expect(revs[0].bodyYAML).toContain('user: alice');
    expect(revs[0].bodyYAML).toContain('status: 42');

    expect(revs[1].body).toBeNull();
    expect(revs[1].bodyYAML).toBe('');
  });

  it('should return correct logIndex, timestamp, and legacyTimestamp for Event and Revision', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-label' },
      { id: 2, value: 'user-name' },
      { id: 3, value: 'log-summary' },
    ]);

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
        changedTime: 1234567890n,
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

    const rawLogs: LogDTO[] = [
      { id: 1, ts: 100n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
      { id: 2, ts: 200n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
    ];

    logStore.initialize(rawLogs, 2);
    timelineStore.initialize(rawTimelines, 1, rawRevisions, 1, rawEvents, 1);

    const timeline = timelineStore.getTimeline(10);

    expect(timeline.revisions[0].logIndex).toBe(0);

    expect(timeline.events[0].logIndex).toBe(1);
    expect(timeline.events[0].timestamp).toBe(200n);
    expect(timeline.events[0].legacyTimestamp).toBe(0.0002);
  });

  it('should traverse descendants recursively using the descendants generator', () => {
    internPool.addStrings([
      { id: 1, value: 'parent-tl' },
      { id: 2, value: 'child-tl' },
      { id: 3, value: 'grandchild-tl' },
      { id: 4, value: 'sibling-tl' },
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
      {
        id: 30,
        timelineTypeId: 1,
        nameStringId: 3,
        parentTimelineId: 20,
        revisionIds: [],
        eventIds: [],
      },
      {
        id: 40,
        timelineTypeId: 1,
        nameStringId: 4,
        parentTimelineId: 10,
        revisionIds: [],
        eventIds: [],
      },
    ];

    timelineStore.initialize(rawTimelines, 4, [], 0, [], 0);

    const parent = timelineStore.getTimeline(10);
    const descendantIds = Array.from(parent.descendants()).map((t) => t.id);

    expect(descendantIds).toEqual([20, 30, 40]);
  });

  it('should retrieve revision pairs correctly by logIndex', () => {
    internPool.addStrings([
      { id: 1, value: 'parent-tl' },
      { id: 2, value: 'user' },
    ]);

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [100, 101],
        eventIds: [],
      },
      {
        id: 20,
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
        changedTime: 1234567890n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
      },
      {
        id: 101,
        logId: 2,
        changedTime: 1234567890n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
      },
    ];

    const rawLogs: LogDTO[] = [
      { id: 1, ts: 100n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 200n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
    ];

    logStore.initialize(rawLogs, 2);
    timelineStore.initialize(rawTimelines, 2, rawRevisions, 2, [], 0);

    const timeline = timelineStore.getTimeline(10);

    // For logIndex = 0 (revision 100) -> first revision: { previous: null, current: revision 100 }
    const pair0 = timeline.getRevisionPairByLogId(0);
    expect(pair0).not.toBeNull();
    expect(pair0!.previous).toBeNull();
    expect(pair0!.current.id).toBe(100);

    // For logIndex = 1 (revision 101) -> { previous: revision 100, current: revision 101 }
    const pair1 = timeline.getRevisionPairByLogId(1);
    expect(pair1).not.toBeNull();
    expect(pair1!.previous?.id).toBe(100);
    expect(pair1!.current.id).toBe(101);

    // For a logIndex that does not correspond to any revision
    spyOn(console, 'warn');
    const pairNotFound = timeline.getRevisionPairByLogId(99);
    expect(pairNotFound).toBeNull();
    expect(console.warn).toHaveBeenCalled();

    // For timeline with no revisions
    const emptyTimeline = timelineStore.getTimeline(20);
    expect(emptyTimeline.getRevisionPairByLogId(0)).toBeNull();
  });

  it('should lookup events and revisions from logIndex using binary search', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-label' },
      { id: 2, value: 'user-name' },
      { id: 3, value: 'log-summary' },
    ]);

    const rawTimelines: TimelineDTO[] = [
      {
        id: 10,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [100, 101],
        eventIds: [200, 201],
      },
    ];

    const rawRevisions: RevisionDTO[] = [
      {
        id: 100,
        logId: 1,
        changedTime: 100n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
      },
      {
        id: 101,
        logId: 3,
        changedTime: 300n,
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
        id: 201,
        logId: 4,
      },
    ];

    const rawLogs: LogDTO[] = [
      { id: 1, ts: 100n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
      { id: 2, ts: 200n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
      { id: 3, ts: 300n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
      { id: 4, ts: 400n, logTypeId: 1, severityTypeId: 1, summaryStringId: 3 },
    ];

    logStore.initialize(rawLogs, 4);
    timelineStore.initialize(rawTimelines, 1, rawRevisions, 2, rawEvents, 2);

    const timeline = timelineStore.getTimeline(10);

    const log1 = logStore.getLog(1);
    const log2 = logStore.getLog(2);
    const log3 = logStore.getLog(3);
    const log4 = logStore.getLog(4);

    // Test lookups for revisions
    expect(timeline.lookupRevisionFromLog(log1)).not.toBeNull();
    expect(timeline.lookupRevisionFromLog(log1)!.id).toBe(100);
    expect(timeline.lookupRevisionFromLog(log3)).not.toBeNull();
    expect(timeline.lookupRevisionFromLog(log3)!.id).toBe(101);
    expect(timeline.lookupRevisionFromLog(log2)).toBeNull();
    expect(timeline.lookupRevisionFromLog(null)).toBeNull();

    // Test lookups for events
    expect(timeline.lookupEventFromLog(log2)).not.toBeNull();
    expect(timeline.lookupEventFromLog(log2)!.id).toBe(200);
    expect(timeline.lookupEventFromLog(log4)).not.toBeNull();
    expect(timeline.lookupEventFromLog(log4)!.id).toBe(201);
    expect(timeline.lookupEventFromLog(log1)).toBeNull();
    expect(timeline.lookupEventFromLog(null)).toBeNull();
  });
});
