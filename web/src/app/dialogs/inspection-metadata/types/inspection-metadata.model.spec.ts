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

import { convertToInspectionMetadataViewModel } from './inspection-metadata.model';
import { InspectionMetadataOfRunResult } from 'src/app/common/schema/api-types';

describe('inspection-metadata.model', () => {
  describe('convertToInspectionMetadataViewModel', () => {
    it('should correctly convert empty or default metadata', () => {
      const emptyRaw: InspectionMetadataOfRunResult = {
        header: {
          inspectionType: '',
          inspectionName: '',
          inspectionTypeIconPath: '',
          startTimeUnixSeconds: 0,
          endTimeUnixSeconds: 0,
          inspectTimeUnixSeconds: 0,
          suggestedFilename: '',
          fileSize: 0,
        },
        query: [],
        log: [],
        plan: { taskGraph: '' },
        error: { errorMessages: [] },
      };

      const vm = convertToInspectionMetadataViewModel(emptyRaw, 0);
      expect(vm.overview.inspectionType).toBe('Unknown');
      expect(vm.overview.inspectionName).toBe('Untitled Inspection');
      expect(vm.overview.fileSizeText).toBe('0 B');
      expect(vm.overview.durationText).toBe('0s');
      expect(vm.overview.formattedStartTime).toBe('-');
      expect(vm.overview.formattedEndTime).toBe('-');
      expect(vm.queries).toEqual([]);
      expect(vm.logs).toEqual([]);
      expect(vm.plan.taskGraph).toBe('');
      expect(vm.errors).toEqual([]);
    });

    it('should format duration correctly for various time ranges', () => {
      const createWithTimes = (
        start: number,
        end: number,
      ): InspectionMetadataOfRunResult => ({
        header: {
          inspectionType: 'test',
          inspectionName: 'Test',
          inspectionTypeIconPath: '',
          startTimeUnixSeconds: start,
          endTimeUnixSeconds: end,
          inspectTimeUnixSeconds: start,
          suggestedFilename: 'test.khi',
        },
        query: [],
        log: [],
        plan: { taskGraph: '' },
        error: { errorMessages: [] },
      });

      // 45 seconds duration
      const vm45s = convertToInspectionMetadataViewModel(
        createWithTimes(1000, 1045),
        0,
      );
      expect(vm45s.overview.durationText).toBe('45s');

      // 1m 30s duration
      const vm90s = convertToInspectionMetadataViewModel(
        createWithTimes(1000, 1090),
        0,
      );
      expect(vm90s.overview.durationText).toBe('1m 30s');

      // 1h 1m 5s duration
      const vm1h = convertToInspectionMetadataViewModel(
        createWithTimes(1000, 1000 + 3665),
        0,
      );
      expect(vm1h.overview.durationText).toBe('1h 1m 5s');

      // Invalid or zero start time
      const vmZeroStart = convertToInspectionMetadataViewModel(
        createWithTimes(0, 100),
        0,
      );
      expect(vmZeroStart.overview.durationText).toBe('0s');
      expect(vmZeroStart.overview.formattedStartTime).toBe('-');

      // End time before start time
      const vmNegativeDuration = convertToInspectionMetadataViewModel(
        createWithTimes(1000, 900),
        0,
      );
      expect(vmNegativeDuration.overview.durationText).toBe('0s');
    });

    it('should convert complete metadata with all fields populated', () => {
      const raw: InspectionMetadataOfRunResult = {
        header: {
          inspectionType: 'gcp-gke',
          inspectionName: 'Cluster Audit',
          inspectionTypeIconPath: 'icons/gke.svg',
          startTimeUnixSeconds: 1700000000,
          endTimeUnixSeconds: 1700003600,
          inspectTimeUnixSeconds: 1700000100,
          suggestedFilename: 'cluster-audit.khi',
          fileSize: 2048576,
        },
        query: [
          {
            id: 'q1',
            name: 'Audit Logs',
            query: 'resource.type="k8s_cluster"',
            estimatedCount: 1200,
          },
        ],
        log: [
          {
            id: 'task-1',
            name: 'GKE Task',
            log: 'Starting query...',
          },
        ],
        plan: {
          taskGraph: 'digraph G { A -> B; }',
        },
        error: {
          errorMessages: [
            {
              errorId: 'ERR_PERMISSION',
              message: 'Permission denied',
              link: 'https://cloud.google.com/docs',
            },
          ],
        },
        jobCommand: {
          command: './khi --job-mode',
        },
      };

      const vm = convertToInspectionMetadataViewModel(raw, 9);
      expect(vm.overview.inspectionType).toBe('gcp-gke');
      expect(vm.overview.inspectionName).toBe('Cluster Audit');
      expect(vm.overview.durationText).toBe('1h');
      expect(vm.overview.fileSizeText).toBe('2.0 MB');
      expect(vm.overview.formattedStartTime).toBe('2023-11-15T07:13:20+09:00');
      expect(vm.overview.formattedEndTime).toBe('2023-11-15T08:13:20+09:00');
      expect(vm.overview.suggestedFilename).toBe('cluster-audit.khi');
      expect(vm.queries.length).toBe(1);
      expect(vm.queries[0].name).toBe('Audit Logs');
      expect(vm.logs.length).toBe(1);
      expect(vm.logs[0].log).toBe('Starting query...');
      expect(vm.plan.taskGraph).toBe('digraph G { A -> B; }');
      expect(vm.errors.length).toBe(1);
      expect(vm.errors[0].errorId).toBe('ERR_PERMISSION');
      expect(vm.jobCommand).toBe('./khi --job-mode');
    });
  });
});
