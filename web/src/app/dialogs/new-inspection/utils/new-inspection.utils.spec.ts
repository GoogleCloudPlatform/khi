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
  CheckboxParameterFormField,
  FileParameterFormField,
  GroupParameterFormField,
  ParameterFormField,
  ParameterFormValidationTiming,
  ParameterHintType,
  ParameterInputType,
  SetParameterFormField,
  TextParameterFormField,
  UploadStatus,
} from 'src/app/common/schema/form-types';
import {
  EstimatedCountPreset,
  InspectionMetadataQuery,
} from 'src/app/common/schema/metadata-types';
import { TotalEstimatedLogsSeverity } from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import {
  buildParameterStepViewModel,
  computeTotalEstimatedLogs,
  countAllFields,
  countErrorFields,
  countPendingFields,
  flattenDefaultValues,
  hasDryRunErrors,
  hasFormErrors,
} from 'src/app/dialogs/new-inspection/utils/new-inspection.utils';

describe('new-inspection.utils', () => {
  describe('computeTotalEstimatedLogs', () => {
    it('should return undefined for empty or undefined query list', () => {
      expect(computeTotalEstimatedLogs(undefined)).toBeUndefined();
      expect(computeTotalEstimatedLogs([])).toBeUndefined();
    });

    it('should calculate complete total with Normal severity when < 1,000,000', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 500 },
        { id: 'q2', name: 'q2', query: 'query2', estimatedCount: 1500 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 2000,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~2,000 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should calculate complete total with 0 logs', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 0 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~0 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should format partial estimate with > prefix when some queries are in-flight', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1250 },
        { id: 'q2', name: 'q2', query: 'query2' },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1250,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: '>1,250 logs estimated so far',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should display Estimating total logs... when all queries are unestimated', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1' },
        { id: 'q2', name: 'q2', query: 'query2' },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: 'Estimating total logs...',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should assign Warning severity for counts between 1,000,000 and 4,999,999', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1200000 },
        { id: 'q2', name: 'q2', query: 'query2', estimatedCount: 300000 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1500000,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~1,500,000 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Warning,
      });
    });

    it('should assign Danger severity for counts >= 5,000,000', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 5000000 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 5000000,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~5,000,000 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Danger,
      });
    });

    it('should assign Danger severity for partial estimates >= 5,000,000', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 6500000 },
        { id: 'q2', name: 'q2', query: 'query2' },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 6500000,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: '>6,500,000 logs estimated so far',
        severity: TotalEstimatedLogsSeverity.Danger,
      });
    });

    it('should display Incomplete parameters when all queries are incomplete with no counts', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', incomplete: true },
        { id: 'q2', name: 'q2', query: 'query2', incomplete: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: false,
        isEstimating: false,
        isIncomplete: true,
        displayText: 'Incomplete parameters',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should display >N logs estimated (some parameters incomplete) when some queries are estimated and others are incomplete', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 2500 },
        { id: 'q2', name: 'q2', query: 'query2', incomplete: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 2500,
        isComplete: false,
        isEstimating: false,
        isIncomplete: true,
        displayText: '>2,500 logs estimated (some parameters incomplete)',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should set isEstimating to true when some queries are incomplete and others are still estimating', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1000 },
        { id: 'q2', name: 'q2', query: 'query2' },
        { id: 'q3', name: 'q3', query: 'query3', incomplete: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1000,
        isComplete: false,
        isEstimating: true,
        isIncomplete: true,
        displayText: '>1,000 logs estimated (some parameters incomplete)',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should treat queries with pending = true as estimating', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1000 },
        { id: 'q2', name: 'q2', query: 'query2', pending: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1000,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: '>1,000 logs estimated so far',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should display Few total logs estimated when only Few preset queries exist with 0 known count', () => {
      const queries: InspectionMetadataQuery[] = [
        {
          id: 'q1',
          name: 'q1',
          query: 'query1',
          estimatedCountPreset: EstimatedCountPreset.Few,
        },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: 'Few total logs estimated',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should include Few preset query as resolved alongside numeric estimated queries', () => {
      const queries: InspectionMetadataQuery[] = [
        {
          id: 'q1',
          name: 'q1',
          query: 'query1',
          estimatedCountPreset: EstimatedCountPreset.Few,
        },
        { id: 'q2', name: 'q2', query: 'query2', estimatedCount: 500 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 500,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~500 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should show isEstimating = true when a Few preset query is present with an unestimated query', () => {
      const queries: InspectionMetadataQuery[] = [
        {
          id: 'q1',
          name: 'q1',
          query: 'query1',
          estimatedCountPreset: EstimatedCountPreset.Few,
        },
        { id: 'q2', name: 'q2', query: 'query2' },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: 'Estimating total logs...',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should treat query with EstimatedCountPreset.None as unestimated when count is undefined', () => {
      const queries: InspectionMetadataQuery[] = [
        {
          id: 'q1',
          name: 'q1',
          query: 'query1',
          estimatedCountPreset: EstimatedCountPreset.None,
        },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: 'Estimating total logs...',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });
  });

  describe('hasFormErrors and hasDryRunErrors', () => {
    it('hasFormErrors should return true if any field has error hint', () => {
      expect(
        hasFormErrors([
          {
            id: 'field1',
            type: ParameterInputType.Text,
            label: 'Field 1',
            description: '',
            hint: 'Error message',
            hintType: ParameterHintType.Error,
            default: '',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Blur,
          },
        ]),
      ).toBe(true);
    });

    it('hasFormErrors should return true if nested group field has error hint', () => {
      expect(
        hasFormErrors([
          {
            id: 'group1',
            type: ParameterInputType.Group,
            label: 'Group 1',
            description: '',
            hint: '',
            hintType: ParameterHintType.None,
            collapsible: false,
            collapsedByDefault: false,
            children: [
              {
                id: 'nested-field',
                type: ParameterInputType.Text,
                label: 'Nested Field',
                description: '',
                hint: 'Nested error',
                hintType: ParameterHintType.Error,
                default: '',
                readonly: false,
                suggestions: [],
                validationTiming: ParameterFormValidationTiming.Blur,
              },
            ],
          },
        ]),
      ).toBe(true);
    });

    it('hasFormErrors should return false when no errors exist', () => {
      expect(
        hasFormErrors([
          {
            id: 'field1',
            type: ParameterInputType.Text,
            label: 'Field 1',
            description: '',
            hint: '',
            hintType: ParameterHintType.None,
            default: '',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Blur,
          },
        ]),
      ).toBe(false);
    });

    it('hasDryRunErrors should return true when query is incomplete', () => {
      const response: InspectionDryRunResponse = {
        metadata: {
          form: [],
          query: [
            {
              id: 'q1',
              name: 'Query 1',
              query: 'q',
              incomplete: true,
            },
          ],
          plan: { taskGraph: '' },
        },
      };
      expect(hasDryRunErrors(response)).toBe(true);
    });

    it('hasDryRunErrors should return false when valid', () => {
      const response: InspectionDryRunResponse = {
        metadata: {
          form: [],
          query: [
            {
              id: 'q1',
              name: 'Query 1',
              query: 'q',
              incomplete: false,
            },
          ],
          plan: { taskGraph: '' },
        },
      };
      expect(hasDryRunErrors(response)).toBe(false);
    });

    it('hasDryRunErrors should return true when form has errors', () => {
      const response: InspectionDryRunResponse = {
        metadata: {
          form: [
            {
              id: 'field1',
              type: ParameterInputType.Text,
              label: 'Field 1',
              description: '',
              hint: 'Invalid value',
              hintType: ParameterHintType.Error,
              default: '',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            },
          ],
          query: [],
          plan: { taskGraph: '' },
        },
      };
      expect(hasDryRunErrors(response)).toBe(true);
    });
  });

  describe('flattenDefaultValues', () => {
    it('should return empty object for empty parameters list', () => {
      expect(flattenDefaultValues([])).toEqual({});
    });

    it('should extract defaults for text, set, and checkbox fields', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'text-param',
          type: ParameterInputType.Text,
          label: 'Text',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: 'default-text',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
        {
          id: 'set-param',
          type: ParameterInputType.Set,
          label: 'Set',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          options: [],
          default: ['val1', 'val2'],
          allowAddAll: false,
          allowRemoveAll: false,
          allowCustomValue: false,
        } as SetParameterFormField,
        {
          id: 'checkbox-param',
          type: ParameterInputType.Checkbox,
          label: 'Checkbox',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          readonly: false,
          default: true,
        } as CheckboxParameterFormField,
      ];

      expect(flattenDefaultValues(fields)).toEqual({
        'text-param': 'default-text',
        'set-param': ['val1', 'val2'],
        'checkbox-param': true,
      });
    });

    it('should recursively flatten default values for nested group fields', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'outer-group',
          type: ParameterInputType.Group,
          label: 'Outer Group',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          collapsible: false,
          collapsedByDefault: false,
          children: [
            {
              id: 'nested-text',
              type: ParameterInputType.Text,
              label: 'Nested Text',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              default: 'nested-val',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            } as TextParameterFormField,
            {
              id: 'inner-group',
              type: ParameterInputType.Group,
              label: 'Inner Group',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              collapsible: false,
              collapsedByDefault: false,
              children: [
                {
                  id: 'deep-checkbox',
                  type: ParameterInputType.Checkbox,
                  label: 'Deep Checkbox',
                  description: '',
                  hint: '',
                  hintType: ParameterHintType.None,
                  readonly: false,
                  default: false,
                } as CheckboxParameterFormField,
              ],
            } as GroupParameterFormField,
          ],
        } as GroupParameterFormField,
      ];

      expect(flattenDefaultValues(fields)).toEqual({
        'nested-text': 'nested-val',
        'deep-checkbox': false,
      });
    });

    it('should ignore file parameter fields without default values', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'file-param',
          type: ParameterInputType.File,
          label: 'File',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          token: { id: 'token-1' },
          status: UploadStatus.Waiting,
        } as FileParameterFormField,
      ];

      expect(flattenDefaultValues(fields)).toEqual({});
    });

    it('should ignore dangerous prototype pollution keys', () => {
      const fields: ParameterFormField[] = [
        {
          id: '__proto__',
          type: ParameterInputType.Text,
          label: 'Proto',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: 'polluted',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
        {
          id: 'constructor',
          type: ParameterInputType.Text,
          label: 'Constructor',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: 'polluted',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
        {
          id: 'prototype',
          type: ParameterInputType.Text,
          label: 'Prototype',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: 'polluted',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
        {
          id: 'safe-key',
          type: ParameterInputType.Text,
          label: 'Safe',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: 'safe-value',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(flattenDefaultValues(fields)).toEqual({
        'safe-key': 'safe-value',
      });
    });
  });

  describe('countErrorFields', () => {
    it('should return 0 when no fields have error hints', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(countErrorFields(fields, () => false)).toBe(0);
    });

    it('should count fields with Error hintType', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: 'Error 1',
          hintType: ParameterHintType.Error,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
        {
          id: 'field2',
          type: ParameterInputType.Text,
          label: 'Field 2',
          description: '',
          hint: 'Warning',
          hintType: ParameterHintType.Warning,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(countErrorFields(fields, () => false)).toBe(1);
    });

    it('should recursively count error fields in nested group fields', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'group1',
          type: ParameterInputType.Group,
          label: 'Group 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          collapsible: false,
          collapsedByDefault: false,
          children: [
            {
              id: 'nested1',
              type: ParameterInputType.Text,
              label: 'Nested 1',
              description: '',
              hint: 'Nested Error',
              hintType: ParameterHintType.Error,
              default: '',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            } as TextParameterFormField,
          ],
        } as GroupParameterFormField,
      ];

      expect(countErrorFields(fields, () => false)).toBe(1);
    });

    it('should suppress error count when field has pending = true', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: 'Error 1',
          hintType: ParameterHintType.Error,
          pending: true,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(countErrorFields(fields, () => false)).toBe(0);
    });

    it('should suppress error count when isValidating returns true', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: 'Error 1',
          hintType: ParameterHintType.Error,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(countErrorFields(fields, (id) => id === 'field1')).toBe(0);
    });
  });

  describe('countPendingFields', () => {
    it('should return 0 when no fields are pending or validating', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(countPendingFields(fields, () => false)).toBe(0);
    });

    it('should count fields with pending = true', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          pending: true,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(countPendingFields(fields, () => false)).toBe(1);
    });

    it('should count fields where isValidating returns true', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
      ];

      expect(countPendingFields(fields, (id) => id === 'field1')).toBe(1);
    });

    it('should recursively count pending fields in nested group fields', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'group1',
          type: ParameterInputType.Group,
          label: 'Group 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          collapsible: false,
          collapsedByDefault: false,
          children: [
            {
              id: 'nested1',
              type: ParameterInputType.Text,
              label: 'Nested 1',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              pending: true,
              default: '',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            } as TextParameterFormField,
            {
              id: 'nested2',
              type: ParameterInputType.Text,
              label: 'Nested 2',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              default: '',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            } as TextParameterFormField,
          ],
        } as GroupParameterFormField,
      ];

      expect(countPendingFields(fields, (id) => id === 'nested2')).toBe(2);
    });
  });

  describe('countAllFields', () => {
    it('should return 0 for empty list', () => {
      expect(countAllFields([])).toBe(0);
    });

    it('should count non-group fields and ignore Group containers', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          default: '',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
        {
          id: 'field2',
          type: ParameterInputType.Checkbox,
          label: 'Field 2',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          readonly: false,
          default: true,
        } as CheckboxParameterFormField,
      ];

      expect(countAllFields(fields)).toBe(2);
    });

    it('should count non-group fields inside nested Group fields', () => {
      const fields: ParameterFormField[] = [
        {
          id: 'group1',
          type: ParameterInputType.Group,
          label: 'Group 1',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          collapsible: false,
          collapsedByDefault: false,
          children: [
            {
              id: 'field1',
              type: ParameterInputType.Text,
              label: 'Field 1',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              default: '',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            } as TextParameterFormField,
            {
              id: 'group2',
              type: ParameterInputType.Group,
              label: 'Group 2',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              collapsible: false,
              collapsedByDefault: false,
              children: [
                {
                  id: 'field2',
                  type: ParameterInputType.Checkbox,
                  label: 'Field 2',
                  description: '',
                  hint: '',
                  hintType: ParameterHintType.None,
                  readonly: false,
                  default: false,
                } as CheckboxParameterFormField,
              ],
            } as GroupParameterFormField,
          ],
        } as GroupParameterFormField,
      ];

      expect(countAllFields(fields)).toBe(2);
    });
  });

  describe('buildParameterStepViewModel', () => {
    it('should construct ParameterStepViewModel with correct rootGroupForm and field counts', () => {
      const formFields: ParameterFormField[] = [
        {
          id: 'field1',
          type: ParameterInputType.Text,
          label: 'Field 1',
          description: '',
          hint: 'Field Error',
          hintType: ParameterHintType.Error,
          default: 'val',
          readonly: false,
          suggestions: [],
          validationTiming: ParameterFormValidationTiming.Blur,
        } as TextParameterFormField,
        {
          id: 'field2',
          type: ParameterInputType.Checkbox,
          label: 'Field 2',
          description: '',
          hint: '',
          hintType: ParameterHintType.None,
          pending: true,
          readonly: false,
          default: false,
        } as CheckboxParameterFormField,
      ];
      const metadata: InspectionMetadataInDryrun = {
        form: formFields,
        query: [
          {
            id: 'q1',
            name: 'Query 1',
            query: 'query',
            estimatedCount: 100,
          },
        ],
        plan: {
          taskGraph: 'graph-dot-string',
        },
        jobCommand: {
          command: 'khi run',
        },
      };

      const vm = buildParameterStepViewModel(metadata);

      expect(vm.rootGroupForm).toEqual({
        id: 'root',
        label: '',
        description: '',
        hint: '',
        hintType: ParameterHintType.None,
        type: ParameterInputType.Group,
        collapsible: false,
        collapsedByDefault: false,
        children: formFields,
      });
      expect(vm.queries).toEqual(metadata.query);
      expect(vm.plan).toEqual(metadata.plan);
      expect(vm.job).toEqual(metadata.jobCommand);
      expect(vm.fieldCount).toBe(2);
      expect(vm.totalEstimatedSummary).toEqual({
        knownCount: 100,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~100 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });
  });
});
