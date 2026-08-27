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

import { create, Message } from '@bufbuild/protobuf';
import {
  createClient,
  Interceptor,
  StreamRequest,
  StreamResponse,
  Transport,
} from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import {
  InspectionService,
  WatchInspectionsResponse,
  WatchInspectionsResponseSchema,
} from 'src/app/generated/api/v1/inspection_pb';
import {
  PopupService,
  WatchPopupResponse,
  WatchPopupResponseSchema,
} from 'src/app/generated/api/v1/popup_pb';
import {
  ServerStatusService,
  WatchServerStatResponse,
  WatchServerStatResponseSchema,
} from 'src/app/generated/api/v1/server_status_pb';
import {
  FilterTimelineRequest,
  FilterTimelineResponse,
  FilterTimelineResponseSchema,
  OpenWorkbenchRequest,
  OpenWorkbenchResponse,
  OpenWorkbenchResponseSchema,
  OpenWorkbenchResponse_Stage,
  WatchIndexProgressRequest,
  WatchIndexProgressResponse,
  WatchIndexProgressResponseSchema,
  WatchIndexProgressResponse_IndexState,
  WorkbenchService,
} from 'src/app/generated/api/v1/workbench_pb';
import { ApiPathUtil } from 'src/app/services/api/api-path-util';
import { environment } from 'src/environments/environment';

/**
 * Checks whether legacy polling mode is enabled via URL query parameter or environment configuration.
 */
export function isLegacyPollingMode(): boolean {
  if (typeof window !== 'undefined') {
    const params = new URLSearchParams(window.location.search);
    if (params.get('pollLegacy') === 'true') {
      return true;
    }
  }
  return environment.usePollingLegacy ?? false;
}

/**
 * StreamToPollAdapter maps a stream request into an async iterable of response messages via unary polling.
 */
export type StreamToPollAdapter = (
  req: StreamRequest,
  transport: Transport,
) => AsyncIterable<Message>;

/**
 * Retrieves the first message from an AsyncIterable stream request.
 */
async function getFirstMessage<T>(iterable: AsyncIterable<T>): Promise<T> {
  const iterator = iterable[Symbol.asyncIterator]();
  const res = await iterator.next();
  if (res.done || res.value === undefined) {
    throw new Error('No request message received.');
  }
  return res.value;
}

/**
 * Delays execution for the specified milliseconds, resolving early if the signal is aborted.
 */
function delay(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal?.aborted) {
      resolve();
      return;
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort);
      resolve();
    }, ms);
    const onAbort = () => {
      clearTimeout(timer);
      resolve();
    };
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}

/**
 * Adapts WatchPopup streaming RPC to unary PullPopup calls.
 */
async function* adaptWatchPopup(
  req: StreamRequest,
  transport: Transport,
): AsyncIterable<WatchPopupResponse> {
  const popupClient = createClient(PopupService, transport);
  let lastPopupId: string | null = null;
  while (!req.signal.aborted) {
    try {
      const res = await popupClient.pullPopup({}, { signal: req.signal });
      if (res.popup && res.popup.id !== lastPopupId) {
        lastPopupId = res.popup.id;
        yield create(WatchPopupResponseSchema, {
          event: { case: 'popup', value: res.popup },
        });
      } else if (!res.popup && lastPopupId !== null) {
        lastPopupId = null;
        yield create(WatchPopupResponseSchema, {
          event: { case: 'dismissed', value: true },
        });
      }
    } catch {
      if (req.signal.aborted) {
        return;
      }
    }
    await delay(500, req.signal);
  }
}

/**
 * Adapts WatchServerStat streaming RPC to unary PullServerStat calls.
 */
async function* adaptWatchServerStat(
  req: StreamRequest,
  transport: Transport,
): AsyncIterable<WatchServerStatResponse> {
  const statClient = createClient(ServerStatusService, transport);
  while (!req.signal.aborted) {
    try {
      const res = await statClient.pullServerStat({}, { signal: req.signal });
      yield create(WatchServerStatResponseSchema, {
        serverStat: res.serverStat,
      });
    } catch {
      if (req.signal.aborted) {
        return;
      }
    }
    await delay(1000, req.signal);
  }
}

/**
 * Adapts WatchInspections streaming RPC to unary PullInspections calls.
 */
async function* adaptWatchInspections(
  req: StreamRequest,
  transport: Transport,
): AsyncIterable<WatchInspectionsResponse> {
  const inspClient = createClient(InspectionService, transport);
  while (!req.signal.aborted) {
    try {
      const res = await inspClient.pullInspections({}, { signal: req.signal });
      yield create(WatchInspectionsResponseSchema, {
        inspections: res.inspections,
      });
    } catch {
      if (req.signal.aborted) {
        return;
      }
    }
    await delay(500, req.signal);
  }
}

/**
 * Adapts WatchIndexProgress streaming RPC to unary PullIndexProgress calls.
 */
async function* adaptWatchIndexProgress(
  req: StreamRequest,
  transport: Transport,
): AsyncIterable<WatchIndexProgressResponse> {
  const wbClient = createClient(WorkbenchService, transport);
  const input = (await getFirstMessage(
    req.message,
  )) as WatchIndexProgressRequest;
  while (!req.signal.aborted) {
    try {
      const res = await wbClient.pullIndexProgress(
        { workbenchId: input.workbenchId },
        {
          signal: req.signal,
        },
      );
      yield create(WatchIndexProgressResponseSchema, {
        state: res.state,
        progressPercentage: res.progressPercentage,
        message: res.message,
      });
      if (
        res.state === WatchIndexProgressResponse_IndexState.READY ||
        res.state === WatchIndexProgressResponse_IndexState.FAILED
      ) {
        return;
      }
    } catch {
      if (req.signal.aborted) {
        return;
      }
    }
    await delay(500, req.signal);
  }
}

