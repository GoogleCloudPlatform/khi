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
  tick,
} from '@angular/core/testing';
import { MAT_DIALOG_DATA } from '@angular/material/dialog';
import { By } from '@angular/platform-browser';
import { create } from '@bufbuild/protobuf';
import { Code, ConnectError } from '@connectrpc/connect';
import { InspectionRunTaskGraphLayoutComponent } from 'src/app/dialogs/inspection-run-task-graph/components/inspection-run-task-graph-layout.component';
import { InspectionRunTaskGraphSmartComponent } from 'src/app/dialogs/inspection-run-task-graph/inspection-run-task-graph-smart.component';
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
 * Stream that stays open without ever emitting, mimicking an inspection that reports nothing yet.
 */
async function* pendingStream(): AsyncGenerator<WatchInspectionRunTaskGraphResponse> {
  await new Promise<void>(() => {});
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

  beforeEach(async () => {
    streams = [];
    let nextStreamIndex = 0;
    const connectClientMock = {
      inspectionTaskGraphClient: {
        watchInspectionRunTaskGraph: () => {
          const factory = streams[nextStreamIndex] ?? pendingStream;
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
    streams.push(async function* () {
      yield createResponse(createSnapshot(false, 90n * MILLI_IN_NANO));
      await new Promise<void>(() => {});
    });

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
    streams.push(async function* () {
      yield createResponse(createSnapshot(true, 900n * MILLI_IN_NANO));
    });

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
      async function* () {
        throw new ConnectError('backend unavailable', Code.Unavailable);
      },
      async function* () {
        yield createResponse(createSnapshot(true, 900n * MILLI_IN_NANO));
      },
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    expect(currentViewModel().watchErrorMessage).toContain(
      'backend unavailable',
    );

    tick(1000);
    fixture.detectChanges();

    expect(currentViewModel().watchErrorMessage).toBe('');
    expect(currentViewModel().isRunFinished).toBeTrue();
  }));

  it('reconnects when the server closes the stream before the run finished', fakeAsync(() => {
    streams.push(
      async function* () {
        yield createResponse(createSnapshot(false, 90n * MILLI_IN_NANO));
      },
      async function* () {
        yield createResponse(createSnapshot(true, 900n * MILLI_IN_NANO));
      },
    );

    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    expect(currentViewModel().isRunFinished).toBeFalse();

    tick(1000);
    fixture.detectChanges();

    expect(currentViewModel().isRunFinished).toBeTrue();
    expect(currentViewModel().finishedTaskCount).toBe(3);
  }));
});
