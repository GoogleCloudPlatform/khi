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

import { isSharedBuffer } from 'src/app/store/domain/types';
import { SearchWorkerRequest } from 'src/app/worker/search/search-types';
import { SearchWorkerState } from 'src/app/worker/search/search-worker-state';

export function handleSearchLogs(
  request: Extract<SearchWorkerRequest, { type: 'SEARCH_LOGS' }>,
  state: SearchWorkerState,
): void {
  if (!state.timelineStore || !state.logStore) {
    throw new Error('Stores not initialized inside Worker');
  }
  const compileRes = state.logCelEnv.compile(request.celExpr);
  if (!compileRes.success) {
    throw compileRes.error ?? new Error(`Compile failed: ${request.celExpr}`);
  }

  const progressArray = new Int32Array(request.progressSab);
  const workerIndex = request.workerIndex;
  const progressOffset = workerIndex;

  const numWorkers = request.numWorkers;
  const cancellationIndex = numWorkers;
  const isShared = isSharedBuffer(request.progressSab);

  let matchCount = 0;
  const requestView = new Int32Array(request.requestBuf);
  const resultView = new Int32Array(request.resultBuf);
  const requestCount = requestView[0];

  let count = isShared
    ? Atomics.load(progressArray, progressOffset)
    : progressArray[progressOffset];
  const matchedIdsSet = new Set<number>();
  for (let i = 1; i <= requestCount; i++) {
    const tId = requestView[i];
    const isCancelled = isShared
      ? Atomics.load(progressArray, cancellationIndex) !== 0
      : progressArray[cancellationIndex] !== 0;

    if (isCancelled) {
      console.debug(
        `[SearchWorker #${workerIndex}] SEARCH_LOGS aborted (cancelled).`,
      );
      resultView[0] = matchCount;
      return;
    }
    const timeline = state.timelineStore.getTimeline(tId);
    for (const e of timeline.events) {
      const logId = e.log.id;
      if (!matchedIdsSet.has(logId)) {
        const l = state.logStore.getLog(logId);
        if (state.logCelEnv.evaluate(l)) {
          matchedIdsSet.add(logId);
          matchCount++;
          resultView[matchCount] = logId;
        }
      }
      count++;
      if (isShared) {
        Atomics.store(progressArray, progressOffset, count);
      } else {
        progressArray[progressOffset] = count;
      }
    }
    for (const r of timeline.revisions) {
      const logId = r.log.id;
      if (!matchedIdsSet.has(logId)) {
        const l = state.logStore.getLog(logId);
        if (state.logCelEnv.evaluate(l)) {
          matchedIdsSet.add(logId);
          matchCount++;
          resultView[matchCount] = logId;
        }
      }
      count++;
      if (isShared) {
        Atomics.store(progressArray, progressOffset, count);
      } else {
        progressArray[progressOffset] = count;
      }
    }
  }

  console.debug(
    `[SearchWorker #${workerIndex}] SEARCH_LOGS complete. ` +
      `Processed: ${count}, Matched: ${matchedIdsSet.size}`,
  );

  resultView[0] = matchCount;
}
