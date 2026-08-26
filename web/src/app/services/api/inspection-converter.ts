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

import { create, isFieldSet } from '@bufbuild/protobuf';
import {
  FormField,
  GroupFormField,
  TextFormField,
  FileFormField,
  SetFormField,
  ParameterHintType as ProtoParameterHintType,
  ValidationTiming as ProtoValidationTiming,
  UploadStatus as ProtoUploadStatus,
  InspectionPhase as ProtoInspectionPhase,
  ParameterValue,
  ParameterValueSchema,
  InspectionListItem,
  GetInspectionMetadataResponse,
  DryRunInspectionResponse,
  InspectionQuerySchema,
} from 'src/app/generated/api/v1/inspection_pb';
import {
  ParameterFormField,
  GroupParameterFormField,
  TextParameterFormField,
  FileParameterFormField,
  SetParameterFormField,
  ParameterInputType,
  ParameterHintType,
  ParameterFormValidationTiming,
  UploadStatus,
} from 'src/app/common/schema/form-types';
import {
  InspectionMetadataInInspectionList,
  InspectionMetadataOfRunResult,
  InspectionDryRunResponse,
} from 'src/app/common/schema/api-types';
import { InspectionMetadataProgressPhase } from 'src/app/common/schema/metadata-types';

/**
 * Maps proto ParameterHintType to frontend ParameterHintType enum.
 */
export function mapProtoHintTypeToFormHintType(
  protoHint: ProtoParameterHintType,
): ParameterHintType {
  switch (protoHint) {
    case ProtoParameterHintType.ERROR:
      return ParameterHintType.Error;
    case ProtoParameterHintType.WARNING:
      return ParameterHintType.Warning;
    case ProtoParameterHintType.INFO:
      return ParameterHintType.Info;
    case ProtoParameterHintType.NONE:
    case ProtoParameterHintType.UNSPECIFIED:
    default:
      return ParameterHintType.None;
  }
}

/**
 * Maps proto ValidationTiming to frontend ParameterFormValidationTiming enum.
 */
export function mapProtoValidationTimingToFormTiming(
  protoTiming: ProtoValidationTiming,
): ParameterFormValidationTiming {
  switch (protoTiming) {
    case ProtoValidationTiming.BLUR:
      return ParameterFormValidationTiming.Blur;
    case ProtoValidationTiming.CHANGE:
    case ProtoValidationTiming.UNSPECIFIED:
    default:
      return ParameterFormValidationTiming.Change;
  }
}

/**
 * Maps proto UploadStatus to frontend UploadStatus enum.
 */
export function mapProtoUploadStatusToFormUploadStatus(
  protoStatus: ProtoUploadStatus,
): UploadStatus {
  switch (protoStatus) {
    case ProtoUploadStatus.UPLOADING:
      return UploadStatus.Uploading;
    case ProtoUploadStatus.VERIFYING:
      return UploadStatus.Verifying;
    case ProtoUploadStatus.DONE:
      return UploadStatus.Done;
    case ProtoUploadStatus.WAITING:
    case ProtoUploadStatus.UNSPECIFIED:
    default:
      return UploadStatus.Waiting;
  }
}

/**
 * Maps proto InspectionPhase to frontend InspectionMetadataProgressPhase.
 */
export function mapProtoInspectionPhaseToProgressPhase(
  phase?: ProtoInspectionPhase,
): InspectionMetadataProgressPhase {
  switch (phase) {
    case ProtoInspectionPhase.RUNNING:
      return 'RUNNING';
    case ProtoInspectionPhase.ERROR:
      return 'ERROR';
    case ProtoInspectionPhase.CANCELLED:
      return 'CANCELLED';
    case ProtoInspectionPhase.DONE:
      return 'DONE';
    default:
      return 'RUNNING';
  }
}

/**
 * Converts a proto FormField into the frontend ParameterFormField domain model.
 */
