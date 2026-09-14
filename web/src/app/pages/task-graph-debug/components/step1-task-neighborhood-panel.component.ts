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
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  output,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import {
  RegisteredTaskGroupInfo,
  RegisteredTaskInfo,
  TaskDependencyCardinality,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { DagCanvasComponent } from 'src/app/shared/components/dag-viewer/dag-canvas.component';
import {
  DagNodeRunPhase,
  DagViewerEdge,
  DagViewerNode,
  getTaskDescription,
  isFormTask,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Structure containing 3-layer neighborhood nodes, directed edges, and metadata for a focal task.
 */
export interface TaskNeighborhoodGraph {
  /**
   * Focal center task metadata, or null if unselected/unregistered.
   */
  readonly centerTask: RegisteredTaskInfo | null;

  /**
   * Nodes representing upstream parents, the focal task, and downstream children.
   */
  readonly nodes: readonly DagViewerNode[];

  /**
   * Directed dependency edges connecting the 3 layers.
   */
  readonly edges: readonly DagViewerEdge[];

  /**
   * Total number of upstream parent candidates found.
   */
  readonly parentCount: number;

  /**
   * Total number of downstream consumer candidates found.
   */
  readonly childCount: number;
}

/**
 * Converts a RegisteredTaskInfo into a DagViewerNode representation with an assigned topological layer.
 */
function toDagViewerNode(
  task: RegisteredTaskInfo,
  topologicalOrder: number,
): DagViewerNode {
  return {
    id: task.taskImplementationId,
    referenceId: task.taskReferenceId,
    isFeature: task.isFeature,
    isFormTask: isFormTask(task.labels),
    topologicalOrder,
    priority: task.priority,
    labels: task.labels,
    outputType: task.outputType,
    providedTags: task.providedTags,
    runPhase: DagNodeRunPhase.NONE,
    runDurationMs: 0,
  };
}

/**
 * Indexes task metadata by implementation ID and reference ID for efficient lookup.
 */
function indexRegisteredTasks(taskGroups: readonly RegisteredTaskGroupInfo[]): {
  readonly taskMap: Map<string, RegisteredTaskInfo>;
  readonly refToTasksMap: Map<string, RegisteredTaskInfo[]>;
} {
  const taskMap = new Map<string, RegisteredTaskInfo>();
  const refToTasksMap = new Map<string, RegisteredTaskInfo[]>();

  for (const group of taskGroups) {
    for (const task of group.tasks) {
      taskMap.set(task.taskImplementationId, task);
      const list = refToTasksMap.get(task.taskReferenceId) ?? [];
      list.push(task);
      refToTasksMap.set(task.taskReferenceId, list);
    }
  }

  return { taskMap, refToTasksMap };
}

/**
 * Extracts fan-in aggregation tag names provided by the specified task.
 */
function extractProvidedTags(task: RegisteredTaskInfo): Set<string> {
  const providedTags = new Set<string>();
  for (const item of task.providedTags) {
    if (item.tag) {
      providedTags.add(item.tag);
    }
  }
  return providedTags;
}

/**
 * Finds upstream parent task candidates and constructs connecting dependency edges.
 */
function findParentCandidates(
  centerTask: RegisteredTaskInfo,
  taskMap: Map<string, RegisteredTaskInfo>,
  refToTasksMap: Map<string, RegisteredTaskInfo[]>,
): {
  readonly parentTasks: Map<string, RegisteredTaskInfo>;
  readonly parentEdges: DagViewerEdge[];
} {
  const parentTasks = new Map<string, RegisteredTaskInfo>();
  const parentEdges: DagViewerEdge[] = [];

  for (const dep of centerTask.dependencies) {
    if (dep.cardinality === TaskDependencyCardinality.FAN_IN) {
      const targetTag = dep.targetTag;
      for (const candidate of taskMap.values()) {
        if (candidate.providedTags.some((p) => p.tag === targetTag)) {
          parentTasks.set(candidate.taskImplementationId, candidate);
          parentEdges.push({
            id: `${candidate.taskImplementationId}->${centerTask.taskImplementationId}:fan_in:${targetTag}`,
            sourceId: candidate.taskImplementationId,
            destinationId: centerTask.taskImplementationId,
            sourceReferenceId: candidate.taskReferenceId,
            cardinality: TaskDependencyCardinality.FAN_IN,
            tag: targetTag,
            priority: 0,
            outputType: dep.outputType || candidate.outputType,
          });
        }
      }
    } else {
      const candidates = refToTasksMap.get(dep.targetReferenceId) ?? [];
      for (const candidate of candidates) {
        parentTasks.set(candidate.taskImplementationId, candidate);
        parentEdges.push({
          id: `${candidate.taskImplementationId}->${centerTask.taskImplementationId}:p2p:${dep.targetReferenceId}`,
          sourceId: candidate.taskImplementationId,
          destinationId: centerTask.taskImplementationId,
          sourceReferenceId: candidate.taskReferenceId,
          cardinality: dep.cardinality,
          tag: '',
          priority: 0,
          outputType: candidate.outputType || dep.outputType,
        });
      }
    }
  }

  return { parentTasks, parentEdges };
}

/**
 * Finds downstream consumer task candidates and constructs connecting dependency edges.
 */
function findChildCandidates(
  centerTask: RegisteredTaskInfo,
  taskMap: Map<string, RegisteredTaskInfo>,
  providedTags: Set<string>,
): {
  readonly childTasks: Map<string, RegisteredTaskInfo>;
  readonly childEdges: DagViewerEdge[];
} {
  const childTasks = new Map<string, RegisteredTaskInfo>();
  const childEdges: DagViewerEdge[] = [];

  for (const candidate of taskMap.values()) {
    if (candidate.taskImplementationId === centerTask.taskImplementationId) {
      continue;
    }
    for (const dep of candidate.dependencies) {
      if (dep.cardinality === TaskDependencyCardinality.FAN_IN) {
        if (providedTags.has(dep.targetTag)) {
          childTasks.set(candidate.taskImplementationId, candidate);
          childEdges.push({
            id: `${centerTask.taskImplementationId}->${candidate.taskImplementationId}:fan_in:${dep.targetTag}`,
            sourceId: centerTask.taskImplementationId,
            destinationId: candidate.taskImplementationId,
            sourceReferenceId: centerTask.taskReferenceId,
            cardinality: TaskDependencyCardinality.FAN_IN,
            tag: dep.targetTag,
            priority: 0,
            outputType: dep.outputType || centerTask.outputType,
          });
        }
      } else {
        if (dep.targetReferenceId === centerTask.taskReferenceId) {
          childTasks.set(candidate.taskImplementationId, candidate);
          childEdges.push({
            id: `${centerTask.taskImplementationId}->${candidate.taskImplementationId}:p2p:${dep.targetReferenceId}`,
            sourceId: centerTask.taskImplementationId,
            destinationId: candidate.taskImplementationId,
            sourceReferenceId: centerTask.taskReferenceId,
            cardinality: dep.cardinality,
            tag: '',
            priority: 0,
            outputType: centerTask.outputType || dep.outputType,
          });
        }
      }
    }
  }

  return { childTasks, childEdges };
}

/**
 * Computes a 3-layer local neighborhood (upstream parents, focal task, downstream children)
 * from all registered task metadata without server roundtrips.
 */
export function computeTaskNeighborhoodGraph(
  taskGroups: readonly RegisteredTaskGroupInfo[],
  centerTaskId: string | null,
): TaskNeighborhoodGraph {
  if (!centerTaskId) {
    return {
      centerTask: null,
      nodes: [],
      edges: [],
      parentCount: 0,
      childCount: 0,
    };
  }

  const { taskMap, refToTasksMap } = indexRegisteredTasks(taskGroups);
  const centerTask = taskMap.get(centerTaskId);
  if (!centerTask) {
    return {
      centerTask: null,
      nodes: [],
      edges: [],
      parentCount: 0,
      childCount: 0,
    };
  }

  const providedTags = extractProvidedTags(centerTask);
  const { parentTasks, parentEdges } = findParentCandidates(
    centerTask,
    taskMap,
    refToTasksMap,
  );
  const { childTasks, childEdges } = findChildCandidates(
    centerTask,
    taskMap,
    providedTags,
  );

  // Assemble distinct nodes: parents (0), focal task (1), children (2)
  const nodeMap = new Map<string, DagViewerNode>();
  for (const parent of parentTasks.values()) {
    nodeMap.set(parent.taskImplementationId, toDagViewerNode(parent, 0));
  }
  nodeMap.set(centerTask.taskImplementationId, toDagViewerNode(centerTask, 1));
  for (const child of childTasks.values()) {
    if (!nodeMap.has(child.taskImplementationId)) {
      nodeMap.set(child.taskImplementationId, toDagViewerNode(child, 2));
    }
  }

  // Deduplicate edges
  const edgeMap = new Map<string, DagViewerEdge>();
  for (const edge of [...parentEdges, ...childEdges]) {
    edgeMap.set(edge.id, edge);
  }

  return {
    centerTask,
    nodes: Array.from(nodeMap.values()),
    edges: Array.from(edgeMap.values()),
    parentCount: parentTasks.size,
    childCount: childTasks.size,
  };
}

/**
 * Slide-over side panel displaying the 3-layer dependency neighborhood of a selected task.
 */
@Component({
  selector: 'khi-step1-task-neighborhood-panel',
  templateUrl: './step1-task-neighborhood-panel.component.html',
  styleUrls: ['./step1-task-neighborhood-panel.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIconModule,
    MatButtonModule,
    DagCanvasComponent,
    KHIIconRegistrationModule,
  ],
})
export class Step1TaskNeighborhoodPanelComponent {
  /**
   * All registered task groups.
   */
  readonly taskGroups = input<readonly RegisteredTaskGroupInfo[]>([]);

  /**
   * Implementation ID of the focal center task.
   */
  readonly centerTaskId = input<string | null>(null);

  /**
   * Whether history navigation can move backward.
   */
  readonly canGoBack = input<boolean>(false);

  /**
   * Whether history navigation can move forward.
   */
  readonly canGoForward = input<boolean>(false);

  /**
   * Emitted when user requests to close the side panel.
   */
  readonly closePanel = output<void>();

  /**
   * Emitted when user navigates backward in history.
   */
  readonly goBack = output<void>();

  /**
   * Emitted when user navigates forward in history.
   */
  readonly goForward = output<void>();

  /**
   * Emitted when user selects a neighbor node to become the new center task.
   */
  readonly selectTask = output<string>();

  /**
   * Computed 3-layer neighborhood graph for the focal task.
   */
  readonly neighborhoodGraph = computed<TaskNeighborhoodGraph>(() =>
    computeTaskNeighborhoodGraph(this.taskGroups(), this.centerTaskId()),
  );

  /**
   * Positionable nodes for DAG canvas rendering.
   */
  readonly nodes = computed<readonly DagViewerNode[]>(
    () => this.neighborhoodGraph().nodes,
  );

  /**
   * Directed edges for DAG canvas rendering.
   */
  readonly edges = computed<readonly DagViewerEdge[]>(
    () => this.neighborhoodGraph().edges,
  );

  /**
   * Focal center task metadata.
   */
  readonly centerTask = computed<RegisteredTaskInfo | null>(
    () => this.neighborhoodGraph().centerTask,
  );

  /**
   * Human-readable description of the focal center task.
   */
  readonly centerTaskDescription = computed<string>(() => {
    const center = this.centerTask();
    if (!center) {
      return '';
    }
    return getTaskDescription(center.labels);
  });

  /**
   * Handles node click within the neighborhood canvas.
   */
  onNodeClick(nodeId: string): void {
    if (nodeId !== this.centerTaskId()) {
      this.selectTask.emit(nodeId);
    }
  }

  /**
   * Handles backward navigation button click.
   */
  onBackClick(): void {
    this.goBack.emit();
  }

  /**
   * Handles forward navigation button click.
   */
  onForwardClick(): void {
    this.goForward.emit();
  }

  /**
   * Handles close panel button click.
   */
  onCloseClick(): void {
    this.closePanel.emit();
  }
}
