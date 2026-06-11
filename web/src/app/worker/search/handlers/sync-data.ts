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
  SearchWorkerRequest,
  SearchWorkerResponse,
} from 'src/app/worker/search/search-types';
import { SearchWorkerState } from 'src/app/worker/search/search-worker-state';
import { WorkerStyleStore } from 'src/app/worker/worker-style-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';

export function handleSyncData(
  request: Extract<SearchWorkerRequest, { type: 'SYNC_DATA' }>,
  state: SearchWorkerState,
): void {
  const styleStore = new WorkerStyleStore(request.styleStoreSharedData);
  const internPoolStore = InternPoolStore.fromSharedData(
    request.internPoolSharedData,
  );
  const logStore = LogStore.fromSharedData(
    internPoolStore,
    styleStore,
    request.logStoreSharedData,
  );
  const timelineStore = TimelineStore.fromSharedData(
    internPoolStore,
    styleStore,
    logStore,
    request.timelineStoreSharedData,
  );

  state.styleStore = styleStore;
  state.internPoolStore = internPoolStore;
  state.logStore = logStore;
  state.timelineStore = timelineStore;

  console.debug(`[SearchWorker #${request.workerIndex}] SYNC_DATA complete.`);

  postMessage({
    type: 'SYNC_COMPLETE',
    requestId: request.requestId,
  } as SearchWorkerResponse);
}