/**
 * Adapts OpenWorkbench streaming RPC to OpenWorkbenchSync calls with cancellation support.
 */
async function* adaptOpenWorkbench(
  req: StreamRequest,
  transport: Transport,
): AsyncIterable<OpenWorkbenchResponse> {
  const wbClient = createClient(WorkbenchService, transport);
  const input = (await getFirstMessage(req.message)) as OpenWorkbenchRequest;
  let jobId = '';
  try {
    while (!req.signal.aborted) {
      const res = await wbClient.openWorkbenchSync(
        {
          userId: input.userId,
          sessionId: input.sessionId,
          inspectionId: input.inspectionId,
          jobId,
        },
        { signal: req.signal },
      );
      jobId = res.jobId;
      yield create(OpenWorkbenchResponseSchema, {
        stage: res.stage,
        progressPercentage: res.progressPercentage,
        message: res.message,
        workbenchId: res.workbenchId,
      });
      if (res.stage === OpenWorkbenchResponse_Stage.READY) {
        return;
      }
    }
  } finally {
    if (req.signal.aborted && jobId) {
      void wbClient.cancelOpenWorkbenchSync({
        userId: input.userId,
        sessionId: input.sessionId,
        jobId,
      });
    }
  }
}

/**
 * Adapts FilterTimeline streaming RPC to FilterTimelineSync calls with cancellation support.
 */
async function* adaptFilterTimeline(
  req: StreamRequest,
  transport: Transport,
): AsyncIterable<FilterTimelineResponse> {
  const wbClient = createClient(WorkbenchService, transport);
  const input = (await getFirstMessage(req.message)) as FilterTimelineRequest;
  let jobId = '';
  try {
    const initialRes = await wbClient.filterTimelineSync(
      {
        workbenchId: input.workbenchId,
        timelineQuery: input.timelineQuery,
        timelineExclusionQuery: input.timelineExclusionQuery,
        logQuery: input.logQuery,
        excludeNoLogs: input.excludeNoLogs,
        jobId: '',
      },
      { signal: req.signal },
    );
    jobId = initialRes.jobId;
    if (initialRes.progress) {
      yield create(FilterTimelineResponseSchema, {
        payload: { case: 'progress', value: initialRes.progress },
      });
    }
    if (initialRes.isDone) {
      if (initialRes.errorMessage) {
        throw new Error(initialRes.errorMessage);
      }
      if (initialRes.result) {
        yield create(FilterTimelineResponseSchema, {
          payload: { case: 'result', value: initialRes.result },
        });
      }
      return;
    }

    while (!req.signal.aborted) {
      const pollRes = await wbClient.filterTimelineSync(
        {
          workbenchId: input.workbenchId,
          jobId,
        },
        { signal: req.signal },
      );
      if (pollRes.progress) {
        yield create(FilterTimelineResponseSchema, {
          payload: { case: 'progress', value: pollRes.progress },
        });
      }
      if (pollRes.isDone) {
        if (pollRes.errorMessage) {
          throw new Error(pollRes.errorMessage);
        }
        if (pollRes.result) {
          yield create(FilterTimelineResponseSchema, {
            payload: { case: 'result', value: pollRes.result },
          });
        }
        return;
      }
    }
  } finally {
    if (req.signal.aborted && jobId) {
      void wbClient.cancelFilterTimelineSync({
        workbenchId: input.workbenchId,
        jobId,
      });
    }
  }
}

const STREAM_TO_POLL_ADAPTERS = new Map<string, StreamToPollAdapter>([
  ['WatchPopup', adaptWatchPopup],
  ['WatchServerStat', adaptWatchServerStat],
  ['WatchInspections', adaptWatchInspections],
  ['WatchIndexProgress', adaptWatchIndexProgress],
  ['OpenWorkbench', adaptOpenWorkbench],
  ['FilterTimeline', adaptFilterTimeline],
]);

/**
 * Creates an interceptor that transparently converts server-streaming RPCs into polling-based unary RPC calls.
 *
 * @param customUnaryTransport Optional unary transport used for polling requests, primarily for unit testing.
 * @param forceLegacyPolling Optional boolean flag overriding the default isLegacyPollingMode check.
 */
export function createLegacyPollingInterceptor(
  customUnaryTransport?: Transport,
  forceLegacyPolling?: boolean,
): Interceptor {
  const unaryTransport =
    customUnaryTransport ??
    createConnectTransport({
      baseUrl: ApiPathUtil.getServerBaseUrl(),
      useBinaryFormat: environment.production,
    });

  return (next) => async (req) => {
    const isLegacy = forceLegacyPolling ?? isLegacyPollingMode();
    if (!req.stream || !isLegacy) {
      return next(req);
    }

    const adapter = STREAM_TO_POLL_ADAPTERS.get(req.method.name);
    if (!adapter) {
      return next(req);
    }

    const responseStream = adapter(req, unaryTransport);

    const response: StreamResponse = {
      stream: true,
      service: req.service,
      method: req.method,
      header: new Headers(),
      trailer: new Headers(),
      message: responseStream,
    };
    return response;
  };
}
