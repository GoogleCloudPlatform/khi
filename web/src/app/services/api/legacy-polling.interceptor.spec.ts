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
  create,
  DescMessage,
  DescMethodStreaming,
  DescMethodUnary,
  Message,
  MessageInitShape,
} from '@bufbuild/protobuf';
import {
  ContextValues,
  createClient,
  createContextValues,
  StreamRequest,
  StreamResponse,
  Transport,
  UnaryRequest,
  UnaryResponse,
} from '@connectrpc/connect';
import {
  InspectionListItemSchema,
  InspectionService,
  PullInspectionsResponseSchema,
} from 'src/app/generated/api/v1/inspection_pb';
import {
  PopupFormSchema,
  PopupService,
  PullPopupResponseSchema,
} from 'src/app/generated/api/v1/popup_pb';
import {
  PullServerStatResponseSchema,
  ServerStatSchema,
  ServerStatusService,
} from 'src/app/generated/api/v1/server_status_pb';
import {
  CancelFilterTimelineSyncResponseSchema,
  CancelOpenWorkbenchSyncResponseSchema,
  FilterProgressSchema,
  FilterResultMode,
  FilterResultSchema,
  FilterTimelineSyncResponseSchema,
  OpenWorkbenchResponse_Stage,
  OpenWorkbenchSyncResponseSchema,
  PullIndexProgressResponseSchema,
  WatchIndexProgressResponse_IndexState,
  WorkbenchService,
} from 'src/app/generated/api/v1/workbench_pb';
import {
  createLegacyPollingInterceptor,
  isLegacyPollingMode,
} from 'src/app/services/api/legacy-polling.interceptor';
import { environment } from 'src/environments/environment';

