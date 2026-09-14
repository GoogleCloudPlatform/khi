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
  ProvidedTagInfo,
  TaskDependencyCardinality,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';

/**
 * Label key identifying form input tasks in KHI.
 */
export const FORM_TASK_LABEL_KEY = 'khi.google.com/inspection/is-form-task';

/**
 * Item representing a tag provided by a task with its output type.
 */
export interface ProvidedTagItem {
  /**
   * Tag identifier string.
   */
  readonly tag: string;

  /**
   * Go output type string of the tag data.
   */
  readonly outputType?: string;
}

/**
 * Checks whether a task is a user input form task based on its labels.
 */
export function isFormTask(labels: Readonly<Record<string, string>>): boolean {
  return (
    labels[FORM_TASK_LABEL_KEY] === 'true' ||
    'khi.google.com/inspection/form-field-label' in labels
  );
}

/**
 * Label key identifying the human-readable description of a task in KHI.
 */
export const TASK_DESCRIPTION_LABEL_KEY = 'khi.google.com/task-description';

/**
 * Returns the human-readable description of a task from its labels without fallback.
 */
export function getTaskDescription(
  labels?: Readonly<Record<string, string>> | null,
): string {
  if (!labels) {
    return '';
  }
  return labels[TASK_DESCRIPTION_LABEL_KEY] ?? '';
}

/**
 * Execution phase of a task node rendered in the DAG viewer.
 */
export enum DagNodeRunPhase {
  /**
   * The node belongs to a graph that is not tied to any run, so no decoration is applied.
   */
  NONE = 'NONE',

  /**
   * The task is registered in the run but has not started yet.
   */
  WAITING = 'WAITING',

  /**
   * The task is currently running.
   */
  RUNNING = 'RUNNING',

  /**
   * The task finished successfully.
   */
  DONE = 'DONE',

  /**
   * The task finished with an error.
   */
  ERROR = 'ERROR',
}

/**
 * Represents a node within the DAG viewer.
 */
export interface DagViewerNode {
  /**
   * Unique implementation identifier of the task.
   */
  readonly id: string;

  /**
   * Reference identifier representing the contract/role of the task.
   */
  readonly referenceId: string;

  /**
   * Indicates whether this task is an optional feature.
   */
  readonly isFeature: boolean;

  /**
   * Indicates whether this task is a form input task.
   */
  readonly isFormTask: boolean;

  /**
   * Order of execution in the resolved topological sequence.
   */
  readonly topologicalOrder: number;

  /**
   * Selection priority weight for the implementation.
   */
  readonly priority: number;

  /**
   * Metadata labels attached to this task.
   */
  readonly labels: Readonly<Record<string, string>>;

  /**
   * Output type string of the task result.
   */
  readonly outputType: string;

  /**
   * Tags provided by this task.
   */
  readonly providedTags: readonly ProvidedTagInfo[];

  /**
   * Execution phase used to decorate the node border.
   */
  readonly runPhase: DagNodeRunPhase;

  /**
   * Elapsed time for a running task or total duration for a finished task, in milliseconds.
   * Zero means the duration is unknown and must not be rendered.
   */
  readonly runDurationMs: number;
}

/**
 * Represents a directed dependency edge between two tasks in the DAG viewer.
 */
export interface DagViewerEdge {
  /**
   * Unique identifier of this edge.
   */
  readonly id: string;

  /**
   * Implementation ID of the source task providing data or dependency.
   */
  readonly sourceId: string;

  /**
   * Implementation ID of the destination task consuming the dependency.
   */
  readonly destinationId: string;

  /**
   * Reference ID of the source task.
   */
  readonly sourceReferenceId: string;

  /**
   * Dependency cardinality (point-to-point or fan-in).
   */
  readonly cardinality: TaskDependencyCardinality;

  /**
   * Fan-in aggregation tag name, if applicable.
   */
  readonly tag: string;

  /**
   * Dependency edge priority.
   */
  readonly priority: number;

  /**
   * Output type string carried by this dependency edge.
   */
  readonly outputType: string;
}

/**
 * Node with 2D coordinates and bounding box dimensions computed by DAG layout.
 */
export interface DagPositionedNode extends DagViewerNode {
  /**
   * X coordinate of the top-left corner in SVG canvas space.
   */
  readonly x: number;

  /**
   * Y coordinate of the top-left corner in SVG canvas space.
   */
  readonly y: number;

  /**
   * Width of the node card.
   */
  readonly width: number;

  /**
   * Height of the node card.
   */
  readonly height: number;

  /**
   * Topological layer index (column) assigned to this node.
   */
  readonly layer: number;
}

/**
 * Edge with SVG path definition and anchor coordinates.
 */
export interface DagPositionedEdge extends DagViewerEdge {
  /**
   * SVG path data string (d attribute) for cubic Bezier curve.
   */
  readonly pathD: string;

  /**
   * X coordinate of the starting point (source node output).
   */
  readonly startX: number;

  /**
   * Y coordinate of the starting point (source node output).
   */
  readonly startY: number;

  /**
   * X coordinate of the end point (destination node input).
   */
  readonly endX: number;

  /**
   * Y coordinate of the end point (destination node input).
   */
  readonly endY: number;

  /**
   * X coordinate for rendering the edge label or chip.
   */
  readonly labelX: number;

  /**
   * Y coordinate for rendering the edge label or chip.
   */
  readonly labelY: number;
}

/**
 * Complete layout result containing positioned nodes, edges, and canvas bounds.
 */
export interface DagLayoutResult {
  /**
   * Positioned nodes to render.
   */
  readonly nodes: readonly DagPositionedNode[];

  /**
   * Positioned edges to render.
   */
  readonly edges: readonly DagPositionedEdge[];

  /**
   * Total width of the canvas layout.
   */
  readonly width: number;

  /**
   * Total height of the canvas layout.
   */
  readonly height: number;
}
