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
import { LogStore } from 'src/app/store/domain/log-store';

describe('TimelineStore', () => {
  let internPool: InternPoolStore;
  let styleStore: StyleStore;
  let logStore: LogStore;
  let store: TimelineStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = InternPoolStore.create();
    styleStore = new StyleStore();
    logStore = LogStore.create(internPool, styleStore, 0);
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
    logStore = LogStore.create(internPool, styleStore, 1);
    logStore.addLogs(logs);
    logStore.shrinkToFit();

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

    store = TimelineStore.create(internPool, styleStore, logStore, 1, 1, 1);
    store.addRevisions(rawRevisions);
    store.addEvents(rawEvents);
    store.addTimelines(rawTimelines);
    store.shrinkToFit();

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

    store = TimelineStore.create(internPool, styleStore, logStore, 2);
    store.addTimelines(rawTimelines);
    store.shrinkToFit();

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

    store = TimelineStore.create(internPool, styleStore, logStore, 2);
    store.addTimelines(rawTimelines);
    store.shrinkToFit();

    const childIds = store._getChildIdsForTimeline(10);
    expect(childIds.length).toBe(1);
    expect(childIds[0]).toBe(20);

    expect(store._getChildIdsForTimeline(20).length).toBe(0);
    expect(() => store._getChildIdsForTimeline(999)).toThrowError(
      'Timeline ID 999 not found',
    );
  });

  describe('create and dynamic methods', () => {
    it('should dynamically add revisions, events, and timelines and shrink to fit', () => {
      const dynamicStore = TimelineStore.create(
        internPool,
        styleStore,
        logStore,
        1,
        1,
        1,
      );
      expect(dynamicStore.count).toBe(0);
      expect(dynamicStore.revisionCount).toBe(0);
      expect(dynamicStore.eventCount).toBe(0);

      dynamicStore.addRevision({
        id: 101,
        logId: 1,
        changedTime: 500n,
        principalStringId: 1,
        verbTypeId: 1,
        stateTypeId: 1,
        resourceBodyStructId: 10,
        fieldAnnotations: [],
      });

      dynamicStore.addEvent({
        id: 201,
        logId: 2,
      });

      dynamicStore.addTimeline({
        id: 1,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [101],
        eventIds: [201],
      });

      dynamicStore.addTimeline({
        id: 2,
        timelineTypeId: 1,
        nameStringId: 2,
        parentTimelineId: 1,
        revisionIds: [],
        eventIds: [],
      });

      expect(dynamicStore.count).toBe(2);
      expect(dynamicStore.revisionCount).toBe(1);
      expect(dynamicStore.eventCount).toBe(1);

      // Verify lazy relationship rebuild even without calling shrinkToFit
      const childIdsBeforeShrink = dynamicStore._getChildIdsForTimeline(1);
      expect(childIdsBeforeShrink).toEqual([2]);

      dynamicStore.shrinkToFit();

      expect(dynamicStore.count).toBe(2);
      expect(dynamicStore.revisionCount).toBe(1);
      expect(dynamicStore.eventCount).toBe(1);

      const t1 = dynamicStore.getTimeline(1);
      expect(t1.id).toBe(1);
      const t2 = dynamicStore.getTimeline(2);
      expect(t2.id).toBe(2);

      const childIds = dynamicStore._getChildIdsForTimeline(1);
      expect(childIds).toEqual([2]);
    });

    it('should handle shrinkToFit on an empty store with capacity 0', () => {
      const emptyStore = TimelineStore.create(
        internPool,
        styleStore,
        logStore,
        0,
        0,
        0,
      );
      emptyStore.shrinkToFit();
      expect(emptyStore.count).toBe(0);
      expect(emptyStore.revisionCount).toBe(0);
      expect(emptyStore.eventCount).toBe(0);
    });

    it('should handle multiple buffer expansions across all views from capacity 1 and preserve domain relationships', () => {
      const localLogStore = LogStore.create(internPool, styleStore, 15);
      for (let i = 1; i <= 15; i++) {
        localLogStore.addLog({
          id: i,
          ts: BigInt(i * 100),
          logTypeId: 1,
          severityTypeId: 1,
          summaryStringId: 1,
        });
      }

      const store = TimelineStore.create(
        internPool,
        styleStore,
        localLogStore,
        1,
        1,
        1,
      );
      for (let i = 1; i <= 15; i++) {
        store.addRevision({
          id: i,
          logId: i,
          changedTime: BigInt(i * 100),
          principalStringId: 1,
          verbTypeId: 1,
          stateTypeId: 1,
          resourceBodyStructId: i * 10,
        });
        store.addEvent({ id: i, logId: i });
        store.addTimeline({
          id: i,
          timelineTypeId: 1,
          nameStringId: 1,
          parentTimelineId: i > 1 ? 1 : 0,
          revisionIds: [i],
          eventIds: [i],
        });
      }

      expect(store.count).toBe(15);
      expect(store.revisionCount).toBe(15);
      expect(store.eventCount).toBe(15);

      store.shrinkToFit();

      for (let i = 1; i <= 15; i++) {
        const t = store.getTimeline(i);
        expect(t.id).toBe(i);
        expect(t.revisions.length).toBe(1);
        expect(t.revisions[0].id).toBe(i);
        expect(t.revisions[0].structId).toBe(i * 10);
        expect(t.revisions[0].log.timestamp).toBe(BigInt(i * 100));
        expect(t.events.length).toBe(1);
        expect(t.events[0].id).toBe(i);
      }

      const rootChildren = store._getChildIdsForTimeline(1);
      expect(rootChildren.length).toBe(14);
    });

    it('should correctly build and query reverse log indexes', () => {
      internPool.addStrings([
        { id: 1, value: 'timeline-1' },
        { id: 2, value: 'timeline-2' },
        { id: 3, value: 'summary' },
      ]);

      logStore.addLogs([
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
      ]);

      store.addRevision({
        id: 10,
        logId: 1,
        changedTime: 100n,
        principalStringId: 1,
        verbTypeId: 1,
        stateTypeId: 1,
      });
      store.addEvent({ id: 20, logId: 2 });
      store.addEvent({ id: 21, logId: 1 }); // Log 1 is shared in timeline 2 as an event

      store.addTimeline({
        id: 100,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [10],
        eventIds: [],
      });

      store.addTimeline({
        id: 200,
        timelineTypeId: 1,
        nameStringId: 2,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [20, 21],
      });

      store.shrinkToFit();

      // Query log 1 (appears in timeline 100 as revision 10 and timeline 200 as event 21)
      expect(store.getTimelineIdsForLogId(1)).toEqual([100, 200]);
      const timelinesForLog1 = store.getTimelinesForLogId(1);
      expect(timelinesForLog1.length).toBe(2);
      expect(timelinesForLog1[0].id).toBe(100);
      expect(timelinesForLog1[1].id).toBe(200);

      const log1 = logStore.getLog(1);
      expect(timelinesForLog1[0].lookupRevisionFromLog(log1)?.id).toBe(10);
      expect(timelinesForLog1[1].lookupEventFromLog(log1)?.id).toBe(21);

      // Query log 2 (appears only in timeline 200 as event 20)
      expect(store.getTimelineIdsForLogId(2)).toEqual([200]);
      const timelinesForLog2 = store.getTimelinesForLogId(2);
      expect(timelinesForLog2.length).toBe(1);
      const log2 = logStore.getLog(2);
      expect(timelinesForLog2[0].lookupEventFromLog(log2)?.id).toBe(20);

      // Query log 3 (not in any timeline)
      expect(store.getTimelineIdsForLogId(3)).toEqual([]);
      expect(store.getTimelinesForLogId(3)).toEqual([]);
    });
  });
});
