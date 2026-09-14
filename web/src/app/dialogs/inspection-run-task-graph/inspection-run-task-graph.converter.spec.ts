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

import { create } from '@bufbuild/protobuf';
import {
  computeRunElapsedMs,
  computeTaskRunDurationMs,
  convertToDagNodeRunPhase,
  convertToDagViewerEdges,
  convertToDagViewerNodes,
  countFinishedTasks,
} from 'src/app/dialogs/inspection-run-task-graph/inspection-run-task-graph.converter';
import {
  InspectionRunTaskGraphSnapshot,
  InspectionRunTaskGraphSnapshotSchema,
  TaskDAGEdgeSchema,
  TaskDAGNodeSchema,
  TaskRunNodeStatus,
  TaskRunNodeStatusSchema,
  TaskRunPhase,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagNodeRunPhase,
  FORM_TASK_LABEL_KEY,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

const MILLI_IN_NANO = 1_000_000n;

function createStatus(
  taskImplementationId: string,
  phase: TaskRunPhase,
  startTimeUnixNano: bigint,
  endTimeUnixNano: bigint,
): TaskRunNodeStatus {
  return create(TaskRunNodeStatusSchema, {
    taskImplementationId,
    phase,
    startTimeUnixNano,
    endTimeUnixNano,
  });
}

function createSnapshot(
  nodeStatuses: TaskRunNodeStatus[],
  isRunFinished: boolean,
  snapshotTimeUnixNano: bigint,
): InspectionRunTaskGraphSnapshot {
  return create(InspectionRunTaskGraphSnapshotSchema, {
    nodeStatuses,
    isRunFinished,
    snapshotTimeUnixNano,
  });
}

describe('convertToDagNodeRunPhase', () => {
  const testCases: {
    name: string;
    phase: TaskRunPhase;
    want: DagNodeRunPhase;
  }[] = [
    {
      name: 'maps WAITING',
      phase: TaskRunPhase.WAITING,
      want: DagNodeRunPhase.WAITING,
    },
    {
      name: 'maps RUNNING',
      phase: TaskRunPhase.RUNNING,
      want: DagNodeRunPhase.RUNNING,
    },
    {
      name: 'maps DONE',
      phase: TaskRunPhase.DONE,
      want: DagNodeRunPhase.DONE,
    },
    {
      name: 'maps ERROR',
      phase: TaskRunPhase.ERROR,
      want: DagNodeRunPhase.ERROR,
    },
    {
      name: 'falls back to NONE for UNSPECIFIED',
      phase: TaskRunPhase.UNSPECIFIED,
      want: DagNodeRunPhase.NONE,
    },
  ];

  for (const testCase of testCases) {
    it(testCase.name, () => {
      expect(convertToDagNodeRunPhase(testCase.phase)).toBe(testCase.want);
    });
  }
});

describe('computeTaskRunDurationMs', () => {
  it('returns zero while the task has not started', () => {
    const status = createStatus('a#default', TaskRunPhase.WAITING, 0n, 0n);

    expect(computeTaskRunDurationMs(status, 100n * MILLI_IN_NANO)).toBe(0);
  });

  it('measures against the snapshot time while the task is running', () => {
    const status = createStatus(
      'a#default',
      TaskRunPhase.RUNNING,
      10n * MILLI_IN_NANO,
      0n,
    );

    expect(computeTaskRunDurationMs(status, 35n * MILLI_IN_NANO)).toBe(25);
  });

  it('measures against the end time once the task finished', () => {
    const status = createStatus(
      'a#default',
      TaskRunPhase.DONE,
      10n * MILLI_IN_NANO,
      30n * MILLI_IN_NANO,
    );

    expect(computeTaskRunDurationMs(status, 90n * MILLI_IN_NANO)).toBe(20);
  });

  it('returns zero when the end time is not after the start time', () => {
    const status = createStatus(
      'a#default',
      TaskRunPhase.DONE,
      30n * MILLI_IN_NANO,
      20n * MILLI_IN_NANO,
    );

    expect(computeTaskRunDurationMs(status, 90n * MILLI_IN_NANO)).toBe(0);
  });
});

describe('convertToDagViewerNodes', () => {
  const nodes = [
    create(TaskDAGNodeSchema, {
      taskImplementationId: 'a#default',
      taskReferenceId: 'a',
      isFeature: true,
      topologicalOrder: 0,
      priority: 10,
      labels: { [FORM_TASK_LABEL_KEY]: 'true' },
      outputType: 'string',
    }),
    create(TaskDAGNodeSchema, {
      taskImplementationId: 'b#default',
      taskReferenceId: 'b',
      topologicalOrder: 1,
    }),
  ];

  it('decorates nodes with the run phase and duration of the matching status', () => {
    const statuses = new Map([
      [
        'a#default',
        createStatus(
          'a#default',
          TaskRunPhase.RUNNING,
          10n * MILLI_IN_NANO,
          0n,
        ),
      ],
    ]);

    const got = convertToDagViewerNodes(nodes, statuses, 40n * MILLI_IN_NANO);

    expect(got[0].id).toBe('a#default');
    expect(got[0].isFeature).toBeTrue();
    expect(got[0].isFormTask).toBeTrue();
    expect(got[0].outputType).toBe('string');
    expect(got[0].runPhase).toBe(DagNodeRunPhase.RUNNING);
    expect(got[0].runDurationMs).toBe(30);
  });

  it('leaves nodes without a status undecorated', () => {
    const got = convertToDagViewerNodes(nodes, new Map(), 40n * MILLI_IN_NANO);

    expect(got[1].runPhase).toBe(DagNodeRunPhase.NONE);
    expect(got[1].runDurationMs).toBe(0);
  });
});

describe('convertToDagViewerEdges', () => {
  it('assigns identifiers derived from the endpoints and the index', () => {
    const edges = [
      create(TaskDAGEdgeSchema, {
        sourceImplementationId: 'a#default',
        destinationImplementationId: 'b#default',
        sourceReferenceId: 'a',
        outputType: 'string',
      }),
    ];

    const got = convertToDagViewerEdges(edges);

    expect(got.length).toBe(1);
    expect(got[0].id).toBe('edge-0-a#default->b#default');
    expect(got[0].sourceId).toBe('a#default');
    expect(got[0].destinationId).toBe('b#default');
    expect(got[0].outputType).toBe('string');
  });
});

describe('computeRunElapsedMs', () => {
  it('returns zero while no task has started', () => {
    const snapshot = createSnapshot(
      [createStatus('a#default', TaskRunPhase.WAITING, 0n, 0n)],
      false,
      100n * MILLI_IN_NANO,
    );

    expect(computeRunElapsedMs(snapshot)).toBe(0);
  });

  it('measures from the earliest start to the snapshot time while running', () => {
    const snapshot = createSnapshot(
      [
        createStatus(
          'a#default',
          TaskRunPhase.DONE,
          10n * MILLI_IN_NANO,
          40n * MILLI_IN_NANO,
        ),
        createStatus(
          'b#default',
          TaskRunPhase.RUNNING,
          40n * MILLI_IN_NANO,
          0n,
        ),
      ],
      false,
      90n * MILLI_IN_NANO,
    );

    expect(computeRunElapsedMs(snapshot)).toBe(80);
  });

  it('measures from the earliest start to the latest end once finished', () => {
    const snapshot = createSnapshot(
      [
        createStatus(
          'a#default',
          TaskRunPhase.DONE,
          10n * MILLI_IN_NANO,
          40n * MILLI_IN_NANO,
        ),
        createStatus(
          'b#default',
          TaskRunPhase.DONE,
          40n * MILLI_IN_NANO,
          60n * MILLI_IN_NANO,
        ),
      ],
      true,
      900n * MILLI_IN_NANO,
    );

    expect(computeRunElapsedMs(snapshot)).toBe(50);
  });
});

describe('countFinishedTasks', () => {
  it('counts both succeeded and failed tasks', () => {
    const statuses = [
      createStatus('a#default', TaskRunPhase.DONE, 1n, 2n),
      createStatus('b#default', TaskRunPhase.ERROR, 1n, 2n),
      createStatus('c#default', TaskRunPhase.RUNNING, 1n, 0n),
      createStatus('d#default', TaskRunPhase.WAITING, 0n, 0n),
    ];

    expect(countFinishedTasks(statuses)).toBe(2);
  });
});
