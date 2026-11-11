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
  InspectionDryRunResponse,
  InspectionFeature,
  InspectionMetadataInDryrun,
  InspectionType,
} from 'src/app/common/schema/api-types';
import {
  GroupParameterFormField,
  ParameterFormField,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import {
  EstimatedCountPreset,
  InspectionMetadataQuery,
} from 'src/app/common/schema/metadata-types';
import {
  ParameterStepViewModel,
  TotalEstimatedLogsSeverity,
  TotalEstimatedLogsSummary,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import { TaskGraphDebugTab } from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { ApiPathUtil } from 'src/app/services/api/api-path-util';

function resolveLogSeverity(count: number): TotalEstimatedLogsSeverity {
  if (count >= 5_000_000) {
    return TotalEstimatedLogsSeverity.Danger;
  }
  if (count >= 1_000_000) {
    return TotalEstimatedLogsSeverity.Warning;
  }
  return TotalEstimatedLogsSeverity.Normal;
}

function isQueryUnestimated(query: InspectionMetadataQuery): boolean {
  const hasNoCount = query.estimatedCount === undefined || !!query.pending;
  const isNotPreset =
    !query.estimatedCountPreset ||
    query.estimatedCountPreset === EstimatedCountPreset.None;
  return hasNoCount && !query.incomplete && isNotPreset;
}

function buildIncompleteSummary(
  knownCount: number,
  isEstimating: boolean,
  severity: TotalEstimatedLogsSeverity,
): TotalEstimatedLogsSummary {
  if (knownCount > 0) {
    const formattedCount = knownCount.toLocaleString('en-US');
    return {
      knownCount,
      isComplete: false,
      isEstimating,
      isIncomplete: true,
      displayText: `>${formattedCount} logs estimated (some parameters incomplete)`,
      severity,
    };
  }
  return {
    knownCount: 0,
    isComplete: false,
    isEstimating,
    isIncomplete: true,
    displayText: 'Incomplete parameters',
    severity: TotalEstimatedLogsSeverity.Normal,
  };
}

function buildEstimatingSummary(
  knownCount: number,
  severity: TotalEstimatedLogsSeverity,
): TotalEstimatedLogsSummary {
  if (knownCount > 0) {
    const formattedCount = knownCount.toLocaleString('en-US');
    return {
      knownCount,
      isComplete: false,
      isEstimating: true,
      isIncomplete: false,
      displayText: `>${formattedCount} logs estimated so far`,
      severity,
    };
  }
  return {
    knownCount: 0,
    isComplete: false,
    isEstimating: true,
    isIncomplete: false,
    displayText: 'Estimating total logs...',
    severity: TotalEstimatedLogsSeverity.Normal,
  };
}

/**
 * Computes the aggregate estimated log count summary across all queries.
 *
 * @param queries The queries to evaluate.
 * @returns The estimated logs summary, or undefined if no queries are provided.
 */
export function computeTotalEstimatedLogs(
  queries?: readonly InspectionMetadataQuery[],
): TotalEstimatedLogsSummary | undefined {
  if (!queries || queries.length === 0) {
    return undefined;
  }
  const hasIncomplete = queries.some((q) => q.incomplete);
  const hasUnestimated = queries.some(isQueryUnestimated);
  const hasPreset = queries.some(
    (q) =>
      q.estimatedCountPreset &&
      q.estimatedCountPreset !== EstimatedCountPreset.None,
  );
  const knownCount = queries
    .filter((q) => q.estimatedCount !== undefined && !q.pending)
    .reduce((sum, q) => sum + (q.estimatedCount ?? 0), 0);
  const severity = resolveLogSeverity(knownCount);

  if (hasIncomplete) {
    return buildIncompleteSummary(knownCount, hasUnestimated, severity);
  }
  if (hasUnestimated) {
    return buildEstimatingSummary(knownCount, severity);
  }
  if (knownCount === 0 && hasPreset) {
    return {
      knownCount: 0,
      isComplete: true,
      isEstimating: false,
      isIncomplete: false,
      displayText: 'Few total logs estimated',
      severity,
    };
  }
  const formattedCount = knownCount.toLocaleString('en-US');
  return {
    knownCount,
    isComplete: true,
    isEstimating: false,
    isIncomplete: false,
    displayText: `~${formattedCount} total logs estimated`,
    severity,
  };
}

type LeafParameterFormField = Exclude<
  ParameterFormField,
  GroupParameterFormField
>;

function isInputField(field: LeafParameterFormField): boolean {
  return field.type !== ParameterInputType.Callout;
}

function someLeafField(
  fields: readonly ParameterFormField[],
  predicate: (field: LeafParameterFormField) => boolean,
): boolean {
  for (const field of fields) {
    if (field.type === ParameterInputType.Group) {
      if (someLeafField(field.children, predicate)) {
        return true;
      }
    } else if (predicate(field)) {
      return true;
    }
  }
  return false;
}

function countLeafFields(
  fields: readonly ParameterFormField[],
  predicate: (field: LeafParameterFormField) => boolean,
): number {
  let count = 0;
  for (const field of fields) {
    if (field.type === ParameterInputType.Group) {
      count += countLeafFields(field.children, predicate);
    } else if (predicate(field)) {
      count++;
    }
  }
  return count;
}

/**
 * Recursively checks if any parameter field in the form has an error hint.
 *
 * @param fields The list of form fields to check.
 * @returns True if at least one field has an error hint.
 */
export function hasFormErrors(fields: readonly ParameterFormField[]): boolean {
  return someLeafField(
    fields,
    (field) => field.hintType === ParameterHintType.Error,
  );
}

/**
 * Checks if a dryrun response contains validation errors or incomplete queries.
 *
 * @param response The dryrun response to validate.
 * @returns True if the dryrun result has errors or incomplete parameters.
 */
export function hasDryRunErrors(response: InspectionDryRunResponse): boolean {
  if (hasFormErrors(response.metadata.form)) {
    return true;
  }
  if (response.metadata.query?.some((q) => q.incomplete)) {
    return true;
  }
  return false;
}

/**
 * Converts an array of form fields into a flattened map of default values.
 *
 * @param parameters The list of parameter form fields.
 * @returns A record mapping parameter IDs to their default values.
 */
export function flattenDefaultValues(
  parameters: readonly ParameterFormField[],
): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const parameter of parameters) {
    if (
      parameter.id === '__proto__' ||
      parameter.id === 'constructor' ||
      parameter.id === 'prototype'
    ) {
      continue;
    }
    switch (parameter.type) {
      case ParameterInputType.Text:
      case ParameterInputType.Set:
      case ParameterInputType.Checkbox:
        result[parameter.id] = parameter.default;
        break;
      case ParameterInputType.Group:
        Object.assign(result, flattenDefaultValues(parameter.children));
        break;
      case ParameterInputType.File:
      case ParameterInputType.Callout:
        break;
      default: {
        const exhaustiveCheck: never = parameter;
        void exhaustiveCheck;
        break;
      }
    }
  }
  return result;
}

