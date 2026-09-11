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

import { TaskFilterEvaluation } from 'src/app/generated/api/v1/inspection_task_graph_pb';

/**
 * Identifies the active diagnostic step tab in the task graph debug view.
 */
export enum TaskGraphDebugTab {
  /**
   * Step 1: All registered tasks grouped by reference identifier.
   */
  REGISTRY = 'REGISTRY',

  /**
   * Step 2: InspectionType compatibility filtering and priority deduplication.
   */
  INSPECTION_TYPE_FILTER = 'INSPECTION_TYPE_FILTER',

  /**
   * Step 3: Resolved DAG interactive execution graph.
   */
  DAG_VIEWER = 'DAG_VIEWER',
}

/**
 * Filter evaluation status classification for display in Step 2.
 */
export enum TaskFilterStatus {
  /**
   * Task matched inspection type and won priority competition.
   */
  SELECTED = 'SELECTED',

  /**
   * Task matched inspection type but lost priority competition to another implementation.
   */
  SUPERSEDED = 'SUPERSEDED',

  /**
   * Task did not match inspection type label selector.
   */
  INCOMPATIBLE = 'INCOMPATIBLE',
}

/**
 * Key-value pair entry for rendering task labels.
 */
export interface TaskLabelEntry {
  readonly key: string;
  readonly value: string;
}

/**
 * Event payload emitted when a feature toggle is switched in Step 2.
 */
export interface FeatureToggleChangeEvent {
  readonly taskImplementationId: string;
  readonly enabled: boolean;
}

/**
 * Enhanced filter evaluation item with computed display status.
 */
export interface TaskFilterEvaluationItem {
  /**
   * Original filter evaluation metadata from backend.
   */
  readonly evaluation: TaskFilterEvaluation;

  /**
   * Classified evaluation status.
   */
  readonly status: TaskFilterStatus;
}