export function convertProtoFormFieldToParameterFormField(
  field: FormField,
): ParameterFormField | null {
  if (!field.kind) {
    return null;
  }

  const commonId = field.id;
  const commonLabel = field.label;
  const commonDesc = field.description;
  const commonHint = field.hint;
  const commonHintType = mapProtoHintTypeToFormHintType(field.hintType);

  switch (field.kind.case) {
    case 'group': {
      const gf: GroupFormField = field.kind.value;
      const children: ParameterFormField[] = [];
      for (const child of gf.children) {
        const converted = convertProtoFormFieldToParameterFormField(child);
        if (converted) {
          children.push(converted);
        }
      }
      const groupResult: GroupParameterFormField = {
        id: commonId,
        type: ParameterInputType.Group,
        label: commonLabel,
        description: commonDesc,
        hint: commonHint,
        hintType: commonHintType,
        children,
        collapsible: gf.collapsible,
        collapsedByDefault: gf.collapsedByDefault,
      };
      return groupResult;
    }
    case 'text': {
      const tf: TextFormField = field.kind.value;
      const textResult: TextParameterFormField = {
        id: commonId,
        type: ParameterInputType.Text,
        label: commonLabel,
        description: commonDesc,
        hint: commonHint,
        hintType: commonHintType,
        readonly: tf.readonly,
        default: tf.defaultValue,
        suggestions: [...tf.suggestions],
        validationTiming: mapProtoValidationTimingToFormTiming(
          tf.validationTiming,
        ),
      };
      return textResult;
    }
    case 'file': {
      const ff: FileFormField = field.kind.value;
      const fileResult: FileParameterFormField = {
        id: commonId,
        type: ParameterInputType.File,
        label: commonLabel,
        description: commonDesc,
        hint: commonHint,
        hintType: commonHintType,
        token: { id: ff.tokenId },
        status: mapProtoUploadStatusToFormUploadStatus(ff.status),
      };
      return fileResult;
    }
    case 'set': {
      const sf: SetFormField = field.kind.value;
      const setResult: SetParameterFormField = {
        id: commonId,
        type: ParameterInputType.Set,
        label: commonLabel,
        description: commonDesc,
        hint: commonHint,
        hintType: commonHintType,
        options: sf.options.map((opt) => ({
          id: opt.id,
          description: opt.description,
        })),
        default: [...sf.defaultValues],
        allowAddAll: sf.allowAddAll,
        allowRemoveAll: sf.allowRemoveAll,
        allowCustomValue: sf.allowCustomValue,
      };
      return setResult;
    }
    default:
      return null;
  }
}

/**
 * Converts a dictionary of parameter key-values to proto ParameterValue list.
 */
export function convertMapToParameterValues(
  params: Record<string, unknown>,
): ParameterValue[] {
  const result: ParameterValue[] = [];
  for (const [id, val] of Object.entries(params)) {
    if (val === undefined || val === null) {
      continue;
    }
    if (typeof val === 'string') {
      result.push(
        create(ParameterValueSchema, {
          id,
          value: {
            case: 'textValue',
            value: { value: val },
          },
        }),
      );
    } else if (typeof val === 'number' || typeof val === 'boolean') {
      result.push(
        create(ParameterValueSchema, {
          id,
          value: {
            case: 'textValue',
            value: { value: String(val) },
          },
        }),
      );
    } else if (Array.isArray(val)) {
      result.push(
        create(ParameterValueSchema, {
          id,
          value: {
            case: 'setValue',
            value: { values: val.map((v) => String(v)) },
          },
        }),
      );
    } else if (
      typeof val === 'object' &&
      val !== null &&
      'id' in (val as Record<string, unknown>)
    ) {
      result.push(
        create(ParameterValueSchema, {
          id,
          value: {
            case: 'fileValue',
            value: {
              token: String((val as Record<string, unknown>)['id']),
            },
          },
        }),
      );
    }
  }
  return result;
}

/**
 * Converts proto InspectionListItem to frontend InspectionMetadataInInspectionList model.
 */
