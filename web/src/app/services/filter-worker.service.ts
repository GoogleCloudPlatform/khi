/**
 * Copyright 2024 Google LLC
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

import * as LogFilterWorker from '../worker/worker-types';
import { InspectionDataStoreService } from './inspection-data-store.service';
import { LogEntry } from '../store/log';
import {
  forkJoin,
  map,
  mergeMap,
  Observable,
  of,
  take,
  withLatestFrom,
} from 'rxjs';
import { ReferenceResolverStore } from '../common/loader/reference-resolver';
import { ConnectorPool } from './worker/pool';
import { WebWorkerConnector } from './worker/connector';

/**
 * FilterWorkerService provides log filter feature with regex.
 * KHI use multiple WebWorker to filter them because filtering massive count of logs with regex in main thread cause browser page freezed.
 */
export class FilterWorkerService {
  /**
   * The count of worker pool.
   */
  private static LOG_FILTER_WORKER_POOL_COUNT: number = 8;

  /**
   * The maximum count of logs. Logs are divided into subsets not exceeding this count and distributed to multiple workers.
   */
  private static MAX_LOG_COUNT_PER_SINGLE_FILTER_SUBTASK: number = 5000;

  private readonly workerPool: ConnectorPool;

  constructor(private dataStore: InspectionDataStoreService) {
    const workerConnectors = [];
    for (let i = 0; i < FilterWorkerService.LOG_FILTER_WORKER_POOL_COUNT; i++) {
      workerConnectors.push(
        new WebWorkerConnector(
          new Worker(
            new URL('../worker/log-filter/log-filter.worker', import.meta.url),
          ),
        ),
      );
    }
    this.workerPool = new ConnectorPool(workerConnectors);
  }

  public filterLogs(
    allLogs: LogEntry[],
    regexInStr: string,
  ): Observable<Set<number>> {
    // Split a regex filtering task into multiple smaller tasks.
    const tasks = [] as { start: number; length: number }[];
    for (
      let i = 0;
      i < allLogs.length;
      i += FilterWorkerService.MAX_LOG_COUNT_PER_SINGLE_FILTER_SUBTASK
    ) {
      tasks.push({
        start: i,
        length: Math.min(
          FilterWorkerService.MAX_LOG_COUNT_PER_SINGLE_FILTER_SUBTASK,
          allLogs.length - i,
        ),
      });
    }
    const filteredIndexSet = new Set<number>();
    return this.workerPool
      .requestSeriesOfTasks(tasks, (task) =>
        of(task).pipe(
          withLatestFrom(this.dataStore.referenceResolver),
          mergeMap(([task, referenceResolver]) =>
            FilterWorkerServieUtil.logEntriesToFilterWorkerLogs(
              referenceResolver,
              allLogs.slice(task.start, task.start + task.length),
            ),
          ),
          map(
            (value) =>
              ({
                regexInStr: regexInStr,
                logs: value,
              }) as LogFilterWorker.FilterQuery,
          ),
        ),
      )
      .pipe(
        map((indexedResponse) => {
          const payload =
            indexedResponse.response as LogFilterWorker.FilterResult;
          payload.notMatch.forEach((index) => filteredIndexSet.add(index));
          return filteredIndexSet;
        }),
      );
  }
}

/**
 * Provides utility functions used from FilterWorkerService.
 */
export class FilterWorkerServieUtil {
  /**
   * Convert an array of LogEntry to an array of LogFilterWorker.FilterWorkerLog with resolving its body text.
   */
  public static logEntriesToFilterWorkerLogs(
    referenceResolverStore: ReferenceResolverStore,
    logs: LogEntry[],
  ): Observable<LogFilterWorker.FilterWorkerLog[]> {
    return forkJoin(
      logs.map((l) =>
        referenceResolverStore.getText(l.body).pipe(
          map(
            (logBody) =>
              ({
                index: l.logIndex,
                logBody: logBody,
                logSummary: l.summary,
              }) as LogFilterWorker.FilterWorkerLog,
          ),
        ),
      ),
    ).pipe(take(1));
  }
}
