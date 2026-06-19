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

import { Injectable, OnDestroy } from '@angular/core';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import {
  SearchWorkerRequest,
  SearchWorkerResponse,
} from 'src/app/worker/search/search-types';
import { SearchWorkerState } from 'src/app/worker/search/search-worker-state';
import { handleSearchTimelines } from 'src/app/worker/search/handlers/search-timelines';
import { handleSearchLogs } from 'src/app/worker/search/handlers/search-logs';
import { isSharedArrayBufferSupported } from '../store/domain/types';
import { CancellationError } from 'src/app/store/domain/filter/types';

interface SearchSession {
  readonly type: 'SEARCH_TIMELINES' | 'SEARCH_LOGS';
  readonly celExpr: string;
  readonly timelineIds: readonly number[];
  nextIndex: number;
  readonly matchedIds: Set<number>;
  readonly onProgress?: (current: number, total: number) => void;
  readonly progressSab: SharedArrayBuffer | ArrayBuffer;
  activeWorkersCount: number;
  resolve(matchedIds: Set<number>): void;
  reject(error: Error): void;
}

const MAX_REQUEST_ELEMENTS = 10_000; // 40KB
const MAX_RESULT_ELEMENTS = 1_000_000; // 4MB
const MAX_TIMELINES_PER_FRAME_ON_MAIN_THREAD = 10000;
const MAX_LOGS_PER_FRAME_ON_MAIN_THREAD = 10000;
const MAX_PROCESSING_TIME_MS_ON_MAIN_THREAD = 100;

/**
 * Orchestrates a pool of WebWorkers to run CEL log and timeline searches concurrently.
 */
@Injectable({ providedIn: 'root' })
export class SearchWorkerManager implements OnDestroy {
  private readonly workers: Worker[] = [];
  private readonly numWorkers: number;
  private readonly activeSessions = new Map<string, SearchSession>();
  private sessionCounter = 0;
  private activeTimelineStore: TimelineStore | null = null;
  private activeTimelineSearchRequestId: string | null = null;
  private activeLogSearchRequestId: string | null = null;

  private readonly useWorker = isSharedArrayBufferSupported();
  private readonly mainThreadState = new SearchWorkerState();

  private readonly requestBufs: (SharedArrayBuffer | ArrayBuffer)[] = [];
  private readonly resultBufs: (SharedArrayBuffer | ArrayBuffer)[] = [];

  constructor() {
    const cores = Math.ceil((navigator.hardwareConcurrency || 4) / 2);
    this.numWorkers = this.useWorker ? Math.max(1, cores - 1) : 1;
    for (let i = 0; i < this.numWorkers; i++) {
      if (this.useWorker) {
        this.requestBufs.push(
          new SharedArrayBuffer((MAX_REQUEST_ELEMENTS + 1) * 4),
        );
        this.resultBufs.push(
          new SharedArrayBuffer((MAX_RESULT_ELEMENTS + 1) * 4),
        );
      } else {
        this.requestBufs.push(new ArrayBuffer((MAX_REQUEST_ELEMENTS + 1) * 4));
        this.resultBufs.push(new ArrayBuffer((MAX_RESULT_ELEMENTS + 1) * 4));
      }
    }

    if (this.useWorker) {
      for (let i = 0; i < this.numWorkers; i++) {
        const worker = new Worker(
          new URL('../worker/search/search.worker', import.meta.url),
          { type: 'module' },
        );
        worker.addEventListener(
          'message',
          (event: MessageEvent<SearchWorkerResponse>) => {
            this.handleWorkerMessage(event.data);
          },
        );
        this.workers.push(worker);
      }
    }
  }

