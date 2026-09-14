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
  ElementRef,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import {
  RegisteredTaskGroupInfo,
  RegisteredTaskInfo,
  TaskDependencyCardinality,
  TaskDependencyScope,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  getTaskDescription,
  isFormTask,
} from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';
import { TaskLabelEntry } from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { Step1TaskNeighborhoodPanelComponent } from 'src/app/pages/task-graph-debug/components/step1-task-neighborhood-panel.component';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Renders Step 1: All registered tasks grouped by TaskReferenceID.
 */
@Component({
  selector: 'khi-step1-registry-table',
  templateUrl: './step1-registry-table.component.html',
  styleUrls: ['./step1-registry-table.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIconModule,
    MatButtonModule,
    Step1TaskNeighborhoodPanelComponent,
    KHIIconRegistrationModule,
  ],
})
export class Step1RegistryTableComponent {
  /**
   * Export TaskDependencyCardinality enum to the template.
   */
  protected readonly TaskDependencyCardinality = TaskDependencyCardinality;

  /**
   * Helper function to detect form input tasks.
   */
  protected readonly isFormTask = isFormTask;

  /**
   * Helper function to extract human-readable task description.
   */
  protected readonly getTaskDescription = getTaskDescription;

  /**
   * All registered task groups by reference identifier.
   */
  readonly taskGroups = input<readonly RegisteredTaskGroupInfo[]>([]);

  /**
   * Text search query filter.
   */
  readonly searchQuery = signal<string>('');

  /**
   * Task groups filtered by search query matching reference ID or implementation ID.
   */
  readonly filteredTaskGroups = computed<readonly RegisteredTaskGroupInfo[]>(
    () => {
      const q = this.searchQuery().trim().toLowerCase();
      const groups = this.taskGroups();
      if (!q) {
        return groups;
      }
      return groups.filter((g) => {
        if (g.taskReferenceId.toLowerCase().includes(q)) {
          return true;
        }
        return g.tasks.some((task) =>
          task.taskImplementationId.toLowerCase().includes(q),
        );
      });
    },
  );

  /**
   * Total number of distinct task implementations registered.
   */
  readonly totalImplementationsCount = computed<number>(() =>
    this.taskGroups().reduce((acc, g) => acc + g.tasks.length, 0),
  );

  /**
   * Currently selected task implementation ID for neighborhood inspection, or null.
   */
  readonly selectedTaskId = signal<string | null>(null);

  /**
   * Linear navigation history stack of inspected task implementation IDs.
   */
  readonly history = signal<readonly string[]>([]);

  /**
   * Current active cursor index within the navigation history stack.
   */
  readonly historyIndex = signal<number>(-1);

  /**
   * Indicates whether backward history navigation is available.
   */
  readonly canGoBack = computed<boolean>(() => this.historyIndex() > 0);

  /**
   * Indicates whether forward history navigation is available.
   */
  readonly canGoForward = computed<boolean>(
    () =>
      this.historyIndex() >= 0 &&
      this.historyIndex() < this.history().length - 1,
  );

  private readonly elementRef = inject<ElementRef<HTMLElement>>(ElementRef);

  /**
   * Selects a task implementation, opens the neighborhood panel, and records navigation history.
   */
  selectTask(taskId: string): void {
    const currentTaskId =
      this.historyIndex() >= 0 ? this.history()[this.historyIndex()] : null;
    if (currentTaskId !== taskId) {
      const currentHistory = this.history().slice(0, this.historyIndex() + 1);
      const nextHistory = [...currentHistory, taskId];
      this.history.set(nextHistory);
      this.historyIndex.set(nextHistory.length - 1);
    }
    this.selectedTaskId.set(taskId);
    this.scrollToTask(taskId);
  }

  /**
   * Navigates backward to the previous task in the exploration history.
   */
  goBack(): void {
    if (!this.canGoBack()) {
      return;
    }
    const nextIndex = this.historyIndex() - 1;
    this.historyIndex.set(nextIndex);
    const taskId = this.history()[nextIndex];
    this.selectedTaskId.set(taskId);
    this.scrollToTask(taskId);
  }

  /**
   * Navigates forward to the next task in the exploration history.
   */
  goForward(): void {
    if (!this.canGoForward()) {
      return;
    }
    const nextIndex = this.historyIndex() + 1;
    this.historyIndex.set(nextIndex);
    const taskId = this.history()[nextIndex];
    this.selectedTaskId.set(taskId);
    this.scrollToTask(taskId);
  }

  /**
   * Closes the task neighborhood inspection panel.
   */
  closePanel(): void {
    this.selectedTaskId.set(null);
  }

  /**
   * Smoothly scrolls the registry list to bring the specified task card into view.
   */
  private scrollToTask(taskId: string): void {
    requestAnimationFrame(() => {
      const el = this.elementRef.nativeElement.querySelector(
        `#task-impl-${CSS.escape(taskId)}`,
      );
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    });
  }

  /**
   * Updates the search filter query.
   */
  onSearchInput(event: Event): void {
    const inputElement = event.target as HTMLInputElement;
    this.searchQuery.set(inputElement.value);
  }

  /**
   * Clears the search filter query.
   */
  clearSearch(): void {
    this.searchQuery.set('');
  }

  /**
   * Converts task metadata labels into an iterable array of key-value tuples.
   */
  getLabelEntries(task: RegisteredTaskInfo): readonly TaskLabelEntry[] {
    return Object.entries(task.labels).map(([key, value]) => ({ key, value }));
  }

  /**
   * Returns human-readable label for dependency scope.
   */
  getScopeLabel(scope: TaskDependencyScope): string {
    switch (scope) {
      case TaskDependencyScope.ACTIVE_FEATURES:
        return 'Active features';
      case TaskDependencyScope.ACTIVE_GRAPH:
        return 'Active graph';
      case TaskDependencyScope.ALL:
        return 'All candidates';
      default:
        return 'Unspecified';
    }
  }
}
