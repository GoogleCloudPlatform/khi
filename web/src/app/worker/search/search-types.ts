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

import { InternPoolSharedData } from 'src/app/store/domain/intern-pool-store';
import { LogStoreSharedData } from 'src/app/store/domain/log-store';
import { TimelineStoreSharedData } from 'src/app/store/domain/timeline-store';
import { StyleStoreSharedData } from 'src/app/store/domain/style';

/**
 * Message request type sent from the manager (main thread) to SearchWorkers.
 */
export type SearchWorkerRequest =
  | {
      readonly type: 'SYNC_DATA';
      readonly requestId: string;
      readonly workerIndex: number;
      readonly internPoolSharedData: InternPoolSharedData;
      readonly logStoreSharedData: LogStoreSharedData;
      readonly timelineStoreSharedData: TimelineStoreSharedData;
      readonly styleStoreSharedData: StyleStoreSharedData;
    }
  | {
      readonly type: 'SEARCH_TIMELINES';
      readonly requestId: string;
      readonly workerIndex: number;
      readonly numWorkers: number;
      readonly celExpr: string;
      readonly progressSab: SharedArrayBuffer | ArrayBuffer;
    }
  | {
      readonly type: 'SEARCH_LOGS';
      readonly requestId: string;
      readonly workerIndex: number;
      readonly numWorkers: number;
      readonly celExpr: string;
      readonly timelineIds: readonly number[]; // timelines target for evaluating events & revisions
      readonly progressSab: SharedArrayBuffer | ArrayBuffer;
    };

/**
 * Message response type sent from SearchWorkers back to the manager.
 */
export type SearchWorkerResponse =
  | {
      readonly type: 'SYNC_COMPLETE';
      readonly requestId: string;
    }
  | {
      readonly type: 'SEARCH_COMPLETE';
      readonly requestId: string;
      readonly matchedIds: readonly number[];
    }
  | {
      readonly type: 'ERROR';
      readonly requestId?: string;
      readonly error: string;
    };
