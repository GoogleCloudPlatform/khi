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

import { create } from '@bufbuild/protobuf';
import {
  FormFieldSchema,
  ParameterHintType as ProtoParameterHintType,
  ValidationTiming as ProtoValidationTiming,
  UploadStatus as ProtoUploadStatus,
  InspectionPhase as ProtoInspectionPhase,
  InspectionListItemSchema,
  DryRunInspectionResponseSchema,
  GetInspectionMetadataResponseSchema,
} from 'src/app/generated/api/v1/inspection_pb';
import {
  ParameterInputType,
  ParameterHintType,
  ParameterFormValidationTiming,
  UploadStatus,
  GroupParameterFormField,
  TextParameterFormField,
  FileParameterFormField,
  SetParameterFormField,
} from 'src/app/common/schema/form-types';
import {
  convertMapToParameterValues,
  convertProtoDryRunResponseToFrontend,
  convertProtoFormFieldToParameterFormField,
  convertProtoListItemToInspectionMetadata,
  convertProtoMetadataToInspectionMetadataOfRunResult,
  mapProtoHintTypeToFormHintType,
  mapProtoUploadStatusToFormUploadStatus,
  mapProtoValidationTimingToFormTiming,
} from './inspection-converter';

describe('inspection-converter', () => {
  describe('enum mapping functions', () => {
    it('maps ProtoParameterHintType correctly', () => {
      expect(mapProtoHintTypeToFormHintType(ProtoParameterHintType.ERROR)).toBe(
        ParameterHintType.Error,
      );
      expect(
        mapProtoHintTypeToFormHintType(ProtoParameterHintType.WARNING),
      ).toBe(ParameterHintType.Warning);
      expect(mapProtoHintTypeToFormHintType(ProtoParameterHintType.INFO)).toBe(
        ParameterHintType.Info,
      );
      expect(mapProtoHintTypeToFormHintType(ProtoParameterHintType.NONE)).toBe(
        ParameterHintType.None,
      );
      expect(
        mapProtoHintTypeToFormHintType(ProtoParameterHintType.UNSPECIFIED),
      ).toBe(ParameterHintType.None);
    });

    it('maps ProtoValidationTiming correctly', () => {
      expect(
        mapProtoValidationTimingToFormTiming(ProtoValidationTiming.BLUR),
      ).toBe(ParameterFormValidationTiming.Blur);
      expect(
        mapProtoValidationTimingToFormTiming(ProtoValidationTiming.CHANGE),
      ).toBe(ParameterFormValidationTiming.Change);
      expect(
        mapProtoValidationTimingToFormTiming(ProtoValidationTiming.UNSPECIFIED),
      ).toBe(ParameterFormValidationTiming.Change);
    });

    it('maps ProtoUploadStatus correctly', () => {
      expect(
        mapProtoUploadStatusToFormUploadStatus(ProtoUploadStatus.UPLOADING),
      ).toBe(UploadStatus.Uploading);
      expect(
        mapProtoUploadStatusToFormUploadStatus(ProtoUploadStatus.VERIFYING),
      ).toBe(UploadStatus.Verifying);
      expect(
        mapProtoUploadStatusToFormUploadStatus(ProtoUploadStatus.DONE),
      ).toBe(UploadStatus.Done);
      expect(
        mapProtoUploadStatusToFormUploadStatus(ProtoUploadStatus.WAITING),
      ).toBe(UploadStatus.Waiting);
      expect(
        mapProtoUploadStatusToFormUploadStatus(ProtoUploadStatus.UNSPECIFIED),
      ).toBe(UploadStatus.Waiting);
    });
  });

  describe('convertProtoFormFieldToParameterFormField', () => {
    it('returns null when kind is not set', () => {
      const field = create(FormFieldSchema, {});
      expect(convertProtoFormFieldToParameterFormField(field)).toBeNull();
    });

    it('converts group field recursively', () => {
      const field = create(FormFieldSchema, {
        id: 'group-1',
        label: 'Group 1',
        description: 'Group desc',
        hint: 'Group hint',
        hintType: ProtoParameterHintType.INFO,
        kind: {
          case: 'group',
          value: {
            collapsible: true,
            collapsedByDefault: false,
            children: [
              create(FormFieldSchema, {
                id: 'child-text',
                label: 'Child text',
                kind: {
                  case: 'text',
                  value: {
                    defaultValue: 'val',
                    readonly: false,
                    suggestions: [],
                    validationTiming: ProtoValidationTiming.CHANGE,
                  },
                },
              }),
            ],
          },
        },
      });

      const converted = convertProtoFormFieldToParameterFormField(
        field,
      ) as GroupParameterFormField;
      expect(converted).not.toBeNull();
      expect(converted.id).toBe('group-1');
      expect(converted.type).toBe(ParameterInputType.Group);
      expect(converted.label).toBe('Group 1');
      expect(converted.hintType).toBe(ParameterHintType.Info);
      expect(converted.children.length).toBe(1);
      expect(converted.children[0].id).toBe('child-text');
      expect(converted.children[0].type).toBe(ParameterInputType.Text);
    });

    it('converts text field', () => {
      const field = create(FormFieldSchema, {
        id: 'text-1',
        label: 'Text 1',
        description: 'Text desc',
        hint: 'Required',
        hintType: ProtoParameterHintType.ERROR,
        kind: {
          case: 'text',
          value: {
            readonly: true,
            defaultValue: 'hello',
            suggestions: ['sug1', 'sug2'],
            validationTiming: ProtoValidationTiming.BLUR,
          },
        },
      });

      const converted = convertProtoFormFieldToParameterFormField(
        field,
      ) as TextParameterFormField;
      expect(converted).not.toBeNull();
      expect(converted.id).toBe('text-1');
      expect(converted.type).toBe(ParameterInputType.Text);
      expect(converted.label).toBe('Text 1');
      expect(converted.description).toBe('Text desc');
      expect(converted.hint).toBe('Required');
      expect(converted.hintType).toBe(ParameterHintType.Error);
      expect(converted.readonly).toBeTrue();
      expect(converted.default).toBe('hello');
      expect(converted.suggestions).toEqual(['sug1', 'sug2']);
      expect(converted.validationTiming).toBe(
        ParameterFormValidationTiming.Blur,
      );
    });

    it('converts file field', () => {
      const field = create(FormFieldSchema, {
        id: 'file-1',
        label: 'File 1',
        description: 'File desc',
        hint: 'Upload file',
        hintType: ProtoParameterHintType.WARNING,
        kind: {
          case: 'file',
          value: {
            tokenId: 'token-abc',
            status: ProtoUploadStatus.DONE,
          },
        },
      });

      const converted = convertProtoFormFieldToParameterFormField(
        field,
      ) as FileParameterFormField;
      expect(converted).not.toBeNull();
      expect(converted.id).toBe('file-1');
      expect(converted.type).toBe(ParameterInputType.File);
      expect(converted.token.id).toBe('token-abc');
      expect(converted.status).toBe(UploadStatus.Done);
      expect(converted.hintType).toBe(ParameterHintType.Warning);
    });

    it('converts set field', () => {
      const field = create(FormFieldSchema, {
        id: 'set-1',
        label: 'Set 1',
        description: 'Set desc',
        hint: 'Choose items',
        hintType: ProtoParameterHintType.NONE,
        kind: {
          case: 'set',
          value: {
            options: [{ id: 'opt1', description: 'Option 1' }],
            defaultValues: ['opt1'],
            allowAddAll: true,
            allowRemoveAll: false,
            allowCustomValue: true,
          },
        },
      });

      const converted = convertProtoFormFieldToParameterFormField(
        field,
      ) as SetParameterFormField;
      expect(converted).not.toBeNull();
      expect(converted.id).toBe('set-1');
      expect(converted.type).toBe(ParameterInputType.Set);
      expect(converted.options).toEqual([
        { id: 'opt1', description: 'Option 1' },
      ]);
      expect(converted.default).toEqual(['opt1']);
      expect(converted.allowAddAll).toBeTrue();
      expect(converted.allowRemoveAll).toBeFalse();
      expect(converted.allowCustomValue).toBeTrue();
    });
  });

  describe('convertMapToParameterValues', () => {
    it('converts primitive strings, numbers, booleans, arrays and file tokens', () => {
      const params: Record<string, unknown> = {
        name: 'test-inspection',
        count: 42,
        enabled: true,
        tags: ['alpha', 'beta'],
        uploadedFile: { id: 'tok-123' },
        emptyVal: null,
        undefVal: undefined,
      };

      const result = convertMapToParameterValues(params);
      expect(result.length).toBe(5);

      const nameVal = result.find((p) => p.id === 'name');
      expect(nameVal?.value.case).toBe('textValue');
      if (nameVal?.value.case === 'textValue') {
        expect(nameVal.value.value.value).toBe('test-inspection');
      }

      const countVal = result.find((p) => p.id === 'count');
      expect(countVal?.value.case).toBe('textValue');
      if (countVal?.value.case === 'textValue') {
        expect(countVal.value.value.value).toBe('42');
      }

      const boolVal = result.find((p) => p.id === 'enabled');
      expect(boolVal?.value.case).toBe('textValue');
      if (boolVal?.value.case === 'textValue') {
        expect(boolVal.value.value.value).toBe('true');
      }

      const tagsVal = result.find((p) => p.id === 'tags');
      expect(tagsVal?.value.case).toBe('setValue');
      if (tagsVal?.value.case === 'setValue') {
        expect(tagsVal.value.value.values).toEqual(['alpha', 'beta']);
      }

      const fileVal = result.find((p) => p.id === 'uploadedFile');
      expect(fileVal?.value.case).toBe('fileValue');
      if (fileVal?.value.case === 'fileValue') {
        expect(fileVal.value.value.token).toBe('tok-123');
      }
    });
  });

  describe('convertProtoListItemToInspectionMetadata', () => {
    it('converts proto item to list metadata structure', () => {
      const item = create(InspectionListItemSchema, {
        id: 'insp-1',
        header: {
          inspectionType: 'gcp-gke',
          inspectionName: 'Cluster Audit',
          inspectionTypeIconPath: '/icons/gke.svg',
          startTimeUnixSeconds: 1000n,
          endTimeUnixSeconds: 2000n,
          inspectTimeUnixSeconds: 1500n,
          fileSize: 1024n,
          suggestedFilename: 'cluster.khi',
        },
        progress: {
          phase: ProtoInspectionPhase.RUNNING,
          totalProgress: {
            id: 'total',
            label: 'Overall',
            message: 'In progress',
            percentage: 0.5,
          },
          progresses: [
            {
              id: 'task-1',
              label: 'Step 1',
              message: 'Done',
              percentage: 1.0,
              indeterminate: false,
            },
          ],
        },
        error: {
          errorMessages: [
            {
              errorId: 'ERR_01',
              message: 'A warning occurred',
              link: 'https://khi.example.com',
            },
          ],
        },
      });

      const metadata = convertProtoListItemToInspectionMetadata(item);
      expect(metadata.header.inspectionType).toBe('gcp-gke');
      expect(metadata.header.inspectionName).toBe('Cluster Audit');
      expect(metadata.header.fileSize).toBe(1024);
      expect(metadata.progress.phase).toBe('RUNNING');
      expect(metadata.progress.totalProgress.percentage).toBe(0.5);
      expect(metadata.progress.progresses.length).toBe(1);
      expect(metadata.error.errorMessages.length).toBe(1);
      expect(metadata.error.errorMessages[0].message).toBe(
        'A warning occurred',
      );
    });
  });

  describe('convertProtoDryRunResponseToFrontend', () => {
    it('converts dryrun response correctly', () => {
      const res = create(DryRunInspectionResponseSchema, {
        form: [
          create(FormFieldSchema, {
            id: 'f1',
            label: 'Field 1',
            kind: {
              case: 'text',
              value: { defaultValue: '', readonly: false, suggestions: [] },
            },
          }),
        ],
        queries: [{ id: 'q1', name: 'q1', query: 'resource.type="gke"' }],
        plan: { taskGraph: 'graph TD; A-->B;' },
        jobCommand: { command: 'khi run ...' },
      });

      const converted = convertProtoDryRunResponseToFrontend(res);
      expect(converted.metadata.form.length).toBe(1);
      expect(converted.metadata.form[0].id).toBe('f1');
      expect(converted.metadata.query.length).toBe(1);
      expect(converted.metadata.query[0].query).toBe('resource.type="gke"');
      expect(converted.metadata.plan.taskGraph).toBe('graph TD; A-->B;');
      expect(converted.metadata.jobCommand?.command).toBe('khi run ...');
    });
  });

  describe('convertProtoMetadataToInspectionMetadataOfRunResult', () => {
    it('converts run metadata response correctly', () => {
      const res = create(GetInspectionMetadataResponseSchema, {
        header: {
          inspectionType: 'gcp-gke',
          inspectionName: 'Run 1',
          fileSize: 2048n,
          suggestedFilename: 'run1.khi',
        },
        plan: { taskGraph: 'graph' },
        queries: [{ id: 'q1', name: 'q1', query: 'log' }],
        logs: [{ id: '1', name: 'INFO', log: 'Finished' }],
        error: {
          errorMessages: [{ errorId: 'ERR_1', message: 'err', link: 'link' }],
        },
      });

      const converted =
        convertProtoMetadataToInspectionMetadataOfRunResult(res);
      expect(converted.header.inspectionName).toBe('Run 1');
      expect(converted.header.fileSize).toBe(2048);
      expect(converted.log.length).toBe(1);
      expect(converted.log[0].log).toBe('Finished');
      expect(converted.error.errorMessages.length).toBe(1);
    });
  });
});
