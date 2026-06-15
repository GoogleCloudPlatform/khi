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
import { SearchWorkerManager } from 'src/app/services/search-worker-manager.service';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { TimelineDTO } from 'src/app/store/domain/timeline-store';
import { LogDTO } from 'src/app/store/domain/log-store';

describe('SearchWorkerManager', () => {
  let manager: SearchWorkerManager;
  let styleStore: StyleStore;
  let internPoolStore: InternPoolStore;
  let logStore: LogStore;
  let timelineStore: TimelineStore;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    manager = TestBed.inject(SearchWorkerManager);

    styleStore = new StyleStore();
    styleStore.addSeverities([
      {
        id: 1,
        label: 'INFO',
        shortLabel: 'I',
        backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 1 },
        order: 1,
      },
    ]);
    styleStore.addLogTypes([
      {
        id: 1,
        label: 'k8s',
        description: '',
        backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 1 },
      },
    ]);
    styleStore.addTimelineTypes([
      {
        id: 1,
        label: 'Pod',
        description: '',
        icon: '',
        backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
        foregroundColor: { r: 0, g: 0, b: 0, a: 1 },
        typeChipBackgroundColor: { r: 0, g: 0, b: 0, a: 1 },
        typeChipForegroundColor: { r: 0, g: 0, b: 0, a: 1 },
        visible: true,
        sortPriority: 1,
        height: 20,
      },
    ]);

    internPoolStore = InternPoolStore.create();
    internPoolStore.addStrings([
      { id: 1, value: 'pod-a' },
      { id: 2, value: 'pod-b' },
    ]);

    logStore = LogStore.create(internPoolStore, styleStore);
    const logs: LogDTO[] = [
      {
        id: 10,
        ts: 1000n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 1,
      },
      {
        id: 20,
        ts: 2000n,
        logTypeId: 1,
        severityTypeId: 1,
        summaryStringId: 2,
      },
    ];
    logStore.initialize(logs, 2);

    timelineStore = TimelineStore.create(internPoolStore, styleStore, logStore);
    const timelines: TimelineDTO[] = [
      {
        id: 100,
        timelineTypeId: 1,
        nameStringId: 1,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [10],
      },
      {
        id: 200,
        timelineTypeId: 1,
        nameStringId: 2,
        parentTimelineId: 0,
        revisionIds: [],
        eventIds: [20],
      },
    ];
    timelineStore.initialize(
      timelines,
      2,
      [],
      0,
      [
        { id: 10, logId: 10 },
        { id: 20, logId: 20 },
      ],
      2,
    );
  });

  afterEach(() => {
    manager.ngOnDestroy();
  });

  it('should sync stores and run parallel searches correctly', async () => {
    await manager.syncData(
      internPoolStore,
      logStore,
      timelineStore,
      styleStore,
    );

    // Timeline search: name == 'pod-a' should match timeline 100
    const timelineProgressCalls: { current: number; total: number }[] = [];
    const matchedTimelines = await manager.searchTimelines(
      "name == 'pod-a'",
      (current, total) => {
        timelineProgressCalls.push({ current, total });
      },
    );
    expect(matchedTimelines.has(100)).toBeTrue();
    expect(matchedTimelines.has(200)).toBeFalse();
    expect(timelineProgressCalls.length).toBeGreaterThan(0);
    expect(timelineProgressCalls[timelineProgressCalls.length - 1]).toEqual({
      current: 2,
      total: 2,
    });

    // Log search: summary == 'pod-b' should match log 20
    const logProgressCalls: { current: number; total: number }[] = [];
    const matchedLogs = await manager.searchLogs(
      "summary == 'pod-b'",
      [100, 200],
      (current, total) => {
        logProgressCalls.push({ current, total });
      },
    );
    expect(matchedLogs.has(20)).toBeTrue();
    expect(matchedLogs.has(10)).toBeFalse();
    expect(logProgressCalls.length).toBeGreaterThan(0);
    expect(logProgressCalls[logProgressCalls.length - 1]).toEqual({
      current: 2,
      total: 2,
    });
  });
});