  /**
   * Synchronizes shared store data arrays across all active WebWorkers.
   */
  public syncData(
    internPoolStore: InternPoolStore,
    logStore: LogStore,
    timelineStore: TimelineStore,
    styleStore: StyleStore,
  ): Promise<void[]> {
    this.activeTimelineStore = timelineStore;

    if (!this.useWorker) {
      this.mainThreadState.internPoolStore = internPoolStore;
      this.mainThreadState.logStore = logStore;
      this.mainThreadState.timelineStore = timelineStore;
      this.mainThreadState.styleStore = styleStore;
      return Promise.resolve([]);
    }

    const internPoolSharedData = internPoolStore.getSharedData();
    const logStoreSharedData = logStore.getSharedData();
    const timelineStoreSharedData = timelineStore.getSharedData();
    const styleStoreSharedData = styleStore.getSharedData();

    const promises = this.workers.map((worker, index) => {
      return new Promise<void>((resolve, reject) => {
        const tempSessionId = `sync-${this.sessionCounter++}`;
        const handleSync = (event: MessageEvent<SearchWorkerResponse>) => {
          const res = event.data;
          if (res.type === 'SYNC_COMPLETE' && res.requestId === tempSessionId) {
            worker.removeEventListener('message', handleSync);
            resolve();
          } else if (res.type === 'ERROR' && res.requestId === tempSessionId) {
            worker.removeEventListener('message', handleSync);
            reject(new Error(res.error));
          }
        };
        worker.addEventListener('message', handleSync);

        worker.postMessage({
          type: 'SYNC_DATA',
          requestId: tempSessionId,
          workerIndex: index,
          internPoolSharedData,
          logStoreSharedData,
          timelineStoreSharedData,
          styleStoreSharedData,
        } as SearchWorkerRequest);
      });
    });

    return Promise.all(promises);
  }

  /**
   * Searches timeline IDs matching the CEL expression.
   *
   * Search is performed on Web Workers on the worker mode.
   * Yields control to the main thread periodically if running in non-worker mode
   * to avoid blocking the main UI thread during intensive log processing.
   * @param celExpr The CEL expression to evaluate.
   * @param onProgress Callback function to report progress.
   * @returns A promise resolving to a Set of matched timeline IDs.
   */
  public async searchTimelines(
    celExpr: string,
    onProgress?: (current: number, total: number) => void,
  ): Promise<Set<number>> {
    const allTimelineIds =
      this.activeTimelineStore?.timelines.map((t) => t.id) ?? [];
    if (celExpr === '') {
      return new Set(allTimelineIds);
    }

    // Cancels current ongoing timeline search task if exists.
    if (this.activeTimelineSearchRequestId) {
      this.cancelSearch(this.activeTimelineSearchRequestId);
    }

    const totalTimelines = allTimelineIds.length;
    const startTime = performance.now();
    const requestId = `search-t-${this.sessionCounter++}`;
    this.activeTimelineSearchRequestId = requestId;

    if (!this.useWorker) {
      return this.searchTimelinesMainThread(
        requestId,
        celExpr,
        allTimelineIds,
        totalTimelines,
        startTime,
        onProgress,
      );
    }

    const progressSab = new SharedArrayBuffer((this.workers.length + 1) * 4);
    const progressArray = new Int32Array(progressSab);

    let intervalId: ReturnType<typeof setInterval> | null = null;
    if (onProgress) {
      intervalId = setInterval(() => {
        let current = 0;
        for (let i = 0; i < this.workers.length; i++) {
          current += progressArray[i];
        }
        onProgress(current, totalTimelines);
      }, 50);
    }

    const promise = new Promise<Set<number>>((resolve, reject) => {
      this.activeSessions.set(requestId, {
        type: 'SEARCH_TIMELINES',
        celExpr,
        timelineIds: allTimelineIds,
        nextIndex: 0,
        matchedIds: new Set<number>(),
        activeWorkersCount: 0,
        onProgress,
        progressSab,
        resolve,
        reject,
      });
    });

    const cleanup = () => {
      if (intervalId !== null) clearInterval(intervalId);
    };

    for (let i = 0; i < this.workers.length; i++) {
      this.dispatchNextChunk(requestId, i);
    }

    try {
      const matchedSet = await promise;
      if (onProgress) onProgress(totalTimelines, totalTimelines);
      const elapsedMs = performance.now() - startTime;
      const speed = elapsedMs > 0 ? totalTimelines / (elapsedMs / 1000) : 0;
      console.debug(
        `[SearchWorkerManager] searchTimelines finished:\n` +
          `  Query: "${celExpr}"\n` +
          `  Elapsed Time: ${elapsedMs.toFixed(2)}ms\n` +
          `  Search Speed: ${speed.toFixed(0)} timelines/sec`,
      );
      return matchedSet;
    } finally {
      if (this.activeTimelineSearchRequestId === requestId) {
        this.activeTimelineSearchRequestId = null;
      }
      cleanup();
    }
  }

