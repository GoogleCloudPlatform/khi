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

interface SearchSession {
  readonly expectedResponses: number;
  readonly matchedIds: number[];
  receivedCount: number;
  readonly workerProcessed: number[];
  readonly workerTotals: number[];
  readonly onProgress?: (current: number, total: number) => void;
  readonly progressSab: SharedArrayBuffer;
  resolve(matchedIds: number[]): void;
  reject(error: Error): void;
}

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

  constructor() {
    this.numWorkers = 8; //Math.max(1, (navigator.hardwareConcurrency || 4) - 1);
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
   * Partitions timeline IDs and executes CEL timeline filtering concurrently across workers.
   */
  public async searchTimelines(
    celExpr: string,
    onProgress?: (current: number, total: number) => void,
  ): Promise<Set<number>> {
    if (celExpr === '') {
      const allIds = this.activeTimelineStore?.timelines.map((t) => t.id) ?? [];
      return new Set(allIds);
    }

    if (this.activeTimelineSearchRequestId) {
      this.cancelSearch(this.activeTimelineSearchRequestId);
    }

    const totalTimelines = this.activeTimelineStore?.timelines.length ?? 0;
    const startTime = performance.now();
    const requestId = `search-t-${this.sessionCounter++}`;
    this.activeTimelineSearchRequestId = requestId;

    const progressSab = new SharedArrayBuffer((this.workers.length + 1) * 64);
    const progressArray = new Int32Array(progressSab);

    let intervalId: ReturnType<typeof setInterval> | null = null;
    if (onProgress) {
      intervalId = setInterval(() => {
        let current = 0;
        for (let i = 0; i < this.workers.length; i++) {
          current += progressArray[i * 16];
        }
        onProgress(current, totalTimelines);
      }, 50);
    }

    const promise = new Promise<number[]>((resolve, reject) => {
      this.activeSessions.set(requestId, {
        expectedResponses: this.workers.length,
        matchedIds: [],
        receivedCount: 0,
        workerProcessed: Array.from({ length: this.workers.length }, () => 0),
        workerTotals: Array.from({ length: this.workers.length }, () => 0),
        onProgress,
        progressSab,
        resolve,
        reject,
      });
    });

    const cleanup = () => {
      if (intervalId !== null) {
        clearInterval(intervalId);
      }
    };

    for (let i = 0; i < this.workers.length; i++) {
      this.workers[i].postMessage({
        type: 'SEARCH_TIMELINES',
        requestId,
        workerIndex: i,
        numWorkers: this.workers.length,
        celExpr,
        progressSab,
      } as SearchWorkerRequest);
    }

    try {
      const matchedList = await promise;
      if (onProgress) {
        onProgress(totalTimelines, totalTimelines);
      }
      const elapsedMs = performance.now() - startTime;
      const speed = elapsedMs > 0 ? totalTimelines / (elapsedMs / 1000) : 0;
      console.debug(
        `[SearchWorkerManager] searchTimelines finished:\n` +
          `  Query: "${celExpr}"\n` +
          `  Elapsed Time: ${elapsedMs.toFixed(2)}ms\n` +
          `  Search Speed: ${speed.toFixed(0)} timelines/sec`,
      );
      return new Set(matchedList);
    } finally {
      if (this.activeTimelineSearchRequestId === requestId) {
        this.activeTimelineSearchRequestId = null;
      }
      cleanup();
    }
  }

  /**
   * Partitions timeline IDs and executes CEL log filtering concurrently across workers.
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

    const progressSab = new SharedArrayBuffer((this.workers.length + 1) * 64);
    const progressArray = new Int32Array(progressSab);

    let intervalId: ReturnType<typeof setInterval> | null = null;
    if (onProgress) {
      intervalId = setInterval(() => {
        let current = 0;
        for (let i = 0; i < this.workers.length; i++) {
          current += progressArray[i * 16];
        }
        onProgress(current, totalLogs);
      }, 50);
    }

    const promise = new Promise<number[]>((resolve, reject) => {
      this.activeSessions.set(requestId, {
        expectedResponses: this.workers.length,
        matchedIds: [],
        receivedCount: 0,
        workerProcessed: Array.from({ length: this.workers.length }, () => 0),
        workerTotals: Array.from({ length: this.workers.length }, () => 0),
        onProgress,
        progressSab,
        resolve,
        reject,
      });
    });

    const cleanup = () => {
      if (intervalId !== null) {
        clearInterval(intervalId);
      }
    };

    const chunks = this.partition(timelineIds, this.numWorkers);

    for (let i = 0; i < this.workers.length; i++) {
      const chunk = chunks[i] || [];
      this.workers[i].postMessage({
        type: 'SEARCH_LOGS',
        requestId,
        workerIndex: i,
        numWorkers: this.workers.length,
        celExpr,
        timelineIds: chunk,
        progressSab,
      } as SearchWorkerRequest);
    }

    try {
      const matchedList = await promise;
      if (onProgress) {
        onProgress(totalLogs, totalLogs);
      }
      const elapsedMs = performance.now() - startTime;
      const speed = elapsedMs > 0 ? totalLogs / (elapsedMs / 1000) : 0;
      console.debug(
        `[SearchWorkerManager] searchLogs finished:\n` +
          `  Query: "${celExpr}"\n` +
          `  Elapsed Time: ${elapsedMs.toFixed(2)}ms\n` +
          `  Search Speed: ${speed.toFixed(0)} logs/sec`,
      );
      return new Set(matchedList);
    } finally {
      if (this.activeLogSearchRequestId === requestId) {
        this.activeLogSearchRequestId = null;
      }
      cleanup();
    }
  }

  private handleWorkerMessage(response: SearchWorkerResponse): void {
    if (response.type === 'SYNC_COMPLETE') {
      return; // Handled individually inside syncData() listeners
    }

    if (!response.requestId) {
      return;
    }

    const session = this.activeSessions.get(response.requestId);
    if (!session) {
      return;
    }

    if (response.type === 'ERROR') {
      session.reject(new Error(response.error));
      this.activeSessions.delete(response.requestId);
      return;
    }

    if (response.type === 'SEARCH_COMPLETE') {
      session.matchedIds.push(...response.matchedIds);
      session.receivedCount++;

      if (session.receivedCount === session.expectedResponses) {
        session.resolve(session.matchedIds);
        this.activeSessions.delete(response.requestId);
      }
    }
  }

  private cancelSearch(requestId: string): void {
    const session = this.activeSessions.get(requestId);
    if (!session) {
      return;
    }
    const progressArray = new Int32Array(session.progressSab);
    const cancellationIndex = this.numWorkers * 16;
    Atomics.store(progressArray, cancellationIndex, 1);

    session.reject(new Error('Search cancelled'));
    this.activeSessions.delete(requestId);
  }

  private partition<T>(array: readonly T[], numParts: number): T[][] {
    const result: T[][] = Array.from({ length: numParts }, () => []);
    for (let i = 0; i < array.length; i++) {
      result[i % numParts].push(array[i]);
    }
    return result;
  }

  ngOnDestroy(): void {
    for (const worker of this.workers) {
      worker.terminate();
    }
  }
}
