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
import { create, toBinary } from '@bufbuild/protobuf';
import {
  InternedStructSchema,
  InternedValueSchema,
} from 'src/app/generated/khifile/shared_pb';

describe('Timeline', () => {
  let internPool: InternPoolStore;
  let styleStore: StyleStore;
  let logStore: LogStore;
  let timelineStore: TimelineStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = InternPoolStore.create();
    styleStore = new StyleStore();
    logStore = LogStore.create(internPool, styleStore);
    timelineStore = TimelineStore.create(internPool, styleStore, logStore);

    styleStore.addTimelineTypes([
      {
        id: 1,
        label: 'type-a',
        description: 'desc',
        icon: '',
        backgroundColor: mockColor,
        foregroundColor: mockColor,
        typeChipBackgroundColor: mockColor,
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

  describe('id', () => {
    it('should return the correct ID', () => {
      internPool.addStrings([{ id: 1, value: 'timeline-label' }]);
      const rawTimelines: TimelineDTO[] = [
        {
          id: 10,
          timelineTypeId: 1,
          nameStringId: 1,
          parentTimelineId: 0,
          revisionIds: [],
          eventIds: [],
        },
      ];
      timelineStore.initialize(rawTimelines, 1, [], 0, [], 0);
      const timeline = timelineStore.getTimeline(10);
      expect(timeline.id).toBe(10);
    });
  });

  describe('name', () => {
    it('should return the correct string value from the intern pool', () => {
      internPool.addStrings([{ id: 1, value: 'timeline-label' }]);
      const rawTimelines: TimelineDTO[] = [
        {
          id: 10,
          timelineTypeId: 1,
          nameStringId: 1,
          parentTimelineId: 0,
          revisionIds: [],
          eventIds: [],
        },
      ];
      timelineStore.initialize(rawTimelines, 1, [], 0, [], 0);
      const timeline = timelineStore.getTimeline(10);
      expect(timeline.name).toBe('timeline-label');
    });
  });

  describe('revisions', () => {
    it('should yield populated Revision lazy data', () => {
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
      ];

      const rawLogs: LogDTO[] = [
        {
          id: 1,
          ts: 100n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
      ];

      logStore.initialize(rawLogs, 1);
      timelineStore.initialize(rawTimelines, 1, rawRevisions, 1, [], 0);

      const timeline = timelineStore.getTimeline(10);
      const revs = timeline.revisions;
      expect(revs.length).toBe(1);
      expect(revs[0].id).toBe(100);
      expect(revs[0].changedTime).toBe(1234567890n);
      expect(revs[0].legacyChangedTimeMs).toBe(1234.56789);
      expect(revs[0].principal).toBe('user-name');
      expect(revs[0].log.id).toBe(1);
      expect(revs[0].log.timestamp).toBe(100n);
      expect(revs[0].log.legacyTimestampMs).toBe(0.0001);
    });
  });

  describe('events', () => {
    it('should yield populated Event lazy data', () => {
      internPool.addStrings([
        { id: 1, value: 'timeline-label' },
        { id: 3, value: 'log-summary' },
      ]);

      const rawTimelines: TimelineDTO[] = [
        {
          id: 10,
          timelineTypeId: 1,
          nameStringId: 1,
          parentTimelineId: 0,
          revisionIds: [],
          eventIds: [200],
        },
      ];

      const rawEvents: EventDTO[] = [
        {
          id: 200,
          logId: 2,
        },
      ];

      const rawLogs: LogDTO[] = [
        {
          id: 2,
          ts: 200n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
      ];

      logStore.initialize(rawLogs, 1);
      timelineStore.initialize(rawTimelines, 1, [], 0, rawEvents, 1);

      const timeline = timelineStore.getTimeline(10);
      const events = timeline.events;
      expect(events.length).toBe(1);
      expect(events[0].id).toBe(200);
      expect(events[0].log.id).toBe(2);
      expect(events[0].log.timestamp).toBe(200n);
      expect(events[0].log.legacyTimestampMs).toBe(0.0002);
    });
  });

  describe('layer', () => {
    it('should return correct layer depth based on lineage', () => {
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

      expect(parent.layer).toBe(0);
      expect(child.layer).toBe(1);
      expect(grandchild.layer).toBe(2);
    });
  });

  describe('lookupRevisionFromLog and lookupEventFromLog', () => {
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
        {
          id: 1,
          ts: 100n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
        {
          id: 2,
          ts: 200n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
        {
          id: 3,
          ts: 300n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
        {
          id: 4,
          ts: 400n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
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

  describe('lookupRevisionAtNs', () => {
    it('should lookup active revisions at given times', () => {
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
          changedTime: 100n,
          principalStringId: 2,
          verbTypeId: 1,
          stateTypeId: 1,
        },
        {
          id: 101,
          logId: 2,
          changedTime: 200n,
          principalStringId: 2,
          verbTypeId: 1,
          stateTypeId: 1,
        },
      ];

      const rawLogs: LogDTO[] = [
        {
          id: 1,
          ts: 100n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
        {
          id: 2,
          ts: 200n,
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 3,
        },
      ];

      logStore.initialize(rawLogs, 2);
      timelineStore.initialize(rawTimelines, 2, rawRevisions, 2, [], 0);

      const timeline = timelineStore.getTimeline(10);
      const emptyTimeline = timelineStore.getTimeline(20);

      // 1. Before any revision starts
      expect(timeline.lookupRevisionAtNs(50n)).toBeNull();

      // 2. Exactly at first revision start
      expect(timeline.lookupRevisionAtNs(100n)).not.toBeNull();
      expect(timeline.lookupRevisionAtNs(100n)!.id).toBe(100);

      // 3. Exactly at first revision start with exclusive = true
      expect(timeline.lookupRevisionAtNs(100n, true)).toBeNull();

      // 4. Between first and second revision
      expect(timeline.lookupRevisionAtNs(150n)).not.toBeNull();
      expect(timeline.lookupRevisionAtNs(150n)!.id).toBe(100);
      expect(timeline.lookupRevisionAtNs(150n, true)!.id).toBe(100);

      // 5. Exactly at second revision
      expect(timeline.lookupRevisionAtNs(200n)!.id).toBe(101);
      expect(timeline.lookupRevisionAtNs(200n, true)!.id).toBe(100);

      // 6. After second revision
      expect(timeline.lookupRevisionAtNs(250n)!.id).toBe(101);

      // 7. Empty timeline
      expect(emptyTimeline.lookupRevisionAtNs(150n)).toBeNull();
    });
  });

  describe('type', () => {
    it('should return correct type configuration', () => {
      internPool.addStrings([{ id: 1, value: 'parent' }]);

      const rawTimelines: TimelineDTO[] = [
        {
          id: 10,
          timelineTypeId: 1,
          nameStringId: 1,
          parentTimelineId: 0,
          revisionIds: [],
          eventIds: [],
        },
      ];

      timelineStore.initialize(rawTimelines, 1, [], 0, [], 0);

      const parent = timelineStore.getTimeline(10);
      expect(parent.type.id).toBe(1);
    });
  });

  describe('debugPathText', () => {
    it('should return correct slash-separated hierarchical debug path', () => {
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

      timelineStore.initialize(rawTimelines, 2, [], 0, [], 0);

      const child = timelineStore.getTimeline(20);
      expect(child.debugPathText).toBe('parent/child');
    });
  });

  describe('Revision', () => {
    describe('body and bodyYAML', () => {
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
            body: toBinary(InternedStructSchema, struct),
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
          {
            id: 1,
            ts: 100n,
            logTypeId: 1,
            severityTypeId: 1,
            summaryStringId: 3,
          },
          {
            id: 2,
            ts: 200n,
            logTypeId: 1,
            severityTypeId: 1,
            summaryStringId: 3,
          },
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
    });

    describe('properties', () => {
      it('should return correct adjacent revisions, end times, verb, state, and parent timeline', () => {
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
            eventIds: [],
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
            logId: 2,
            changedTime: 200n,
            principalStringId: 2,
            verbTypeId: 1,
            stateTypeId: 1,
          },
        ];

        const rawLogs: LogDTO[] = [
          {
            id: 1,
            ts: 100n,
            logTypeId: 1,
            severityTypeId: 1,
            summaryStringId: 3,
          },
          {
            id: 2,
            ts: 200n,
            logTypeId: 1,
            severityTypeId: 1,
            summaryStringId: 3,
          },
        ];

        logStore.initialize(rawLogs, 2);
        timelineStore.initialize(rawTimelines, 1, rawRevisions, 2, [], 0);

        const timeline = timelineStore.getTimeline(10);
        const revs = timeline.revisions;

        expect(revs.length).toBe(2);

        // Test: timeline reference
        expect(revs[0].timeline.id).toBe(timeline.id);

        // Test: next / prev
        expect(revs[0].prev).toBeNull();
        expect(revs[0].next?.id).toBe(revs[1].id);
        expect(revs[1].prev?.id).toBe(revs[0].id);
        expect(revs[1].next).toBeNull();

        // Test: getEndNs() & legacyGetEndMs()
        expect(revs[0].getEndNs()).toBe(200n);
        expect(revs[0].legacyGetEndMs()).toBe(0.0002);
        expect(revs[1].getEndNs()).toBeNull();
        expect(revs[1].legacyGetEndMs()).toBeNull();

        // Test: verb and state
        expect(revs[0].verb.id).toBe(1);
        expect(revs[0].state.id).toBe(1);
      });
    });
  });

  describe('Event', () => {
    describe('properties', () => {
      it('should return correct parent timeline reference, logIndex, and timestamps', () => {
        internPool.addStrings([
          { id: 1, value: 'timeline-label' },
          { id: 3, value: 'log-summary' },
        ]);

        const rawTimelines: TimelineDTO[] = [
          {
            id: 10,
            timelineTypeId: 1,
            nameStringId: 1,
            parentTimelineId: 0,
            revisionIds: [],
            eventIds: [200],
          },
        ];

        const rawEvents: EventDTO[] = [
          {
            id: 200,
            logId: 1,
          },
        ];

        const rawLogs: LogDTO[] = [
          {
            id: 1,
            ts: 100n,
            logTypeId: 1,
            severityTypeId: 1,
            summaryStringId: 3,
          },
        ];

        logStore.initialize(rawLogs, 1);
        timelineStore.initialize(rawTimelines, 1, [], 0, rawEvents, 1);

        const timeline = timelineStore.getTimeline(10);
        const ev = timeline.events[0];

        expect(ev.timeline.id).toBe(timeline.id);
        expect(ev.logIndex).toBe(0);
        expect(ev.timestamp).toBe(100n);
        expect(ev.legacyTimestamp).toBe(0.0001);
      });
    });
  });
});
