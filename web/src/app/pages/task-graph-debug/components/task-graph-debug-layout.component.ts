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
  input,
  output,
} from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import {
  FeatureToggleInfo,
  RegisteredInspectionTypeInfo,
  RegisteredTaskGroupInfo,
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';
import { DagCanvasComponent } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-canvas.component';
import { Step1RegistryTableComponent } from 'src/app/pages/task-graph-debug/components/step1-registry-table.component';
import { Step2TypeFilterComponent } from 'src/app/pages/task-graph-debug/components/step2-type-filter.component';
import {
  FeatureToggleChangeEvent,
  TaskGraphDebugTab,
} from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Layout dumb component orchestrating the 3-step task graph debug diagnostics.
 */
@Component({
  selector: 'khi-task-graph-debug-layout',
  templateUrl: './task-graph-debug-layout.component.html',
  styleUrls: ['./task-graph-debug-layout.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIconModule,
    MatProgressBarModule,
    Step1RegistryTableComponent,
    Step2TypeFilterComponent,
    DagCanvasComponent,
    KHIIconRegistrationModule,
  ],
})
export class TaskGraphDebugLayoutComponent {
  /**
   * Export TaskGraphDebugTab enum to the template.
   */
  protected readonly TaskGraphDebugTab = TaskGraphDebugTab;

  /**
   * Currently active diagnostics tab.
   */
  readonly activeTab = input<TaskGraphDebugTab>(TaskGraphDebugTab.REGISTRY);

  /**
   * All registered task groups.
   */
  readonly taskGroups = input<readonly RegisteredTaskGroupInfo[]>([]);

  /**
   * Available inspection types.
   */
  readonly inspectionTypes = input<readonly RegisteredInspectionTypeInfo[]>([]);

  /**
   * Selected inspection type ID.
   */
  readonly selectedInspectionTypeId = input<string>('');

  /**
   * Candidate filtering diagnostic evaluations.
   */
  readonly evaluations = input<readonly TaskFilterEvaluation[]>([]);

  /**
   * Toggleable features under current inspection type.
   */
  readonly availableFeatures = input<readonly FeatureToggleInfo[]>([]);

  /**
   * Nodes to render in the resolved DAG canvas.
   */
  readonly dagNodes = input<readonly DagViewerNode[]>([]);

  /**
   * Edges to render in the resolved DAG canvas.
   */
  readonly dagEdges = input<readonly DagViewerEdge[]>([]);

  /**
   * Whether graph resolution succeeded without error or cycle.
   */
  readonly isResolutionSuccess = input<boolean>(true);

  /**
   * Error message if graph resolution failed.
   */
  readonly resolutionErrorMessage = input<string>('');

  /**
   * Whether an async operation is loading.
   */
  readonly isLoading = input<boolean>(false);

  /**
   * Emitted when user selects a different step tab.
   */
  readonly tabChange = output<TaskGraphDebugTab>();

  /**
   * Emitted when user changes the selected inspection type.
   */
  readonly selectInspectionType = output<string>();

  /**
   * Emitted when user toggles a feature flag.
   */
  readonly featureToggleChange = output<FeatureToggleChangeEvent>();

  /**
   * Switches to the specified diagnostic tab.
   */
  onTabClick(tab: TaskGraphDebugTab): void {
    this.tabChange.emit(tab);
  }

  /**
   * Relays inspection type selection to parent.
   */
  onSelectInspectionType(typeId: string): void {
    this.selectInspectionType.emit(typeId);
  }

  /**
   * Relays feature toggle switch to parent.
   */
  onFeatureToggleChange(event: FeatureToggleChangeEvent): void {
    this.featureToggleChange.emit(event);
  }
}