export function convertProtoListItemToInspectionMetadata(
  item: InspectionListItem,
): InspectionMetadataInInspectionList {
  return {
    header: {
      inspectionType: item.header?.inspectionType ?? '',
      inspectionName: item.header?.inspectionName ?? '',
      inspectionTypeIconPath: item.header?.inspectionTypeIconPath ?? '',
      startTimeUnixSeconds: Number(item.header?.startTimeUnixSeconds ?? 0n),
      endTimeUnixSeconds: Number(item.header?.endTimeUnixSeconds ?? 0n),
      inspectTimeUnixSeconds: Number(item.header?.inspectTimeUnixSeconds ?? 0n),
      fileSize: Number(item.header?.fileSize ?? 0n),
      suggestedFilename: item.header?.suggestedFilename ?? '',
    },
    progress: {
      phase: mapProtoInspectionPhaseToProgressPhase(item.progress?.phase),
      totalProgress: {
        id: item.progress?.totalProgress?.id ?? '',
        label: item.progress?.totalProgress?.label ?? '',
        message: item.progress?.totalProgress?.message ?? '',
        percentage: item.progress?.totalProgress?.percentage ?? 0,
        indeterminate: false,
      },
      progresses: (item.progress?.progresses ?? []).map((p) => ({
        id: p.id,
        label: p.label,
        message: p.message,
        percentage: p.percentage,
        indeterminate: p.indeterminate,
      })),
    },
    error: {
      errorMessages: (item.error?.errorMessages ?? []).map((e) => ({
        errorId: e.errorId,
        message: e.message,
        link: e.link,
      })),
    },
  };
}

/**
 * Converts proto DryRunInspectionResponse to frontend InspectionDryRunResponse.
 */
export function convertProtoDryRunResponseToFrontend(
  res: DryRunInspectionResponse,
): InspectionDryRunResponse {
  const form: ParameterFormField[] = [];
  for (const field of res.form) {
    const converted = convertProtoFormFieldToParameterFormField(field);
    if (converted) {
      form.push(converted);
    }
  }
  return {
    metadata: {
      form,
      query: res.queries.map((q) => ({
        id: q.id,
        name: q.name,
        query: q.query,
        estimatedCount: isFieldSet(
          q,
          InspectionQuerySchema.field.estimatedCount,
        )
          ? Number(q.estimatedCount)
          : undefined,
        incomplete: q.incomplete ? true : undefined,
        pending: q.pending ? true : undefined,
      })),
      plan: {
        taskGraph: res.plan?.taskGraph ?? '',
      },
      jobCommand: res.jobCommand
        ? { command: res.jobCommand.command }
        : undefined,
    },
  };
}

/**
 * Converts proto GetInspectionMetadataResponse to frontend InspectionMetadataOfRunResult.
 */
export function convertProtoMetadataToInspectionMetadataOfRunResult(
  res: GetInspectionMetadataResponse,
): InspectionMetadataOfRunResult {
  return {
    header: {
      inspectionType: res.header?.inspectionType ?? '',
      inspectionName: res.header?.inspectionName ?? '',
      inspectionTypeIconPath: res.header?.inspectionTypeIconPath ?? '',
      startTimeUnixSeconds: Number(res.header?.startTimeUnixSeconds ?? 0n),
      endTimeUnixSeconds: Number(res.header?.endTimeUnixSeconds ?? 0n),
      inspectTimeUnixSeconds: Number(res.header?.inspectTimeUnixSeconds ?? 0n),
      fileSize: Number(res.header?.fileSize ?? 0n),
      suggestedFilename: res.header?.suggestedFilename ?? '',
    },
    plan: {
      taskGraph: res.plan?.taskGraph ?? '',
    },
    query: (res.queries ?? []).map((q) => ({
      id: q.id,
      name: q.name,
      query: q.query,
      estimatedCount: isFieldSet(q, InspectionQuerySchema.field.estimatedCount)
        ? Number(q.estimatedCount)
        : undefined,
      incomplete: q.incomplete ? true : undefined,
      pending: q.pending ? true : undefined,
    })),
    log: (res.logs ?? []).map((l) => ({
      id: l.id,
      name: l.name,
      log: l.log,
    })),
    error: {
      errorMessages: (res.error?.errorMessages ?? []).map((e) => ({
        errorId: e.errorId,
        message: e.message,
        link: e.link,
      })),
    },
  };
}
