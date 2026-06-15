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

import { SearchWorkerRequest } from 'src/app/worker/search/search-types';
import { SearchWorkerState } from 'src/app/worker/search/search-worker-state';

export function handleSearchTimelines(
  request: Extract<SearchWorkerRequest, { type: 'SEARCH_TIMELINES' }>,
  state: SearchWorkerState,
): number[] {
  if (!state.timelineStore) {
    throw new Error('TimelineStore not initialized inside Worker');
  }
  const compileRes = state.timelineCelEnv.compile(request.celExpr);
  if (!compileRes.success) {
    throw compileRes.error ?? new Error(`Compile failed: ${request.celExpr}`);
  }

  const matchedIds: number[] = [];
  let processedCount = 0;
  const timelines = state.timelineStore.timelines;
  const totalTimelines = timelines.length;
  const workerIndex = request.workerIndex;
  const numWorkers = request.numWorkers;
  const progressArray = new Int32Array(request.progressSab);
  const progressOffset = workerIndex * 16;

  const cancellationIndex = numWorkers * 16;

  for (let i = workerIndex; i < totalTimelines; i += numWorkers) {
    if (Atomics.load(progressArray, cancellationIndex) !== 0) {
      console.debug(
        `[SearchWorker #${workerIndex}] SEARCH_TIMELINES aborted (cancelled).`,
      );
      return matchedIds;
    }
    const t = timelines[i];
    if (state.timelineCelEnv.evaluate(t, state.timelineStore)) {
      matchedIds.push(t.id);
    }
    processedCount++;
    progressArray[progressOffset] = processedCount;
  }

  console.debug(
    `[SearchWorker #${workerIndex}] SEARCH_TIMELINES complete. ` +
      `Processed: ${processedCount}, Matched: ${matchedIds.length}`,
  );

  return matchedIds;
}