  /**
   * Searches log event IDs matching the CEL expression across specified timelines.
   *
   * Yields control to the main thread periodically if running in non-worker mode
   * to avoid blocking the main UI thread during intensive log processing.
   * @param celExpr The CEL expression to evaluate.
   * @param timelineIds The list of timeline IDs to search within.
   * @param onProgress Callback function to report progress.
   * @returns A promise resolving to a Set of matched log event IDs.
   */
  public async searchLogs(
    celExpr: string,
    timelineIds: readonly number[],
    onProgress?: (current: number, total: number) => void,
  ): Promise<Set<number>> {
    if (timelineIds.length === 0 || celExpr === '') {
      return new Set();
    }

    let totalLogs = 0;
    for (const tId of timelineIds) {
      const timeline = this.activeTimelineStore?.getTimeline(tId);
      if (timeline) {
        totalLogs += timeline.events.length + timeline.revisions.length;
      }
    }

    if (this.activeLogSearchRequestId) {
      this.cancelSearch(this.activeLogSearchRequestId);
    }

    const startTime = performance.now();
    const requestId = `search-l-${this.sessionCounter++}`;
    this.activeLogSearchRequestId = requestId;

    if (!this.useWorker) {
      return this.searchLogsMainThread(
        requestId,
        celExpr,
        timelineIds,
        totalLogs,
        startTime,
        onProgress,
      );
    }

    const progressSab = new SharedArrayBuffer((this.workers.length + 1) * 4);
    const progressArray = new Int32Array(progressSab);

    let intervalId: ReturnType<typeof setInterval> | null = null;
    if (onProgress) {
      intervalId = setInterval(() => {
        let current = 0;
        for (let i = 0; i < this.workers.length; i++) {
          current += progressArray[i];
        }
        onProgress(current, totalLogs);
      }, 50);
    }

    const promise = new Promise<Set<number>>((resolve, reject) => {
      this.activeSessions.set(requestId, {
        type: 'SEARCH_LOGS',
        celExpr,
        timelineIds,
        nextIndex: 0,
        matchedIds: new Set<number>(),
        activeWorkersCount: 0,
        onProgress,
        progressSab,
        resolve,
        reject,
      });
    });

    const cleanup = () => {
      if (intervalId !== null) clearInterval(intervalId);
    };

    for (let i = 0; i < this.workers.length; i++) {
      this.dispatchNextChunk(requestId, i);
    }

    try {
      const matchedSet = await promise;
      if (onProgress) onProgress(totalLogs, totalLogs);
      const elapsedMs = performance.now() - startTime;
      const speed = elapsedMs > 0 ? totalLogs / (elapsedMs / 1000) : 0;
      console.debug(
        `[SearchWorkerManager] searchLogs finished:\n` +
          `  Query: "${celExpr}"\n` +
          `  Elapsed Time: ${elapsedMs.toFixed(2)}ms\n` +
          `  Search Speed: ${speed.toFixed(0)} logs/sec`,
      );
      return matchedSet;
    } finally {
      if (this.activeLogSearchRequestId === requestId) {
        this.activeLogSearchRequestId = null;
      }
      cleanup();
    }
  }

  private handleWorkerMessage(response: SearchWorkerResponse): void {
    if (response.type === 'SYNC_COMPLETE') return;

    if (!response.requestId) return;

    const session = this.activeSessions.get(response.requestId);
    if (!session) return;

    if (response.type === 'ERROR') {
      session.reject(new Error(response.error));
      this.activeSessions.delete(response.requestId);
      return;
    }

    if (response.type === 'SEARCH_COMPLETE') {
      const workerIndex = response.workerIndex;
      session.activeWorkersCount--;

      const resultView = new Int32Array(this.resultBufs[workerIndex]);
      const matchCount = resultView[0];
      for (let i = 1; i <= matchCount; i++) {
        session.matchedIds.add(resultView[i]);
      }

      this.dispatchNextChunk(response.requestId, workerIndex);
    }
  }

