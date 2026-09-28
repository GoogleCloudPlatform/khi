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
  InspectionMetadataInDryrun,
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

/**
 * Recursively checks if any parameter field in the form has an error hint.
 *
 * @param fields The list of form fields to check.
 * @returns True if at least one field has an error hint.
 */
export function hasFormErrors(fields: readonly ParameterFormField[]): boolean {
  for (const field of fields) {
    if (field.type === ParameterInputType.Group) {
      if (hasFormErrors(field.children)) {
        return true;
      }
    } else if (field.hintType === ParameterHintType.Error) {
      return true;
    }
  }
  return false;
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
      default:
        break;
    }
  }
  return result;
}

/**
 * Counts form fields currently showing an active error hint.
 *
 * Suppresses error count when the field is pending or validating.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @param isValidating A callback returning true if the field is currently being validated.
 * @returns The number of fields with active error hints.
 */
export function countErrorFields(
  parameters: readonly ParameterFormField[],
  isValidating: (id: string) => boolean,
): number {
  let result = 0;
  for (const parameter of parameters) {
    if (parameter.type === ParameterInputType.Group) {
      result += countErrorFields(parameter.children, isValidating);
    } else {
      const isClientValidating = isValidating(parameter.id);
      const isPending = !!parameter.pending || isClientValidating;
      if (parameter.hintType === ParameterHintType.Error && !isPending) {
        result++;
      }
    }
  }
  return result;
}

/**
 * Counts form fields currently in a pending or validating state.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @param isValidating A callback returning true if the field is currently being validated.
 * @returns The number of fields currently pending.
 */
export function countPendingFields(
  parameters: readonly ParameterFormField[],
  isValidating: (id: string) => boolean,
): number {
  let result = 0;
  for (const parameter of parameters) {
    if (parameter.type === ParameterInputType.Group) {
      result += countPendingFields(parameter.children, isValidating);
    } else {
      const isClientValidating = isValidating(parameter.id);
      if (parameter.pending || isClientValidating) {
        result++;
      }
    }
  }
  return result;
}

/**
 * Counts all non-group input fields within the parameter list.
 *
 * @param parameters The list of parameter form fields to evaluate.
 * @returns The total number of non-group fields.
 */
export function countAllFields(
  parameters: readonly ParameterFormField[],
): number {
  let result = 0;
  for (const parameter of parameters) {
    if (parameter.type === ParameterInputType.Group) {
      result += countAllFields(parameter.children);
    } else {
      result++;
    }
  }
  return result;
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
    plan: metadata.plan,
    job: metadata.jobCommand,
    fieldCount: countAllFields(metadata.form),
    totalEstimatedSummary: computeTotalEstimatedLogs(metadata.query),
  };
}
