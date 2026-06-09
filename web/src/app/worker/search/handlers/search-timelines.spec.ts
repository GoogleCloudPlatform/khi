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

import { SearchWorkerState } from 'src/app/worker/search/search-worker-state';
import { handleSearchTimelines } from 'src/app/worker/search/handlers/search-timelines';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { Timeline } from 'src/app/store/domain/timeline';

describe('handleSearchTimelines', () => {
  let state: SearchWorkerState;

  beforeEach(() => {
    state = new SearchWorkerState();
    const internPool = InternPoolStore.create();
    const styleStore = new StyleStore();
    state.styleStore = styleStore;
    state.internPoolStore = internPool;
    state.logStore = LogStore.create(internPool, styleStore);
    state.timelineStore = TimelineStore.create(
      internPool,
      styleStore,
      state.logStore,
    );
  });

  it('should return matched timeline ids using the provided CEL expression', () => {
    // We will bypass Timeline creation complexity by forcing the timeline data in the store
    const mockTimelines = [
      { id: 1, name: 'T1' } as unknown as Timeline,
      { id: 2, name: 'T2' } as unknown as Timeline,
      { id: 3, name: 'T1' } as unknown as Timeline,
    ];
    Object.defineProperty(state.timelineStore, 'timelines', {
      get: () => mockTimelines,
    });
    spyOn(state.timelineStore!, 'getTimeline').and.callFake((id: number) => {
      const found = mockTimelines.find((t) => t.id === id);
      if (!found) {
        throw new Error(`Timeline ${id} not found`);
      }
      return found;
    });

    const progressSab = new SharedArrayBuffer(1024);
    const requestBuf = new SharedArrayBuffer(1024);
    const resultBuf = new SharedArrayBuffer(1024);

    const reqView = new Int32Array(requestBuf);
    reqView[0] = 3;
    reqView[1] = 1;
    reqView[2] = 2;
    reqView[3] = 3;

    // Test the logic directly
    // name == 'T1' will match id=1 and id=3
    handleSearchTimelines(
      {
        type: 'SEARCH_TIMELINES',
        requestId: 'req-1',
        celExpr: "name == 'T1'",
        workerIndex: 0,
        numWorkers: 1,
        requestBuf,
        resultBuf,
        progressSab,
      },
      state,
    );

    const resView = new Int32Array(resultBuf);
    const matchedCount = resView[0];
    const matchedIds = [];
    for (let i = 1; i <= matchedCount; i++) {
      matchedIds.push(resView[i]);
    }

    expect(matchedIds).toEqual([1, 3]);
  });

  it('should process partitions correctly based on workerIndex and numWorkers', () => {
    const mockTimelines = [
      { id: 0, name: 'T1' } as unknown as Timeline,
      { id: 1, name: 'T1' } as unknown as Timeline,
      { id: 2, name: 'T2' } as unknown as Timeline, // Does not match
      { id: 3, name: 'T1' } as unknown as Timeline,
      { id: 4, name: 'T1' } as unknown as Timeline,
    ];
    Object.defineProperty(state.timelineStore, 'timelines', {
      get: () => mockTimelines,
    });
    spyOn(state.timelineStore!, 'getTimeline').and.callFake((id: number) => {
      const found = mockTimelines.find((t) => t.id === id);
      if (!found) {
        throw new Error(`Timeline ${id} not found`);
      }
      return found;
    });

    const progressSab = new SharedArrayBuffer(1024);

    const requestBuf0 = new SharedArrayBuffer(1024);
    const resultBuf0 = new SharedArrayBuffer(1024);
    const reqView0 = new Int32Array(requestBuf0);
    reqView0[0] = 3;
    reqView0[1] = 0;
    reqView0[2] = 2;
    reqView0[3] = 4;

    handleSearchTimelines(
      {
        type: 'SEARCH_TIMELINES',
        requestId: 'req-1',
        celExpr: "name == 'T1'",
        workerIndex: 0,
        numWorkers: 2, // Workers 0 and 1
        requestBuf: requestBuf0,
        resultBuf: resultBuf0,
        progressSab,
      },
      state,
    );
    // Worker 0 processes index 0, 2, 4
    // 0: match, 2: not match, 4: match
    const resView0 = new Int32Array(resultBuf0);
    const worker0Matches = [];
    for (let i = 1; i <= resView0[0]; i++) worker0Matches.push(resView0[i]);
    expect(worker0Matches).toEqual([0, 4]);

    const requestBuf1 = new SharedArrayBuffer(1024);
    const resultBuf1 = new SharedArrayBuffer(1024);
    const reqView1 = new Int32Array(requestBuf1);
    reqView1[0] = 2;
    reqView1[1] = 1;
    reqView1[2] = 3;

    handleSearchTimelines(
      {
        type: 'SEARCH_TIMELINES',
        requestId: 'req-2',
        celExpr: "name == 'T1'",
        workerIndex: 1,
        numWorkers: 2,
        requestBuf: requestBuf1,
        resultBuf: resultBuf1,
        progressSab,
      },
      state,
    );
    // Worker 1 processes index 1, 3
    // 1: match, 3: match
    const resView1 = new Int32Array(resultBuf1);
    const worker1Matches = [];
    for (let i = 1; i <= resView1[0]; i++) worker1Matches.push(resView1[i]);
    expect(worker1Matches).toEqual([1, 3]);
  });
});