  private dispatchNextChunk(sessionId: string, workerIndex: number): void {
    const session = this.activeSessions.get(sessionId);
    if (!session) return;

    if (session.nextIndex >= session.timelineIds.length) {
      if (session.activeWorkersCount === 0) {
        session.resolve(session.matchedIds);
        this.activeSessions.delete(sessionId);
        if (this.activeTimelineSearchRequestId === sessionId)
          this.activeTimelineSearchRequestId = null;
        if (this.activeLogSearchRequestId === sessionId)
          this.activeLogSearchRequestId = null;
      }
      return;
    }

    const remainingTimelines = session.timelineIds.length - session.nextIndex;
    const availableWorkers = this.workers.length - session.activeWorkersCount;

    session.activeWorkersCount++;

    const requestView = new Int32Array(this.requestBufs[workerIndex]);
    let count = 0;

    switch (session.type) {
      case 'SEARCH_TIMELINES': {
        const targetChunkSize =
          availableWorkers > 0
            ? Math.ceil(remainingTimelines / availableWorkers)
            : remainingTimelines;
        const limit = Math.min(MAX_REQUEST_ELEMENTS, targetChunkSize);
        while (
          session.nextIndex < session.timelineIds.length &&
          count < limit
        ) {
          requestView[++count] = session.timelineIds[session.nextIndex++];
        }
        break;
      }
      case 'SEARCH_LOGS': {
        const targetTimelinesPerWorker =
          availableWorkers > 0
            ? Math.ceil(remainingTimelines / availableWorkers)
            : remainingTimelines;
        const limitTimelines = Math.min(
          MAX_REQUEST_ELEMENTS,
          targetTimelinesPerWorker,
        );

        let logsInChunk = 0;
        while (
          session.nextIndex < session.timelineIds.length &&
          count < limitTimelines
        ) {
          const tId = session.timelineIds[session.nextIndex];
          const t = this.activeTimelineStore?.getTimeline(tId);
          const tLogs = t ? t.events.length + t.revisions.length : 0;

          if (count > 0 && logsInChunk + tLogs > MAX_RESULT_ELEMENTS) {
            break;
          }

          requestView[++count] = tId;
          logsInChunk += tLogs;
          session.nextIndex++;
        }
        break;
      }
    }

    requestView[0] = count;

    if (this.useWorker) {
      this.workers[workerIndex].postMessage({
        type: session.type,
        requestId: sessionId,
        workerIndex,
        numWorkers: this.workers.length,
        celExpr: session.celExpr,
        requestBuf: this.requestBufs[workerIndex],
        resultBuf: this.resultBufs[workerIndex],
        progressSab: session.progressSab,
      } as SearchWorkerRequest);
    }
  }

  private cancelSearch(requestId: string): void {
    const session = this.activeSessions.get(requestId);
    if (!session) {
      return;
    }
    const progressArray = new Int32Array(session.progressSab);
    const cancellationIndex = this.numWorkers;
    if (this.useWorker) {
      Atomics.store(progressArray, cancellationIndex, 1);
    } else {
      progressArray[cancellationIndex] = 1;
    }

    session.reject(new CancellationError('Search cancelled'));
    this.activeSessions.delete(requestId);
  }

  private async searchTimelinesMainThread(
    requestId: string,
    celExpr: string,
    allTimelineIds: readonly number[],
    totalTimelines: number,
    startTime: number,
    onProgress?: (current: number, total: number) => void,
  ): Promise<Set<number>> {
    const progressSab = new ArrayBuffer(8);
    const session: SearchSession = {
      type: 'SEARCH_TIMELINES',
      celExpr,
      timelineIds: allTimelineIds,
      nextIndex: 0,
      matchedIds: new Set<number>(),
      activeWorkersCount: 1, // acts as flag
      progressSab,
      resolve: () => {},
      reject: () => {},
    };
    this.activeSessions.set(requestId, session);

    try {
      // Yield to allow synchronous follow-up searches to cancel this search before it starts.
      await Promise.resolve();

      let previousTime = performance.now();
      while (session.nextIndex < session.timelineIds.length) {
        if (!this.activeSessions.has(requestId))
          throw new Error('Search cancelled');

        const requestView = new Int32Array(this.requestBufs[0]);
        let count = 0;
        while (
          session.nextIndex < session.timelineIds.length &&
          count < MAX_TIMELINES_PER_FRAME_ON_MAIN_THREAD
        ) {
          requestView[++count] = session.timelineIds[session.nextIndex++];
        }
        requestView[0] = count;

        handleSearchTimelines(
          {
            type: 'SEARCH_TIMELINES',
            requestId,
            workerIndex: 0,
            numWorkers: 1,
            celExpr,
            requestBuf: this.requestBufs[0],
            resultBuf: this.resultBufs[0],
            progressSab,
          },
          this.mainThreadState,
        );

        const resultView = new Int32Array(this.resultBufs[0]);
        const matchCount = resultView[0];
        for (let i = 1; i <= matchCount; i++) {
          session.matchedIds.add(resultView[i]);
        }

        if (onProgress) onProgress(session.nextIndex, totalTimelines);
        previousTime = await this.waitIfExceedsFrame(previousTime);
      }

      if (onProgress) onProgress(totalTimelines, totalTimelines);
      const elapsedMs = performance.now() - startTime;
      console.debug(
        `[SearchWorkerManager MainThread] searchTimelines finished:\n  Query: "${celExpr}"\n  Elapsed Time: ${elapsedMs.toFixed(2)}ms`,
      );
      return session.matchedIds;
    } finally {
      this.activeSessions.delete(requestId);
      if (this.activeTimelineSearchRequestId === requestId) {
        this.activeTimelineSearchRequestId = null;
      }
    }
  }

