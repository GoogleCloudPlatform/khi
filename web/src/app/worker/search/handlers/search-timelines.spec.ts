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

    const progressSab = new SharedArrayBuffer(1024);

    // Test the logic directly
    // name == 'T1' will match id=1 and id=3
    const matchedIds = handleSearchTimelines(
      {
        type: 'SEARCH_TIMELINES',
        requestId: 'req-1',
        celExpr: "name == 'T1'",
        workerIndex: 0,
        numWorkers: 1,
        progressSab,
      },
      state,
    );

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

    const progressSab = new SharedArrayBuffer(1024);

    const worker0Matches = handleSearchTimelines(
      {
        type: 'SEARCH_TIMELINES',
        requestId: 'req-1',
        celExpr: "name == 'T1'",
        workerIndex: 0,
        numWorkers: 2, // Workers 0 and 1
        progressSab,
      },
      state,
    );
    // Worker 0 processes index 0, 2, 4
    // 0: match, 2: not match, 4: match
    expect(worker0Matches).toEqual([0, 4]);

    const worker1Matches = handleSearchTimelines(
      {
        type: 'SEARCH_TIMELINES',
        requestId: 'req-2',
        celExpr: "name == 'T1'",
        workerIndex: 1,
        numWorkers: 2,
        progressSab,
      },
      state,
    );
    // Worker 1 processes index 1, 3
    // 1: match, 3: match
    expect(worker1Matches).toEqual([1, 3]);
  });
});
