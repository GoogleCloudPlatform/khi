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

import { ManagedFieldsAnnotationProvider } from 'src/app/shared/components/yaml-viewer/managed-fields-annotation.provider';
import { ManagedFieldTooltipComponent } from 'src/app/shared/components/yaml-viewer/components/managed-field-tooltip.component';
import { AnnotationSeverity } from 'src/app/shared/components/yaml-viewer/yaml-annotation';

describe('ManagedFieldsAnnotationProvider', () => {
  let provider: ManagedFieldsAnnotationProvider;
  const TIME_STRING = '2026-07-07T12:00:00Z';
  const TIME_NS = BigInt(new Date(TIME_STRING).getTime()) * 1000000n;
  const TIMEZONE_SHIFT = 9;

  beforeEach(() => {
    provider = new ManagedFieldsAnnotationProvider(TIMEZONE_SHIFT);
  });

  it('should extract paths for f: prefixes correctly', () => {
    const parsedYaml = {
      metadata: {
        name: 'test',
        managedFields: [
          {
            manager: 'test-manager',
            time: TIME_NS,
            operation: 'Update',
            fieldsV1: {
              'f:metadata': {
                'f:name': {},
              },
              'f:spec': {
                'f:replicas': {},
              },
            },
          },
        ],
      },
    };

    const result = provider.getAnnotations(parsedYaml);

    expect(result).toEqual([
      {
        path: ['metadata', 'name'],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
      {
        path: ['spec', 'replicas'],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
    ]);
  });

  it('should extract paths for . prefixes correctly', () => {
    const parsedYaml = {
      metadata: {
        labels: {
          app: 'test',
        },
        managedFields: [
          {
            manager: 'test-manager',
            time: TIME_NS,
            operation: 'Update',
            fieldsV1: {
              'f:metadata': {
                'f:labels': {
                  '.': {},
                  'f:app': {},
                },
              },
            },
          },
        ],
      },
    };

    const result = provider.getAnnotations(parsedYaml);

    expect(result).toEqual([
      {
        path: ['metadata', 'labels'],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
      {
        path: ['metadata', 'labels', 'app'],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
    ]);
  });

  it('should extract paths for k: prefixes (map list) correctly', () => {
    const parsedYaml = {
      status: {
        conditions: [
          { type: 'Ready', status: 'True' },
          { type: 'NetworkUnavailable', status: 'False' },
        ],
      },
      metadata: {
        managedFields: [
          {
            manager: 'test-manager',
            time: TIME_NS,
            operation: 'Update',
            fieldsV1: {
              'f:status': {
                'f:conditions': {
                  'k:{"type":"NetworkUnavailable"}': {
                    '.': {},
                    'f:status': {},
                  },
                },
              },
            },
          },
        ],
      },
    };

    const result = provider.getAnnotations(parsedYaml);

    expect(result).toEqual([
      {
        path: ['status', 'conditions', 1],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
      {
        path: ['status', 'conditions', 1, 'status'],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
    ]);
  });

  it('should extract paths for v: prefixes (set list) correctly', () => {
    const parsedYaml = {
      spec: {
        podCIDRs: ['10.0.0.0/24', '10.160.2.0/24'],
      },
      metadata: {
        managedFields: [
          {
            manager: 'test-manager',
            time: TIME_NS,
            operation: 'Update',
            fieldsV1: {
              'f:spec': {
                'f:podCIDRs': {
                  '.': {},
                  'v:"10.160.2.0/24"': {},
                },
              },
            },
          },
        ],
      },
    };

    const result = provider.getAnnotations(parsedYaml);

    expect(result).toEqual([
      {
        path: ['spec', 'podCIDRs'],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
      {
        path: ['spec', 'podCIDRs', 1],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
    ]);
  });

  it('should use overrideManagedFields if provided and not read from parsedYaml', () => {
    const providerWithOverride = new ManagedFieldsAnnotationProvider(
      TIMEZONE_SHIFT,
      [
        {
          manager: 'override-manager',
          time: TIME_NS,
          operation: 'Update',
          fieldsV1: {
            'f:spec': {
              'f:replicas': {},
            },
          },
        },
      ],
    );

    const parsedYaml = {
      metadata: {},
      spec: {
        replicas: 3,
      },
    };

    const result = providerWithOverride.getAnnotations(parsedYaml);

    expect(result).toEqual([
      {
        path: ['spec', 'replicas'],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'override-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
    ]);
  });

  it('should fallback to original string key if array element matching fails', () => {
    // Providing k: index but the array is empty in actual YAML
    const parsedYaml = {
      status: {
        conditions: [],
      },
      metadata: {
        managedFields: [
          {
            manager: 'test-manager',
            time: TIME_NS,
            operation: 'Update',
            fieldsV1: {
              'f:status': {
                'f:conditions': {
                  'k:{"type":"NetworkUnavailable"}': {
                    'f:status': {},
                  },
                },
              },
            },
          },
        ],
      },
    };

    const result = provider.getAnnotations(parsedYaml);

    expect(result).toEqual([
      {
        path: [
          'status',
          'conditions',
          'k:{"type":"NetworkUnavailable"}',
          'status',
        ],
        component: ManagedFieldTooltipComponent,
        inputs: {
          manager: 'test-manager',
          time: TIME_NS,
          timezoneShift: TIMEZONE_SHIFT,
        },
        severity: AnnotationSeverity.Low,
      },
    ]);
  });
});