describe('LegacyPollingInterceptor', () => {
  describe('isLegacyPollingMode', () => {
    it('returns true when pollLegacy=true is present in location search', () => {
      const originalSearch = window.location.search;
      try {
        history.replaceState(null, '', '?pollLegacy=true');
        expect(isLegacyPollingMode()).toBeTrue();
      } finally {
        history.replaceState(null, '', originalSearch || '/');
      }
    });

    it('falls back to environment.usePollingLegacy when URL parameter is missing', () => {
      const originalSearch = window.location.search;
      try {
        history.replaceState(null, '', '/');
        expect(isLegacyPollingMode()).toBe(
          environment.usePollingLegacy ?? false,
        );
      } finally {
        history.replaceState(null, '', originalSearch || '/');
      }
    });
  });

  describe('interceptor behavior', () => {
    function createMockUnaryTransport(
      handler: (methodName: string, reqMsg: unknown) => Message,
    ): Transport {
      return {
        async unary<I extends DescMessage, O extends DescMessage>(
          method: DescMethodUnary<I, O>,
          _signal: AbortSignal | undefined,
          _timeoutMs: number | undefined,
          header: HeadersInit | undefined,
          message: MessageInitShape<I>,
        ): Promise<UnaryResponse<I, O>> {
          const resMsg = handler(method.name, message);
          const response = {
            stream: false as const,
            service: method.parent,
            method,
            header: new Headers(header),
            trailer: new Headers(),
            message: resMsg,
          };
          return response as UnaryResponse<I, O>;
        },
        async stream() {
          throw new Error('Streaming not supported on mock unary transport.');
        },
      };
    }

    function createMockStreamTransport(
      interceptor: ReturnType<typeof createLegacyPollingInterceptor>,
    ): Transport {
      return {
        async unary() {
          throw new Error('unary not used on stream client');
        },
        async stream<I extends DescMessage, O extends DescMessage>(
          method: DescMethodStreaming<I, O>,
          signal: AbortSignal | undefined,
          _timeoutMs: number | undefined,
          header: HeadersInit | undefined,
          message: AsyncIterable<MessageInitShape<I>>,
          contextValues: ContextValues | undefined,
        ): Promise<StreamResponse<I, O>> {
          const streamReq: StreamRequest = {
            stream: true,
            service: method.parent,
            method,
            url: `http://localhost/${method.name}`,
            requestMethod: 'POST',
            signal: signal ?? new AbortController().signal,
            header: new Headers(header),
            contextValues: contextValues ?? createContextValues(),
            message: message as AsyncIterable<Message>,
          };
          const next = async () => {
            throw new Error('next should not be called');
          };
          return (await interceptor(next)(streamReq)) as StreamResponse<I, O>;
        },
      };
    }

    it('passes through unary requests unchanged', async () => {
      const mockTransport = createMockUnaryTransport(() =>
        create(PullPopupResponseSchema, {}),
      );
      const interceptor = createLegacyPollingInterceptor(mockTransport, true);

      let nextCalled = false;
      const next = async (req: UnaryRequest | StreamRequest) => {
        nextCalled = true;
        const resp: UnaryResponse = {
          stream: false,
          service: req.service,
          method: (req as UnaryRequest).method,
          header: new Headers(),
          trailer: new Headers(),
          message: create(PullPopupResponseSchema, {}),
        };
        return resp;
      };

      const dummyReq: UnaryRequest = {
        stream: false,
        service: PopupService,
        method: PopupService.method.pullPopup,
        url: 'http://localhost/pullPopup',
        requestMethod: 'POST',
        signal: new AbortController().signal,
        header: new Headers(),
        contextValues: createContextValues(),
        message: create(PullPopupResponseSchema, {}),
      };

      await interceptor(next)(dummyReq);
      expect(nextCalled).toBeTrue();
    });

    it('adapts WatchPopup to PullPopup polling', async () => {
      let pullCount = 0;
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'PullPopup') {
          pullCount++;
          if (pullCount === 1) {
            return create(PullPopupResponseSchema, {
              popup: create(PopupFormSchema, {
                id: 'popup-1',
                title: 'First Popup',
              }),
            });
          }
          if (pullCount === 2) {
            return create(PullPopupResponseSchema, {});
          }
        }
        return create(PullPopupResponseSchema, {});
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(PopupService, clientTransport);
      const abortCtrl = new AbortController();
      const events: string[] = [];

      for await (const res of client.watchPopup(
        {},
        { signal: abortCtrl.signal },
      )) {
        if (res.event.case === 'popup') {
          events.push(`popup:${res.event.value.id}`);
        } else if (res.event.case === 'dismissed') {
          events.push('dismissed');
          abortCtrl.abort();
        }
      }

      expect(events).toEqual(['popup:popup-1', 'dismissed']);
    });

    it('adapts WatchServerStat to PullServerStat polling', async () => {
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'PullServerStat') {
          return create(PullServerStatResponseSchema, {
            serverStat: create(ServerStatSchema, {
              cpuUsagePercentage: 45.0,
            }),
          });
        }
        throw new Error(`Unexpected method: ${methodName}`);
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(ServerStatusService, clientTransport);
      const abortCtrl = new AbortController();
      for await (const res of client.watchServerStat(
        {},
        { signal: abortCtrl.signal },
      )) {
        expect(res.serverStat?.cpuUsagePercentage).toBe(45.0);
        abortCtrl.abort();
      }
    });

    it('adapts WatchInspections to PullInspections polling', async () => {
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'PullInspections') {
          return create(PullInspectionsResponseSchema, {
            inspections: [
              create(InspectionListItemSchema, {
                id: 'insp-1',
              }),
            ],
          });
        }
        throw new Error(`Unexpected method: ${methodName}`);
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(InspectionService, clientTransport);
      const abortCtrl = new AbortController();
      for await (const res of client.watchInspections(
        {},
        { signal: abortCtrl.signal },
      )) {
        expect(res.inspections.length).toBe(1);
        expect(res.inspections[0].id).toBe('insp-1');
        abortCtrl.abort();
      }
    });

    it('adapts WatchIndexProgress and terminates on READY', async () => {
      let callCount = 0;
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'PullIndexProgress') {
          callCount++;
          if (callCount === 1) {
            return create(PullIndexProgressResponseSchema, {
              state: WatchIndexProgressResponse_IndexState.BUILDING,
              progressPercentage: 50.0,
              message: 'Indexing...',
            });
          }
          return create(PullIndexProgressResponseSchema, {
            state: WatchIndexProgressResponse_IndexState.READY,
            progressPercentage: 100.0,
            message: 'Ready.',
          });
        }
        throw new Error(`Unexpected method: ${methodName}`);
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(WorkbenchService, clientTransport);
      const states: WatchIndexProgressResponse_IndexState[] = [];
      for await (const res of client.watchIndexProgress({
        workbenchId: 'wb-1',
      })) {
        states.push(res.state);
      }

      expect(states).toEqual([
        WatchIndexProgressResponse_IndexState.BUILDING,
        WatchIndexProgressResponse_IndexState.READY,
      ]);
    });

    it('adapts OpenWorkbench and sends cancellation on abort', async () => {
      let openCalls = 0;
      let canceled = false;
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'OpenWorkbenchSync') {
          openCalls++;
          return create(OpenWorkbenchSyncResponseSchema, {
            jobId: 'job-wb-open',
            stage: OpenWorkbenchResponse_Stage.READING_FILE,
            progressPercentage: 20.0,
            message: 'Reading data...',
          });
        }
        if (methodName === 'CancelOpenWorkbenchSync') {
          canceled = true;
          return create(CancelOpenWorkbenchSyncResponseSchema, {
            canceled: true,
          });
        }
        throw new Error(`Unexpected method: ${methodName}`);
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(WorkbenchService, clientTransport);
      const abortCtrl = new AbortController();
      for await (const res of client.openWorkbench(
        {
          userId: 'u1',
          sessionId: 's1',
          inspectionId: 'i1',
        },
        { signal: abortCtrl.signal },
      )) {
        expect(res.stage).toBe(OpenWorkbenchResponse_Stage.READING_FILE);
        abortCtrl.abort();
      }

      expect(openCalls).toBeGreaterThan(0);
      expect(canceled).toBeTrue();
    });

    it('adapts OpenWorkbench and yields stages until READY', async () => {
      let openCalls = 0;
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'OpenWorkbenchSync') {
          openCalls++;
          if (openCalls === 1) {
            return create(OpenWorkbenchSyncResponseSchema, {
              jobId: 'job-wb-ready',
              stage: OpenWorkbenchResponse_Stage.READING_FILE,
              progressPercentage: 20.0,
              message: 'Reading data...',
            });
          }
          return create(OpenWorkbenchSyncResponseSchema, {
            jobId: 'job-wb-ready',
            stage: OpenWorkbenchResponse_Stage.READY,
            progressPercentage: 100.0,
            message: 'Workbench ready.',
            workbenchId: 'wb-ready-1',
          });
        }
        throw new Error(`Unexpected method: ${methodName}`);
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(WorkbenchService, clientTransport);
      const stages: OpenWorkbenchResponse_Stage[] = [];
      for await (const res of client.openWorkbench({
        userId: 'u1',
        sessionId: 's1',
        inspectionId: 'i1',
      })) {
        stages.push(res.stage);
      }

      expect(stages).toEqual([
        OpenWorkbenchResponse_Stage.READING_FILE,
        OpenWorkbenchResponse_Stage.READY,
      ]);
      expect(openCalls).toBe(2);
    });

    it('adapts FilterTimeline and sends cancellation on abort', async () => {
      let filterCalls = 0;
      let canceled = false;
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'FilterTimelineSync') {
          filterCalls++;
          return create(FilterTimelineSyncResponseSchema, {
            jobId: 'job-filter-1',
            isDone: false,
            progress: create(FilterProgressSchema, {
              stageName: 'Timeline CEL',
              current: 10,
              total: 100,
            }),
          });
        }
        if (methodName === 'CancelFilterTimelineSync') {
          canceled = true;
          return create(CancelFilterTimelineSyncResponseSchema, {
            canceled: true,
          });
        }
        throw new Error(`Unexpected method: ${methodName}`);
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(WorkbenchService, clientTransport);
      const abortCtrl = new AbortController();
      for await (const res of client.filterTimeline(
        {
          workbenchId: 'wb-1',
        },
        { signal: abortCtrl.signal },
      )) {
        if (res.payload.case === 'progress') {
          expect(res.payload.value.stageName).toBe('Timeline CEL');
        }
        abortCtrl.abort();
      }

      expect(filterCalls).toBeGreaterThan(0);
      expect(canceled).toBeTrue();
    });

    it('adapts FilterTimeline and yields final result when done', async () => {
      let filterCalls = 0;
      const mockTransport = createMockUnaryTransport((methodName) => {
        if (methodName === 'FilterTimelineSync') {
          filterCalls++;
          if (filterCalls === 1) {
            return create(FilterTimelineSyncResponseSchema, {
              jobId: 'job-filter-done',
              isDone: false,
              progress: create(FilterProgressSchema, {
                stageName: 'Filtering',
                current: 50,
                total: 100,
              }),
            });
          }
          return create(FilterTimelineSyncResponseSchema, {
            jobId: 'job-filter-done',
            isDone: true,
            result: create(FilterResultSchema, {
              timelineMode: FilterResultMode.INCLUDE,
              logMode: FilterResultMode.INCLUDE,
            }),
          });
        }
        throw new Error(`Unexpected method: ${methodName}`);
      });

      const interceptor = createLegacyPollingInterceptor(mockTransport, true);
      const clientTransport = createMockStreamTransport(interceptor);
      const client = createClient(WorkbenchService, clientTransport);
      const modes: FilterResultMode[] = [];
      for await (const res of client.filterTimeline({
        workbenchId: 'wb-1',
      })) {
        if (res.payload.case === 'result') {
          modes.push(res.payload.value.timelineMode);
        }
      }

      expect(modes).toEqual([FilterResultMode.INCLUDE]);
    });
  });
});
