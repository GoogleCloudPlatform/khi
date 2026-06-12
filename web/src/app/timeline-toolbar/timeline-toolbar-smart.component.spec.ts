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
  compileLogFiltersToCel,
  compileFiltersToCel,
} from './timeline-toolbar-smart.component';
import { TimelineFilterConfig } from 'src/app/timeline-toolbar/types/filter-config';

describe('TimelineToolbarSmart compilation helpers', () => {
  describe('compileLogFiltersToCel', () => {
    it('should return empty string when severity is ANY and searchQuery is empty', () => {
      expect(compileLogFiltersToCel('ANY', '')).toBe('');
      expect(compileLogFiltersToCel('ANY', '   ')).toBe('');
    });

    it('should compile severity filter when severity is not ANY', () => {
      expect(compileLogFiltersToCel('INFO', '')).toBe('severity >= INFO');
      expect(compileLogFiltersToCel('ERROR', '')).toBe('severity >= ERROR');
    });

    it('should compile search query filter when searchQuery is provided', () => {
      expect(compileLogFiltersToCel('ANY', 'hello')).toBe('body("hello")');
    });

    it('should join severity and search query with &&', () => {
      expect(compileLogFiltersToCel('WARNING', 'my-query')).toBe(
        'severity >= WARNING && body("my-query")',
      );
    });

    it('should escape double quotes and backslashes in search query', () => {
      expect(compileLogFiltersToCel('ANY', 'hello "world"')).toBe(
        'body("hello \\"world\\"")',
      );
      expect(compileLogFiltersToCel('ANY', 'path\\to\\file')).toBe(
        'body("path\\\\to\\\\file")',
      );
    });
  });

  describe('compileFiltersToCel', () => {
    it('should return empty string when no filters are provided', () => {
      expect(compileFiltersToCel([])).toBe('');
    });

    it('should compile regex filter with * type', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: '*',
          mode: 'regex',
          value: 'pod-.*',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe('match("pod-.*")');
    });

    it('should compile regex filter with specific type', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'regex',
          value: 'pod-.*',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "pod-.*")',
      );
    });

    it('should escape quotes and backslashes in regex mode', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'regex',
          value: 'test"val\\ue',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "test\\"val\\\\ue")',
      );
    });

    it('should compile selection filter by escaping special regex characters and wrapping in anchor', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'selection',
          value: 'pod-a|pod.b|pod+c',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "^(?:pod-a|pod\\.b|pod\\+c)$")',
      );
    });

    it('should escape quotes and backslashes in selection mode', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'selection',
          value: 'val"ue\\1|val"ue\\2',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "^(?:val\\"ue\\\\\\\\1|val\\"ue\\\\\\\\2)$")',
      );
    });

    it('should join multiple filters with &&', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'regex',
          value: 'pod-.*',
        },
        {
          id: '2',
          timelineType: '*',
          mode: 'selection',
          value: 'ns-1|ns-2',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "pod-.*") && match("^(?:ns-1|ns-2)$")',
      );
    });

    it('should prepend minSeverity when severity is not ANY', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: '*',
          mode: 'regex',
          value: 'pod-.*',
        },
      ];
      expect(compileFiltersToCel(filters, 'ERROR')).toBe(
        'minSeverity(ERROR) && match("pod-.*")',
      );
    });

    it('should return minSeverity only when filters is empty and severity is not ANY', () => {
      expect(compileFiltersToCel([], 'ERROR')).toBe('minSeverity(ERROR)');
    });
  });
});
