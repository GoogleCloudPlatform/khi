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
  InspectionRunTaskGraphSnapshot,
  TaskDAGEdge,
  TaskDAGNode,
  TaskRunNodeStatus,
  TaskRunPhase,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagNodeRunPhase,
  DagViewerEdge,
  DagViewerNode,
  isFormTask,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

/**
 * Number of nanoseconds in a millisecond.
 */
const NANOS_PER_MILLI = 1_000_000;

/**
 * Converts a protobuf run phase into the phase understood by the DAG viewer.
 */
export function convertToDagNodeRunPhase(phase: TaskRunPhase): DagNodeRunPhase {
  switch (phase) {
    case TaskRunPhase.WAITING:
      return DagNodeRunPhase.WAITING;
    case TaskRunPhase.RUNNING:
      return DagNodeRunPhase.RUNNING;
    case TaskRunPhase.DONE:
      return DagNodeRunPhase.DONE;
    case TaskRunPhase.ERROR:
      return DagNodeRunPhase.ERROR;
    default:
      return DagNodeRunPhase.NONE;
  }
}

/**
 * Calculates how long a single task has been running, or how long it took to finish.
 *
 * @param status Run status of a single task.
 * @param snapshotTimeUnixNano Server-side capture time of the snapshot the status belongs to.
 * @returns Duration in milliseconds, or zero when the duration is not known yet.
 */
export function computeTaskRunDurationMs(
  status: TaskRunNodeStatus,
  snapshotTimeUnixNano: bigint,
): number {
  if (status.startTimeUnixNano <= 0n) {
    return 0;
  }
  const endTimeUnixNano =
    status.endTimeUnixNano > 0n ? status.endTimeUnixNano : snapshotTimeUnixNano;
  if (endTimeUnixNano <= status.startTimeUnixNano) {
    return 0;
  }
  return Number(endTimeUnixNano - status.startTimeUnixNano) / NANOS_PER_MILLI;
}

/**
 * Converts protobuf TaskDAGNode array to DagViewerNode models decorated with their run phase.
 */
export function convertToDagViewerNodes(
  nodes: readonly TaskDAGNode[],
  statusByTaskImplementationId: ReadonlyMap<string, TaskRunNodeStatus>,
  snapshotTimeUnixNano: bigint,
): DagViewerNode[] {
  return nodes.map((n) => {
    const status = statusByTaskImplementationId.get(n.taskImplementationId);
    return {
      id: n.taskImplementationId,
      referenceId: n.taskReferenceId,
      isFeature: n.isFeature,
      isFormTask: isFormTask(n.labels),
      topologicalOrder: n.topologicalOrder,
      priority: n.priority,
      labels: n.labels,
      outputType: n.outputType,
      providedTags: n.providedTags,
      runPhase: status
        ? convertToDagNodeRunPhase(status.phase)
        : DagNodeRunPhase.NONE,
      runDurationMs: status
        ? computeTaskRunDurationMs(status, snapshotTimeUnixNano)
        : 0,
    };
  });
}

/**
 * Converts protobuf TaskDAGEdge array to DagViewerEdge models.
 */
export function convertToDagViewerEdges(
  edges: readonly TaskDAGEdge[],
): DagViewerEdge[] {
  return edges.map((e, index) => ({
    id: `edge-${index}-${e.sourceImplementationId}->${e.destinationImplementationId}`,
    sourceId: e.sourceImplementationId,
    destinationId: e.destinationImplementationId,
    sourceReferenceId: e.sourceReferenceId,
    cardinality: e.cardinality,
    tag: e.tag,
    priority: e.priority,
    outputType: e.outputType,
  }));
}

/**
 * Calculates the wall clock time the entire inspection run has spent so far.
 *
 * While the run is in progress the elapsed time is measured against the server-side snapshot
 * time so the client clock skew does not distort the value.
 */
export function computeRunElapsedMs(
  snapshot: InspectionRunTaskGraphSnapshot,
): number {
  let earliestStartUnixNano = 0n;
  let latestEndUnixNano = 0n;
  for (const status of snapshot.nodeStatuses) {
    if (
      status.startTimeUnixNano > 0n &&
      (earliestStartUnixNano === 0n ||
        status.startTimeUnixNano < earliestStartUnixNano)
    ) {
      earliestStartUnixNano = status.startTimeUnixNano;
    }
    if (status.endTimeUnixNano > latestEndUnixNano) {
      latestEndUnixNano = status.endTimeUnixNano;
    }
  }
  if (earliestStartUnixNano === 0n) {
    return 0;
  }
  const endUnixNano = snapshot.isRunFinished
    ? latestEndUnixNano
    : snapshot.snapshotTimeUnixNano;
  if (endUnixNano <= earliestStartUnixNano) {
    return 0;
  }
  return Number(endUnixNano - earliestStartUnixNano) / NANOS_PER_MILLI;
}

/**
 * Counts the tasks that already reached a terminal phase, regardless of success or failure.
 */
export function countFinishedTasks(
  statuses: readonly TaskRunNodeStatus[],
): number {
  return statuses.filter(
    (status) =>
      status.phase === TaskRunPhase.DONE || status.phase === TaskRunPhase.ERROR,
  ).length;
}
