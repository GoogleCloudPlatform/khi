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
  signal,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatSelectModule } from '@angular/material/select';
import {
  FeatureToggleInfo,
  RegisteredInspectionTypeInfo,
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  FeatureToggleChangeEvent,
  TaskFilterEvaluationItem,
  TaskFilterStatus,
  TaskLabelEntry,
} from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Filter mode selection for Step 2 diagnostic evaluation list.
 */
export enum EvaluationStatusFilter {
  /**
   * Show all evaluations regardless of outcome.
   */
  ALL = 'ALL',

  /**
   * Show only selected tasks.
   */
  SELECTED_ONLY = 'SELECTED_ONLY',

  /**
   * Show only superseded tasks.
   */
  SUPERSEDED_ONLY = 'SUPERSEDED_ONLY',

  /**
   * Show only incompatible tasks.
   */
  INCOMPATIBLE_ONLY = 'INCOMPATIBLE_ONLY',
}

/**
 * Renders Step 2: InspectionType selection, feature toggles, and filter evaluation diagnostics.
 */
@Component({
  selector: 'khi-step2-type-filter',
  templateUrl: './step2-type-filter.component.html',
  styleUrls: ['./step2-type-filter.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatFormFieldModule,
    MatSelectModule,
    MatIconModule,
    MatButtonModule,
    MatCheckboxModule,
    KHIIconRegistrationModule,
  ],
})
export class Step2TypeFilterComponent {
  /**
   * Export TaskFilterStatus enum to the template.
   */
  protected readonly TaskFilterStatus = TaskFilterStatus;

  /**
   * Export EvaluationStatusFilter enum to the template.
   */
  protected readonly EvaluationStatusFilter = EvaluationStatusFilter;

  /**
   * Available inspection types.
   */
  readonly inspectionTypes = input<readonly RegisteredInspectionTypeInfo[]>([]);

  /**
   * Selected inspection type ID.
   */
  readonly selectedInspectionTypeId = input<string>('');

  /**
   * Available feature toggles under current inspection type.
   */
  readonly availableFeatures = input<readonly FeatureToggleInfo[]>([]);

  /**
   * Task filter evaluations for the current inspection type and features.
   */
  readonly evaluations = input<readonly TaskFilterEvaluation[]>([]);

  /**
   * Emitted when user chooses a different inspection type.
   */
  readonly selectInspectionType = output<string>();

  /**
   * Emitted when user toggles a feature flag.
   */
  readonly featureToggleChange = output<FeatureToggleChangeEvent>();

  /**
   * Active outcome status filter.
   */
  readonly statusFilter = signal<EvaluationStatusFilter>(
    EvaluationStatusFilter.ALL,
  );

  /**
   * Search query to filter evaluation rows.
   */
  readonly searchQuery = signal<string>('');

  /**
   * Currently active inspection type metadata object.
   */
  readonly currentInspectionType = computed<
    RegisteredInspectionTypeInfo | undefined
  >(() =>
    this.inspectionTypes().find(
      (type) => type.id === this.selectedInspectionTypeId(),
    ),
  );

  /**
   * Classified evaluation items with status enum.
   */
  readonly classifiedEvaluations = computed<
    readonly TaskFilterEvaluationItem[]
  >(() => {
    return this.evaluations().map((evaluation) => {
      let status = TaskFilterStatus.INCOMPATIBLE;
      if (evaluation.isCompatible) {
        status = evaluation.isSelected
          ? TaskFilterStatus.SELECTED
          : TaskFilterStatus.SUPERSEDED;
      }
      return {
        evaluation,
        status,
      };
    });
  });

  /**
   * Filtered evaluation items matching active status filter and search query.
   */
  readonly filteredEvaluations = computed<readonly TaskFilterEvaluationItem[]>(
    () => {
      const statusFilterVal = this.statusFilter();
      const q = this.searchQuery().trim().toLowerCase();

      return this.classifiedEvaluations().filter((item) => {
        if (
          statusFilterVal === EvaluationStatusFilter.SELECTED_ONLY &&
          item.status !== TaskFilterStatus.SELECTED
        ) {
          return false;
        }
        if (
          statusFilterVal === EvaluationStatusFilter.SUPERSEDED_ONLY &&
          item.status !== TaskFilterStatus.SUPERSEDED
        ) {
          return false;
        }
        if (
          statusFilterVal === EvaluationStatusFilter.INCOMPATIBLE_ONLY &&
          item.status !== TaskFilterStatus.INCOMPATIBLE
        ) {
          return false;
        }

        if (q) {
          const evalItem = item.evaluation;
          const matchesRef = evalItem.taskReferenceId.toLowerCase().includes(q);
          const matchesImpl = evalItem.taskImplementationId
            .toLowerCase()
            .includes(q);
          const matchesReason = evalItem.matchReason.toLowerCase().includes(q);
          if (!matchesRef && !matchesImpl && !matchesReason) {
            return false;
          }
        }
        return true;
      });
    },
  );

  /**
   * Number of selected winning tasks.
   */
  readonly selectedCount = computed<number>(
    () =>
      this.classifiedEvaluations().filter(
        (item) => item.status === TaskFilterStatus.SELECTED,
      ).length,
  );

  /**
   * Number of superseded tasks that lost priority competition.
   */
  readonly supersededCount = computed<number>(
    () =>
      this.classifiedEvaluations().filter(
        (item) => item.status === TaskFilterStatus.SUPERSEDED,
      ).length,
  );

  /**
   * Number of incompatible tasks that failed label matching.
   */
  readonly incompatibleCount = computed<number>(
    () =>
      this.classifiedEvaluations().filter(
        (item) => item.status === TaskFilterStatus.INCOMPATIBLE,
      ).length,
  );

  /**
   * Converts inspection type labels into an iterable array of key-value tuples.
   */
  getInspectionTypeLabels(
    itype: RegisteredInspectionTypeInfo,
  ): readonly TaskLabelEntry[] {
    return Object.entries(itype.labels).map(([key, value]) => ({ key, value }));
  }

  /**
   * Handles user changing the inspection type dropdown.
   */
  onInspectionTypeSelect(typeId: string): void {
    this.selectInspectionType.emit(typeId);
  }

  /**
   * Handles user toggling a feature checkbox.
   */
  onFeatureToggle(taskImplementationId: string, enabled: boolean): void {
    this.featureToggleChange.emit({ taskImplementationId, enabled });
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
   * Sets the active evaluation status filter.
   */
  setStatusFilter(filter: EvaluationStatusFilter): void {
    this.statusFilter.set(filter);
  }
}
