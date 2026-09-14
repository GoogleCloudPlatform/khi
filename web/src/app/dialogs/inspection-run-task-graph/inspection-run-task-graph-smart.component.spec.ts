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
  ComponentFixture,
  TestBed,
  fakeAsync,
  flushMicrotasks,
  tick,
} from '@angular/core/testing';
import { MAT_DIALOG_DATA } from '@angular/material/dialog';
import { By } from '@angular/platform-browser';
import { create } from '@bufbuild/protobuf';
import { Code, ConnectError } from '@connectrpc/connect';
import { InspectionRunTaskGraphLayoutComponent } from 'src/app/dialogs/inspection-run-task-graph/components/inspection-run-task-graph-layout.component';
import {
  InspectionRunTaskGraphSmartComponent,
  MAX_CONSECUTIVE_STREAM_ERRORS,
} from 'src/app/dialogs/inspection-run-task-graph/inspection-run-task-graph-smart.component';
import { InspectionRunTaskGraphViewModel } from 'src/app/dialogs/inspection-run-task-graph/types/inspection-run-task-graph.viewmodel';
import {
  InspectionRunTaskGraphSnapshot,
  InspectionRunTaskGraphSnapshotSchema,
  TaskDAGInfoSchema,
  TaskDAGNodeSchema,
  TaskRunNodeStatusSchema,
  TaskRunPhase,
  WatchInspectionRunTaskGraphResponse,
  WatchInspectionRunTaskGraphResponseSchema,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';

const MILLI_IN_NANO = 1_000_000n;

type StreamFactory = () => AsyncIterable<WatchInspectionRunTaskGraphResponse>;

/**
 * Stream factory that stays open without ever emitting, mimicking an inspection that reports nothing yet.
 */
const pendingStreamFactory: StreamFactory = () => ({
  [Symbol.asyncIterator]() {
    return {
      next(): Promise<IteratorResult<WatchInspectionRunTaskGraphResponse>> {
        return new Promise(() => {});
      },
    };
  },
});

/**
 * Creates a StreamFactory yielding an AsyncIterable stream that emits the given responses in order.
 * If shouldStayOpen is true, the stream stays open without ending after emitting all responses.
 */
function createStreamFactory(
  responses: WatchInspectionRunTaskGraphResponse[],
  options?: { shouldStayOpen?: boolean },
): StreamFactory {
  return () => ({
    [Symbol.asyncIterator]() {
      let index = 0;
      return {
        next(): Promise<IteratorResult<WatchInspectionRunTaskGraphResponse>> {
          if (index < responses.length) {
            return Promise.resolve({ done: false, value: responses[index++] });
          }
          if (options?.shouldStayOpen) {
            return new Promise(() => {});
          }
          return Promise.resolve({
            done: true,
            value: undefined as unknown as WatchInspectionRunTaskGraphResponse,
          });
        },
      };
    },
  });
}

/**
 * Creates a StreamFactory yielding an AsyncIterable stream that immediately fails with the given error upon iteration.
 */
function createErrorStreamFactory(error: unknown): StreamFactory {
  return () => ({
    [Symbol.asyncIterator]() {
      return {
        next(): Promise<IteratorResult<WatchInspectionRunTaskGraphResponse>> {
          return Promise.reject(error);
        },
      };
    },
  });
}

function createSnapshot(
  isRunFinished: boolean,
  snapshotTimeUnixNano: bigint,
): InspectionRunTaskGraphSnapshot {
  return create(InspectionRunTaskGraphSnapshotSchema, {
    dag: create(TaskDAGInfoSchema, {
      nodes: [
        create(TaskDAGNodeSchema, {
          taskImplementationId: 'a#default',
          taskReferenceId: 'a',
        }),
        create(TaskDAGNodeSchema, {
          taskImplementationId: 'b#default',
          taskReferenceId: 'b',
        }),
        create(TaskDAGNodeSchema, {
          taskImplementationId: 'c#default',
          taskReferenceId: 'c',
        }),
      ],
    }),
    nodeStatuses: [
      create(TaskRunNodeStatusSchema, {
        taskImplementationId: 'a#default',
        phase: TaskRunPhase.DONE,
        startTimeUnixNano: 10n * MILLI_IN_NANO,
        endTimeUnixNano: 30n * MILLI_IN_NANO,
      }),
      create(TaskRunNodeStatusSchema, {
        taskImplementationId: 'b#default',
        phase: TaskRunPhase.ERROR,
        startTimeUnixNano: 20n * MILLI_IN_NANO,
        endTimeUnixNano: 50n * MILLI_IN_NANO,
      }),
      create(TaskRunNodeStatusSchema, {
        taskImplementationId: 'c#default',
        phase: isRunFinished ? TaskRunPhase.DONE : TaskRunPhase.RUNNING,
        startTimeUnixNano: 30n * MILLI_IN_NANO,
        endTimeUnixNano: isRunFinished ? 60n * MILLI_IN_NANO : 0n,
      }),
    ],
    isRunFinished,
    snapshotTimeUnixNano,
  });
}

function createResponse(
  snapshot: InspectionRunTaskGraphSnapshot,
): WatchInspectionRunTaskGraphResponse {
  return create(WatchInspectionRunTaskGraphResponseSchema, { snapshot });
}

describe('InspectionRunTaskGraphSmartComponent', () => {
  let fixture: ComponentFixture<InspectionRunTaskGraphSmartComponent>;
  let streams: StreamFactory[];
  let nextStreamIndex: number;
  let lastCapturedInspectionId: string | undefined;
  let lastCapturedSignal: AbortSignal | undefined;

  beforeEach(async () => {
    streams = [];
    nextStreamIndex = 0;
    lastCapturedInspectionId = undefined;
    lastCapturedSignal = undefined;
    const connectClientMock = {
      inspectionTaskGraphClient: {
        watchInspectionRunTaskGraph: (
          req: { inspectionId: string },
          options?: { signal?: AbortSignal },
        ) => {
          lastCapturedInspectionId = req.inspectionId;
          lastCapturedSignal = options?.signal;
          const factory = streams[nextStreamIndex] ?? pendingStreamFactory;
          nextStreamIndex++;
          return factory();
        },
      },
    };

    await TestBed.configureTestingModule({
      imports: [InspectionRunTaskGraphSmartComponent],
      providers: [
        {
          provide: ConnectClientService,
          useValue: connectClientMock as unknown as ConnectClientService,
        },
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            inspectionId: 'inspection-1',
            inspectionName: 'sample inspection',
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(InspectionRunTaskGraphSmartComponent);
  });

  function currentViewModel(): InspectionRunTaskGraphViewModel {
    const layout = fixture.debugElement.query(
      By.directive(InspectionRunTaskGraphLayoutComponent),
    ).componentInstance as InspectionRunTaskGraphLayoutComponent;
    return layout.viewModel();
  }

  it('shows an empty summary carrying the dialog title until the first snapshot arrives', fakeAsync(() => {
    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    const viewModel = currentViewModel();
    expect(viewModel.inspectionName).toBe('sample inspection');
    expect(viewModel.totalTaskCount).toBe(0);
    expect(viewModel.nodes.length).toBe(0);
    expect(viewModel.elapsedMs).toBe(0);
    expect(viewModel.isRunFinished).toBeFalse();
  }));

  it('counts terminal tasks and measures elapsed time against the snapshot time while running', fakeAsync(() => {
    streams.push(
      createStreamFactory(
        [createResponse(createSnapshot(false, 90n * MILLI_IN_NANO))],
        { shouldStayOpen: true },
      ),
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    const viewModel = currentViewModel();
    expect(viewModel.totalTaskCount).toBe(3);
    expect(viewModel.finishedTaskCount).toBe(2);
    expect(viewModel.elapsedMs).toBe(80);
    expect(viewModel.isRunFinished).toBeFalse();
    expect(viewModel.nodes.length).toBe(3);
  }));

  it('measures elapsed time against the last end time once the run finished', fakeAsync(() => {
    streams.push(
      createStreamFactory([
        createResponse(createSnapshot(true, 900n * MILLI_IN_NANO)),
      ]),
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    const viewModel = currentViewModel();
    expect(viewModel.finishedTaskCount).toBe(3);
    expect(viewModel.elapsedMs).toBe(50);
    expect(viewModel.isRunFinished).toBeTrue();
  }));

  it('surfaces a stream failure and clears it once the reconnect delivers a snapshot', fakeAsync(() => {
    streams.push(
      createErrorStreamFactory(
        new ConnectError('backend unavailable', Code.Unavailable),
      ),
      createStreamFactory([
        createResponse(createSnapshot(true, 900n * MILLI_IN_NANO)),
      ]),
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    expect(currentViewModel().watchErrorMessage).toContain(
      'backend unavailable',
    );

    tick(1000);
    flushMicrotasks();
    fixture.detectChanges();

    expect(currentViewModel().watchErrorMessage).toBe('');
    expect(currentViewModel().isRunFinished).toBeTrue();
  }));

  it('does not reconnect when stream fails with a non-retryable error', fakeAsync(() => {
    streams.push(
      createErrorStreamFactory(
        new ConnectError('permission denied', Code.PermissionDenied),
      ),
      createStreamFactory([
        createResponse(createSnapshot(true, 900n * MILLI_IN_NANO)),
      ]),
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    expect(currentViewModel().watchErrorMessage).toContain('permission denied');

    tick(2000);
    tick();
    fixture.detectChanges();

    expect(currentViewModel().watchErrorMessage).toContain('permission denied');
    expect(currentViewModel().isRunFinished).toBeFalse();
  }));

  it('reconnects when the server closes the stream before the run finished', fakeAsync(() => {
    streams.push(
      createStreamFactory([
        createResponse(createSnapshot(false, 90n * MILLI_IN_NANO)),
      ]),
      createStreamFactory([
        createResponse(createSnapshot(true, 900n * MILLI_IN_NANO)),
      ]),
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    expect(currentViewModel().isRunFinished).toBeFalse();

    tick(1000);
    flushMicrotasks();
    fixture.detectChanges();

    expect(currentViewModel().isRunFinished).toBeTrue();
    expect(currentViewModel().finishedTaskCount).toBe(3);
  }));

  it('passes inspectionId and abort signal to client, and aborts stream on destroy', fakeAsync(() => {
    fixture.detectChanges();
    tick();

    expect(lastCapturedInspectionId).toBe('inspection-1');
    expect(lastCapturedSignal).toBeDefined();
    expect(lastCapturedSignal?.aborted).toBeFalse();

    fixture.destroy();

    expect(lastCapturedSignal?.aborted).toBeTrue();
  }));

  it('stops reconnecting after exceeding MAX_CONSECUTIVE_STREAM_ERRORS', fakeAsync(() => {
    for (let i = 0; i <= MAX_CONSECUTIVE_STREAM_ERRORS; i++) {
      streams.push(
        createErrorStreamFactory(
          new ConnectError(`failure ${i + 1}`, Code.Unavailable),
        ),
      );
    }

    fixture.detectChanges();
    tick();

    for (let i = 0; i < MAX_CONSECUTIVE_STREAM_ERRORS; i++) {
      tick(1000);
      flushMicrotasks();
      fixture.detectChanges();
    }

    expect(currentViewModel().watchErrorMessage).toContain(
      `failure ${MAX_CONSECUTIVE_STREAM_ERRORS + 1}`,
    );

    tick(5000);
    flushMicrotasks();
    fixture.detectChanges();

    expect(nextStreamIndex).toBe(MAX_CONSECUTIVE_STREAM_ERRORS + 1);
    expect(currentViewModel().isRunFinished).toBeFalse();
  }));

  it('resets consecutive error count when a snapshot is received', fakeAsync(() => {
    streams.push(
      createErrorStreamFactory(new ConnectError('error 1', Code.Unavailable)),
      createErrorStreamFactory(new ConnectError('error 2', Code.Unavailable)),
      createErrorStreamFactory(new ConnectError('error 3', Code.Unavailable)),
      createStreamFactory([
        createResponse(createSnapshot(false, 90n * MILLI_IN_NANO)),
      ]),
      createErrorStreamFactory(new ConnectError('error 4', Code.Unavailable)),
      createStreamFactory([
        createResponse(createSnapshot(true, 900n * MILLI_IN_NANO)),
      ]),
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();
    expect(currentViewModel().watchErrorMessage).toContain('error 1');

    tick(1000);
    flushMicrotasks();
    fixture.detectChanges();
    expect(currentViewModel().watchErrorMessage).toContain('error 2');

    tick(1000);
    flushMicrotasks();
    fixture.detectChanges();
    expect(currentViewModel().watchErrorMessage).toContain('error 3');

    tick(1000);
    flushMicrotasks();
    fixture.detectChanges();
    expect(currentViewModel().watchErrorMessage).toBe('');
    expect(currentViewModel().isRunFinished).toBeFalse();

    tick(1000);
    flushMicrotasks();
    fixture.detectChanges();
    expect(currentViewModel().watchErrorMessage).toContain('error 4');

    tick(1000);
    flushMicrotasks();
    fixture.detectChanges();
    expect(currentViewModel().watchErrorMessage).toBe('');
    expect(currentViewModel().isRunFinished).toBeTrue();
  }));
});
