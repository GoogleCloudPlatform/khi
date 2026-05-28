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
import { create, toBinary } from '@bufbuild/protobuf';
import {
  InternedStructSchema,
  InternedValueSchema,
} from 'src/app/generated/khifile/shared_pb';

describe('TimelineStore', () => {
  let internPool: InternPoolStore;
  let styleStore: StyleStore;
  let logStore: LogStore;
  let store: TimelineStore;

  const mockColor = { r: 0, g: 0, b: 0, a: 1 };

  beforeEach(() => {
    internPool = new InternPoolStore();
    styleStore = new StyleStore();
    logStore = new LogStore(internPool, styleStore);
    store = new TimelineStore(internPool, styleStore, logStore);

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

  it('should successfully populate internal states on initialize', () => {
    internPool.addStrings([
      { id: 1, value: 'timeline-x' },
      { id: 2, value: 'principal-y' },
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
        changedTime: 123456n,
        principalStringId: 2,
        verbTypeId: 1,
        stateTypeId: 1,
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

  it('should return decoded revision body correctly', () => {
    internPool.addStrings([
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
        changedTime: 10n,
        principalStringId: 1,
        verbTypeId: 1,
        stateTypeId: 1,
        body: toBinary(InternedStructSchema, struct),
      },
      {
        id: 101,
        logId: 2,
        changedTime: 20n,
        principalStringId: 1,
        verbTypeId: 1,
        stateTypeId: 1,
      },
    ];

    const logs = [
      { id: 1, ts: 10n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
      { id: 2, ts: 20n, logTypeId: 1, severityTypeId: 1, summaryStringId: 1 },
    ];
    logStore.initialize(logs, 2);

    store.initialize(rawTimelines, 1, rawRevisions, 2, [], 0);

    const t = store.getTimeline(10);
    expect(t.revisions[0].body).toEqual({
      user: 'alice',
      status: 42,
    });
    expect(t.revisions[0].bodyYAML).toBe('user: alice\nstatus: 42\n');
    expect(t.revisions[1].body).toBeNull();
    expect(t.revisions[1].bodyYAML).toBe('');
  });
});
