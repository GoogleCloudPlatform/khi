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
  OnInit,
  inject,
  signal,
} from '@angular/core';
import {
  FeatureToggleInfo,
  RegisteredInspectionTypeInfo,
  RegisteredTaskGroupInfo,
  TaskDAGEdge,
  TaskDAGNode,
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagViewerEdge,
  DagViewerNode,
  isFormTask,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';
import { TaskGraphDebugLayoutComponent } from 'src/app/pages/task-graph-debug/components/task-graph-debug-layout.component';
import {
  FeatureToggleChangeEvent,
  TaskGraphDebugTab,
} from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';

/**
 * Converts protobuf TaskDAGNode array to DagViewerNode models.
 */
function convertToDagViewerNodes(
  nodes: readonly TaskDAGNode[],
): DagViewerNode[] {
  return nodes.map((n) => ({
    id: n.taskImplementationId,
    referenceId: n.taskReferenceId,
    isFeature: n.isFeature,
    isFormTask: isFormTask(n.labels),
    topologicalOrder: n.topologicalOrder,
    priority: n.priority,
    labels: n.labels,
    outputType: n.outputType,
    providedTags: n.providedTags,
  }));
}

/**
 * Converts protobuf TaskDAGEdge array to DagViewerEdge models.
 */
function convertToDagViewerEdges(
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
 * Smart container component managing data fetching, state, and RPC resolution for Task Graph Debugger.
 */
@Component({
  selector: 'khi-task-graph-debug-smart',
  templateUrl: './task-graph-debug-smart.component.html',
  styleUrls: ['./task-graph-debug-smart.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [TaskGraphDebugLayoutComponent],
})
export class TaskGraphDebugSmartComponent implements OnInit {
  private readonly connectClient = inject(ConnectClientService);
  private resolveRequestId = 0;

  /**
   * Currently active diagnostic step tab.
   */
  readonly activeTab = signal<TaskGraphDebugTab>(TaskGraphDebugTab.REGISTRY);

  /**
   * All task groups registered in the server.
   */
  readonly taskGroups = signal<readonly RegisteredTaskGroupInfo[]>([]);

  /**
   * All available inspection types.
   */
  readonly inspectionTypes = signal<readonly RegisteredInspectionTypeInfo[]>(
    [],
  );

  /**
   * Currently selected inspection type identifier.
   */
  readonly selectedInspectionTypeId = signal<string>('');

  /**
   * Feature toggle overrides requested by user.
   */
  readonly featureOverrides = signal<Record<string, boolean>>({});

  /**
   * Step 2 filter compatibility evaluations.
   */
  readonly evaluations = signal<readonly TaskFilterEvaluation[]>([]);

  /**
   * Toggleable features under the current inspection type.
   */
  readonly availableFeatures = signal<readonly FeatureToggleInfo[]>([]);

  /**
   * Resolved execution DAG nodes.
   */
  readonly dagNodes = signal<readonly DagViewerNode[]>([]);

  /**
   * Resolved execution DAG directed edges.
   */
  readonly dagEdges = signal<readonly DagViewerEdge[]>([]);

  /**
   * Whether graph resolution succeeded.
   */
  readonly isResolutionSuccess = signal<boolean>(true);

  /**
   * Error message if resolution failed.
   */
  readonly resolutionErrorMessage = signal<string>('');

  /**
   * Whether an async network request is in flight.
   */
  readonly isLoading = signal<boolean>(false);

  /**
   * Initializes component by fetching task registry.
   */
  ngOnInit(): void {
    void this.fetchRegistry();
  }

  /**
   * Fetches task registry and initializes graph resolution.
   */
  async fetchRegistry(): Promise<void> {
    this.isLoading.set(true);
    try {
      const resp =
        await this.connectClient.inspectionTaskGraphClient.getInspectionTaskRegistry(
          {},
        );
      this.taskGroups.set(resp.taskGroups);
      this.inspectionTypes.set(resp.inspectionTypes);

      if (resp.inspectionTypes.length > 0 && !this.selectedInspectionTypeId()) {
        this.selectedInspectionTypeId.set(resp.inspectionTypes[0].id);
      }

      await this.resolveGraph();
    } catch (err) {
      this.isResolutionSuccess.set(false);
      this.resolutionErrorMessage.set(
        err instanceof Error ? err.message : String(err),
      );
    } finally {
      this.isLoading.set(false);
    }
  }

  /**
   * Resolves the task execution DAG with current inspection type and feature overrides.
   */
  async resolveGraph(): Promise<void> {
    const typeId = this.selectedInspectionTypeId();
    if (!typeId) {
      return;
    }

    const currentRequestId = ++this.resolveRequestId;
    this.isLoading.set(true);
    try {
      const resp =
        await this.connectClient.inspectionTaskGraphClient.resolveInspectionTaskGraph(
          {
            inspectionTypeId: typeId,
            featureOverrides: this.featureOverrides(),
          },
        );

      if (currentRequestId !== this.resolveRequestId) {
        return;
      }

      this.evaluations.set(resp.filteringEvaluations);
      this.availableFeatures.set(resp.availableFeatures);

      if (resp.dag) {
        this.isResolutionSuccess.set(resp.dag.isSuccess);
        this.resolutionErrorMessage.set(resp.dag.errorMessage);
        this.dagNodes.set(convertToDagViewerNodes(resp.dag.nodes));
        this.dagEdges.set(convertToDagViewerEdges(resp.dag.edges));
      } else {
        this.isResolutionSuccess.set(false);
        this.resolutionErrorMessage.set('No DAG topology returned from server');
        this.dagNodes.set([]);
        this.dagEdges.set([]);
      }
    } catch (err) {
      if (currentRequestId !== this.resolveRequestId) {
        return;
      }
      this.isResolutionSuccess.set(false);
      this.resolutionErrorMessage.set(
        err instanceof Error ? err.message : String(err),
      );
    } finally {
      if (currentRequestId === this.resolveRequestId) {
        this.isLoading.set(false);
      }
    }
  }

  /**
   * Handles tab navigation.
   */
  onTabChange(tab: TaskGraphDebugTab): void {
    this.activeTab.set(tab);
  }

  /**
   * Handles target inspection type selection change.
   */
  onSelectInspectionType(typeId: string): void {
    this.selectedInspectionTypeId.set(typeId);
    this.featureOverrides.set({});
    void this.resolveGraph();
  }

  /**
   * Handles feature flag toggle change.
   */
  onFeatureToggleChange(event: FeatureToggleChangeEvent): void {
    this.featureOverrides.update((prev) => ({
      ...prev,
      [event.taskImplementationId]: event.enabled,
    }));
    void this.resolveGraph();
  }
}