  private async searchLogsMainThread(
    requestId: string,
    celExpr: string,
    timelineIds: readonly number[],
    totalLogs: number,
    startTime: number,
    onProgress?: (current: number, total: number) => void,
  ): Promise<Set<number>> {
    const progressSab = new ArrayBuffer(8);
    const session: SearchSession = {
      type: 'SEARCH_LOGS',
      celExpr,
      timelineIds,
      nextIndex: 0,
      matchedIds: new Set<number>(),
      activeWorkersCount: 1,
      progressSab,
      resolve: () => {},
      reject: () => {},
    };
    this.activeSessions.set(requestId, session);

    try {
      // Yield to allow synchronous follow-up searches to cancel this search before it starts.
      await Promise.resolve();

      let previousProcessed = 0;
      let previousTime = performance.now();
      while (session.nextIndex < session.timelineIds.length) {
        if (!this.activeSessions.has(requestId))
          throw new Error('Search cancelled');

        const requestView = new Int32Array(this.requestBufs[0]);
        let count = 0;
        let logsInChunk = 0;
        while (
          session.nextIndex < session.timelineIds.length &&
          count < MAX_TIMELINES_PER_FRAME_ON_MAIN_THREAD
        ) {
          const tId = session.timelineIds[session.nextIndex];
          const t = this.activeTimelineStore?.getTimeline(tId);
          const tLogs = t ? t.events.length + t.revisions.length : 0;
          if (
            count > 0 &&
            logsInChunk + tLogs > MAX_LOGS_PER_FRAME_ON_MAIN_THREAD
          ) {
            break;
          }
          requestView[++count] = tId;
          logsInChunk += tLogs;
          session.nextIndex++;
        }
        requestView[0] = count;

        const progressArray = new Int32Array(progressSab);
        progressArray[0] = 0;

        handleSearchLogs(
          {
            type: 'SEARCH_LOGS',
            requestId,
            workerIndex: 0,
            numWorkers: 1,
            celExpr,
            requestBuf: this.requestBufs[0],
            resultBuf: this.resultBufs[0],
            progressSab,
          },
          this.mainThreadState,
        );

        const resultView = new Int32Array(this.resultBufs[0]);
        const matchCount = resultView[0];
        for (let i = 1; i <= matchCount; i++) {
          session.matchedIds.add(resultView[i]);
        }

        previousProcessed += progressArray[0];
        if (onProgress) onProgress(previousProcessed, totalLogs);

        previousTime = await this.waitIfExceedsFrame(previousTime);
      }

      if (onProgress) onProgress(totalLogs, totalLogs);
      const elapsedMs = performance.now() - startTime;
      console.debug(
        `[SearchWorkerManager MainThread] searchLogs finished:\n  Query: "${celExpr}"\n  Elapsed Time: ${elapsedMs.toFixed(2)}ms`,
      );
      return session.matchedIds;
    } finally {
      this.activeSessions.delete(requestId);
      if (this.activeLogSearchRequestId === requestId) {
        this.activeLogSearchRequestId = null;
      }
    }
  }

  private async waitIfExceedsFrame(
    beginMsOfCurrentTask: number,
  ): Promise<number> {
    if (
      performance.now() - beginMsOfCurrentTask >
      MAX_PROCESSING_TIME_MS_ON_MAIN_THREAD
    ) {
      await new Promise((resolve) => setTimeout(resolve, 0));
      return performance.now();
    }
    return beginMsOfCurrentTask;
  }

  ngOnDestroy(): void {
    for (const worker of this.workers) {
      worker.terminate();
    }
  }
}
