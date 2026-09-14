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
  formatBytes,
  formatDuration,
  formatTimestampSeconds,
  convertToInspectionMetadataViewModel,
} from './inspection-metadata.model';
import { InspectionMetadataOfRunResult } from 'src/app/common/schema/api-types';

describe('inspection-metadata.model', () => {
  describe('formatBytes', () => {
    it('should format 0 and negative bytes as 0 B', () => {
      expect(formatBytes(0)).toBe('0 B');
      expect(formatBytes(-100)).toBe('0 B');
    });

    it('should format small bytes as B', () => {
      expect(formatBytes(512)).toBe('512 B');
    });

    it('should format kilobytes and megabytes', () => {
      expect(formatBytes(1024)).toBe('1.0 KB');
      expect(formatBytes(1536)).toBe('1.5 KB');
      expect(formatBytes(1048576)).toBe('1.0 MB');
      expect(formatBytes(10485760)).toBe('10 MB');
    });
  });

  describe('formatDuration', () => {
    it('should format 0 or negative seconds as 0s', () => {
      expect(formatDuration(0)).toBe('0s');
      expect(formatDuration(-10)).toBe('0s');
    });

    it('should format seconds', () => {
      expect(formatDuration(45)).toBe('45s');
    });

    it('should format minutes and seconds', () => {
      expect(formatDuration(90)).toBe('1m 30s');
    });

    it('should format hours, minutes and seconds', () => {
      expect(formatDuration(3665)).toBe('1h 1m 5s');
    });
  });

  describe('formatTimestampSeconds', () => {
    it('should return - for 0 or negative timestamp', () => {
      expect(formatTimestampSeconds(0)).toBe('-');
      expect(formatTimestampSeconds(-1)).toBe('-');
    });

    it('should return localized string for valid timestamp', () => {
      const result = formatTimestampSeconds(1700000000);
      expect(result).not.toBe('-');
      expect(result.length).toBeGreaterThan(0);
    });
  });

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

      const vm = convertToInspectionMetadataViewModel(emptyRaw);
      expect(vm.overview.inspectionType).toBe('Unknown');
      expect(vm.overview.inspectionName).toBe('Untitled Inspection');
      expect(vm.overview.fileSizeText).toBe('0 B');
      expect(vm.overview.durationText).toBe('0s');
      expect(vm.queries).toEqual([]);
      expect(vm.logs).toEqual([]);
      expect(vm.plan.taskGraph).toBe('');
      expect(vm.errors).toEqual([]);
    });

    it('should convert complete metadata', () => {
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
      };

      const vm = convertToInspectionMetadataViewModel(raw);
      expect(vm.overview.inspectionType).toBe('gcp-gke');
      expect(vm.overview.inspectionName).toBe('Cluster Audit');
      expect(vm.overview.durationText).toBe('1h');
      expect(vm.overview.suggestedFilename).toBe('cluster-audit.khi');
      expect(vm.queries.length).toBe(1);
      expect(vm.queries[0].name).toBe('Audit Logs');
      expect(vm.logs.length).toBe(1);
      expect(vm.logs[0].log).toBe('Starting query...');
      expect(vm.plan.taskGraph).toBe('digraph G { A -> B; }');
      expect(vm.errors.length).toBe(1);
      expect(vm.errors[0].errorId).toBe('ERR_PERMISSION');
    });
  });
});
