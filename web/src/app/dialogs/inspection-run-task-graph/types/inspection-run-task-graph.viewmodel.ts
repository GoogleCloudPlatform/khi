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
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

/**
 * Data handed to the inspection run task graph dialog when it is opened.
 */
export interface InspectionRunTaskGraphDialogData {
  /**
   * Identifier of the inspection to observe.
   */
  readonly inspectionId: string;

  /**
   * Display name of the inspection shown in the dialog header.
   */
  readonly inspectionName: string;
}

/**
 * Rendering state of the inspection run task graph dialog.
 */
export interface InspectionRunTaskGraphViewModel {
  /**
   * Display name of the observed inspection.
   */
  readonly inspectionName: string;

  /**
   * Task nodes decorated with their current run phase.
   */
  readonly nodes: readonly DagViewerNode[];

  /**
   * Dependency edges connecting the task nodes.
   */
  readonly edges: readonly DagViewerEdge[];

  /**
   * Number of tasks that already reached a terminal phase.
   */
  readonly finishedTaskCount: number;

  /**
   * Number of tasks in the graph.
   */
  readonly totalTaskCount: number;

  /**
   * Wall clock time spent by the run so far, in milliseconds.
   */
  readonly elapsedMs: number;

  /**
   * Whether the run already reached its terminal state.
   */
  readonly isRunFinished: boolean;

  /**
   * Failure description of the observation stream itself, or an empty string while it is healthy.
   *
   * This is unrelated to a task failing during the run, which the node colors convey instead.
   */
  readonly watchErrorMessage: string;
}