/**
 * Counts input form fields currently showing an active error hint.
 *
 * Excludes read-only callout banners and suppresses error count when the field is pending or validating.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @param isValidating A callback returning true if the field is currently being validated.
 * @returns The number of input fields with active error hints.
 */
export function countErrorFields(
  parameters: readonly ParameterFormField[],
  isValidating: (id: string) => boolean,
): number {
  return countLeafFields(parameters, (field) => {
    if (!isInputField(field)) {
      return false;
    }
    const isPending = !!field.pending || isValidating(field.id);
    return field.hintType === ParameterHintType.Error && !isPending;
  });
}

/**
 * Counts input form fields currently in a pending or validating state.
 *
 * Excludes read-only callout banners.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @param isValidating A callback returning true if the field is currently being validated.
 * @returns The number of input fields currently pending.
 */
export function countPendingFields(
  parameters: readonly ParameterFormField[],
  isValidating: (id: string) => boolean,
): number {
  return countLeafFields(
    parameters,
    (field) =>
      isInputField(field) && (!!field.pending || isValidating(field.id)),
  );
}

/**
 * Checks whether any read-only callout banner in the form has an active error severity.
 *
 * Suppresses error status while the callout itself is pending.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @returns True if at least one non-pending callout banner has an error severity.
 */
export function hasCalloutErrors(
  parameters: readonly ParameterFormField[],
): boolean {
  return someLeafField(
    parameters,
    (field) =>
      field.type === ParameterInputType.Callout &&
      field.hintType === ParameterHintType.Error &&
      !field.pending,
  );
}

/**
 * Checks whether any read-only callout banner in the form is currently pending.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @returns True if at least one callout banner is pending.
 */
export function hasPendingCallouts(
  parameters: readonly ParameterFormField[],
): boolean {
  return someLeafField(
    parameters,
    (field) => field.type === ParameterInputType.Callout && !!field.pending,
  );
}

/**
 * Counts all user-configurable input fields within the parameter list, excluding groups and read-only callouts.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @returns The total number of user input fields.
 */
export function countAllFields(
  parameters: readonly ParameterFormField[],
): number {
  return countLeafFields(parameters, isInputField);
}

/**
 * Constructs a ParameterStepViewModel from inspection dryrun metadata.
 *
 * @param metadata The dryrun metadata returned from the inspection dryrun endpoint.
 * @returns The constructed view model for the parameter page.
 */
export function buildParameterStepViewModel(
  metadata: InspectionMetadataInDryrun,
): ParameterStepViewModel {
  const rootGroupForm: GroupParameterFormField = {
    id: 'root',
    label: '',
    description: '',
    hint: '',
    hintType: ParameterHintType.None,
    type: ParameterInputType.Group,
    collapsible: false,
    collapsedByDefault: false,
    children: metadata.form,
  };
  return {
    rootGroupForm,
    queries: metadata.query,
    job: metadata.jobCommand,
    fieldCount: countAllFields(metadata.form),
    totalEstimatedSummary: computeTotalEstimatedLogs(metadata.query),
  };
}

/**
 * Builds the URL to open the Task Graph Diagnostics page pre-configured with the selected inspection type and enabled features.
 *
 * @param inspectionType The currently selected inspection type, or null if none is selected.
 * @param features The list of inspection features for the selected inspection type.
 * @returns The constructed Task Graph Diagnostics URL.
 */
export function buildTaskGraphDebugUrl(
  inspectionType: InspectionType | null,
  features: readonly InspectionFeature[],
): string {
  const basePath = ApiPathUtil.getServerBasePath();
  const params = new URLSearchParams();
  params.set('tab', TaskGraphDebugTab.DAG_VIEWER);
  if (inspectionType) {
    params.set('inspectionType', inspectionType.id);
  }
  const enabledFeatureIds = features.filter((f) => f.enabled).map((f) => f.id);
  params.set('features', enabledFeatureIds.join(','));
  return `${basePath}/debug/task-graph?${params.toString()}`;
}
