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
import { searchWorkerState } from 'src/app/worker/search/search-worker-state';
import { handleSyncData } from 'src/app/worker/search/handlers/sync-data';
import { handleSearchTimelines } from 'src/app/worker/search/handlers/search-timelines';
import { handleSearchLogs } from 'src/app/worker/search/handlers/search-logs';

addEventListener('message', (event: MessageEvent<SearchWorkerRequest>) => {
  const request = event.data;

  try {
    switch (request.type) {
      case 'SYNC_DATA':
        handleSyncData(request, searchWorkerState);
        break;
      case 'SEARCH_TIMELINES': {
        const matchedIds = handleSearchTimelines(request, searchWorkerState);
        postMessage({
          type: 'SEARCH_COMPLETE',
          requestId: request.requestId,
          matchedIds,
        } as SearchWorkerResponse);
        break;
      }
      case 'SEARCH_LOGS': {
        const matchedIds = handleSearchLogs(request, searchWorkerState);
        postMessage({
          type: 'SEARCH_COMPLETE',
          requestId: request.requestId,
          matchedIds,
        } as SearchWorkerResponse);
        break;
      }
      default:
        throw new Error(`Unknown request type`);
    }
  } catch (error) {
    postMessage({
      type: 'ERROR',
      requestId: request.requestId,
      error: error instanceof Error ? error.message : String(error),
    } as SearchWorkerResponse);
  }
});
