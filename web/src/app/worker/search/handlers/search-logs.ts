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

export function handleSearchLogs(
  request: Extract<SearchWorkerRequest, { type: 'SEARCH_LOGS' }>,
  state: SearchWorkerState,
): number[] {
  if (!state.timelineStore || !state.logStore) {
    throw new Error('Stores not initialized inside Worker');
  }
  const compileRes = state.logCelEnv.compile(request.celExpr);
  if (!compileRes.success) {
    throw compileRes.error ?? new Error(`Compile failed: ${request.celExpr}`);
  }

  const progressArray = new Int32Array(request.progressSab);
  const workerIndex = request.workerIndex;
  const progressOffset = workerIndex * 16;

  const numWorkers = request.numWorkers;
  const cancellationIndex = numWorkers * 16;

  let count = 0;
  const matchedIdsSet = new Set<number>();
  for (const tId of request.timelineIds) {
    if (Atomics.load(progressArray, cancellationIndex) !== 0) {
      console.debug(
        `[SearchWorker #${workerIndex}] SEARCH_LOGS aborted (cancelled).`,
      );
      return Array.from(matchedIdsSet);
    }
    const timeline = state.timelineStore.getTimeline(tId);
    for (const e of timeline.events) {
      const logId = e.log.id;
      if (!matchedIdsSet.has(logId)) {
        const l = state.logStore.getLog(logId);
        if (state.logCelEnv.evaluate(l)) {
          matchedIdsSet.add(logId);
        }
      }
      count++;
      progressArray[progressOffset] = count;
    }
    for (const r of timeline.revisions) {
      const logId = r.log.id;
      if (!matchedIdsSet.has(logId)) {
        const l = state.logStore.getLog(logId);
        if (state.logCelEnv.evaluate(l)) {
          matchedIdsSet.add(logId);
        }
      }
      count++;
      progressArray[progressOffset] = count;
    }
  }

  console.debug(
    `[SearchWorker #${workerIndex}] SEARCH_LOGS complete. ` +
      `Processed: ${count}, Matched: ${matchedIdsSet.size}`,
  );

  return Array.from(matchedIdsSet);
}
