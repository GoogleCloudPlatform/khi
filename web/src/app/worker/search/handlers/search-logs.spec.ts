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
import { handleSearchLogs } from 'src/app/worker/search/handlers/search-logs';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';

describe('handleSearchLogs', () => {
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

  it('should return matched log ids using the provided CEL expression', () => {
    const mockTimelines = [
      {
        id: 0,
        events: [{ log: { id: 10 } }, { log: { id: 11 } }],
        revisions: [{ log: { id: 12 } }],
      } as unknown as Timeline,
      {
        id: 1,
        events: [{ log: { id: 13 } }],
        revisions: [],
      } as unknown as Timeline,
    ];
    spyOn(state.timelineStore!, 'getTimeline').and.callFake((id: number) => {
      return mockTimelines[id] as unknown as Timeline;
    });

    const mockLogs: Record<number, Partial<Log>> = {
      10: {
        severity: { label: 'INFO' } as unknown as Log['severity'],
        summary: 'test 10',
      },
      11: {
        severity: { label: 'ERROR' } as unknown as Log['severity'],
        summary: 'test 11',
      },
      12: {
        severity: { label: 'INFO' } as unknown as Log['severity'],
        summary: 'test 12',
      },
      13: {
        severity: { label: 'ERROR' } as unknown as Log['severity'],
        summary: 'test 13',
      },
    };

    spyOn(state.logStore!, 'getLog').and.callFake((id: number) => {
      return mockLogs[id] as unknown as Log;
    });

    const progressSab = new SharedArrayBuffer(1024);

    // Matching ERROR logs: log 11 and log 13
    const matchedIds = handleSearchLogs(
      {
        type: 'SEARCH_LOGS',
        requestId: 'req-log-1',
        celExpr: 'severity == ERROR',
        workerIndex: 0,
        numWorkers: 1,
        timelineIds: [0, 1],
        progressSab,
      },
      state,
    );

    expect(matchedIds.sort((a, b) => a - b)).toEqual([11, 13]);
  });
});
