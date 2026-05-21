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
    internPool = new InternPoolStore();
    styleStore = new StyleStore();
    logStore = new LogStore(internPool, styleStore);
    store = new TimelineStore(internPool, styleStore, logStore);

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
});
