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
  mapSeverityToNumber,
  matchTimelinePath,
  matchLogField,
  matchTimelineRevisionBodyField,
} from 'src/app/store/domain/filter/cel-functions';
import {
  CELLog,
  CELTimeline,
  CELRevision,
} from 'src/app/store/domain/filter/cel-types';

describe('cel-functions', () => {
  describe('mapSeverityToNumber', () => {
    it('should return UNKNOWN (0n) for undefined, empty or unrelated labels', () => {
      expect(mapSeverityToNumber(undefined)).toBe(0n);
      expect(mapSeverityToNumber('')).toBe(0n);
      expect(mapSeverityToNumber('DEBUG')).toBe(0n);
      expect(mapSeverityToNumber('UNKNOWN')).toBe(0n);
    });

    it('should return INFO (1n) for labels containing INFO', () => {
      expect(mapSeverityToNumber('INFO')).toBe(1n);
      expect(mapSeverityToNumber('info')).toBe(1n);
      expect(mapSeverityToNumber('LOG_INFO')).toBe(1n);
    });

    it('should return WARNING (2n) for labels containing WARN', () => {
      expect(mapSeverityToNumber('WARN')).toBe(2n);
      expect(mapSeverityToNumber('WARNING')).toBe(2n);
      expect(mapSeverityToNumber('warning')).toBe(2n);
    });

    it('should return ERROR (3n) for labels containing ERROR', () => {
      expect(mapSeverityToNumber('ERROR')).toBe(3n);
      expect(mapSeverityToNumber('error')).toBe(3n);
      expect(mapSeverityToNumber('ERR')).toBe(0n); // 'ERR' does not contain 'ERROR'
    });

    it('should return FATAL (4n) for labels containing FATAL', () => {
      expect(mapSeverityToNumber('FATAL')).toBe(4n);
      expect(mapSeverityToNumber('fatal')).toBe(4n);
    });
  });

  describe('matchTimelinePath', () => {
    const mockTimeline: CELTimeline = {
      name: 'test-timeline',
      timelineType: 'pod',
      path: {
        namespace: 'kube-system',
        name: 'kube-apiserver-kind-control-plane',
        kind: 'Pod',
      },
      events: [],
      revisions: [],
    };

    it('should return false if timeline is undefined', () => {
      expect(matchTimelinePath(undefined, 'namespace', 'kube-system')).toBe(
        false,
      );
    });

    it('should return false if timeline path is undefined', () => {
      const timelineNoPath = {
        ...mockTimeline,
        path: undefined as unknown as Record<string, string>,
      };
      expect(
        matchTimelinePath(timelineNoPath, 'namespace', 'kube-system'),
      ).toBe(false);
    });

    it('should match a specific key with a single value pattern case-insensitively', () => {
      expect(matchTimelinePath(mockTimeline, 'namespace', 'kube-system')).toBe(
        true,
      );
      expect(matchTimelinePath(mockTimeline, 'Namespace', 'KUBE-system')).toBe(
        true,
      );
      expect(matchTimelinePath(mockTimeline, 'kind', '^pod$')).toBe(true);
      expect(matchTimelinePath(mockTimeline, 'kind', 'non-matching')).toBe(
        false,
      );
    });

    it('should match a specific key with multiple value patterns', () => {
      expect(
        matchTimelinePath(mockTimeline, 'namespace', ['default', 'kube-.*']),
      ).toBe(true);
      expect(
        matchTimelinePath(mockTimeline, 'namespace', [
          'default',
          'kube-apiserver',
        ]),
      ).toBe(false);
    });

    it('should match any key when key is *', () => {
      expect(matchTimelinePath(mockTimeline, '*', 'kube-.*')).toBe(true);
      expect(matchTimelinePath(mockTimeline, '*', 'non-matching')).toBe(false);
    });

    it('should return false if the key is not in path', () => {
      expect(matchTimelinePath(mockTimeline, 'nonexistent', '.*')).toBe(false);
    });

    it('should return false if regex pattern is invalid', () => {
      expect(
        matchTimelinePath(mockTimeline, 'namespace', '[invalid-regex'),
      ).toBe(false);
    });
  });

  describe('matchLogField', () => {
    const mockLog: CELLog = {
      logType: 'k8s',
      severity: 1n,
      summary: 'test-log',
      body: {
        verb: 'CREATE',
        object: {
          metadata: {
            name: 'test-pod',
            uid: '12345',
          },
          spec: {
            containers: ['nginx'],
            replicas: 3,
            active: true,
          },
        },
      },
      bodyYAML: 'verb: CREATE\nobject:\n  metadata:\n    name: test-pod\n',
    };

    it('should return false if log is undefined', () => {
      expect(matchLogField(undefined, 'verb', 'CREATE')).toBe(false);
    });

    it('should match against bodyYAML if pathKey is *', () => {
      expect(matchLogField(mockLog, '*', 'metadata')).toBe(true);
      expect(matchLogField(mockLog, '*', 'METADATA')).toBe(true);
      expect(matchLogField(mockLog, '*', 'non-existent')).toBe(false);
    });

    it('should match nested properties using dot-separated paths', () => {
      expect(matchLogField(mockLog, 'verb', 'create')).toBe(true);
      expect(matchLogField(mockLog, 'object.metadata.name', 'test-pod')).toBe(
        true,
      );
      expect(matchLogField(mockLog, 'object.spec.replicas', '3')).toBe(true);
      expect(matchLogField(mockLog, 'object.spec.active', 'true')).toBe(true);
    });

    it('should return false if path traversal hits null, undefined or non-object', () => {
      expect(matchLogField(mockLog, 'object.metadata.nonexistent', '.*')).toBe(
        false,
      );
      expect(matchLogField(mockLog, 'object.spec.containers.0', '.*')).toBe(
        false,
      );
      expect(matchLogField(mockLog, 'object.metadata.name.subpath', '.*')).toBe(
        false,
      );
    });

    it('should return false if leaf value is null or undefined', () => {
      const logWithNullBody: CELLog = {
        ...mockLog,
        body: {
          nullValue: null,
          undefinedValue: undefined,
        },
      };
      expect(matchLogField(logWithNullBody, 'nullValue', '.*')).toBe(false);
      expect(matchLogField(logWithNullBody, 'undefinedValue', '.*')).toBe(
        false,
      );
    });

    it('should match with multiple patterns', () => {
      expect(
        matchLogField(mockLog, 'object.metadata.name', ['wrong-name', '.*pod']),
      ).toBe(true);
      expect(
        matchLogField(mockLog, 'object.metadata.name', [
          'wrong-name',
          'another-wrong',
        ]),
      ).toBe(false);
    });

    it('should return false if regex pattern is invalid', () => {
      expect(matchLogField(mockLog, 'verb', '[invalid-regex')).toBe(false);
    });
  });

  describe('matchTimelineRevisionBodyField', () => {
    const mockRevisions: readonly CELRevision[] = [
      {
        log: {
          logType: 'k8s',
          severity: 1n,
          summary: 'rev-1',
          body: {},
          bodyYAML: '',
        },
        changedTime: 1000n,
        principal: 'user-1',
        verb: 'CREATE',
        state: 'success',
        body: {
          spec: {
            replicas: 1,
          },
        },
        bodyYAML: 'spec:\n  replicas: 1\n',
      },
      {
        log: {
          logType: 'k8s',
          severity: 1n,
          summary: 'rev-2',
          body: {},
          bodyYAML: '',
        },
        changedTime: 2000n,
        principal: 'user-2',
        verb: 'UPDATE',
        state: 'success',
        body: {
          spec: {
            replicas: 2,
          },
        },
        bodyYAML: 'spec:\n  replicas: 2\n',
      },
    ];

    const mockTimeline: CELTimeline = {
      name: 'test-timeline',
      timelineType: 'pod',
      path: {},
      events: [],
      revisions: mockRevisions,
    };

    it('should return false if timeline is undefined', () => {
      expect(
        matchTimelineRevisionBodyField(undefined, 'spec.replicas', '1'),
      ).toBe(false);
    });

    it('should match any revision bodyYAML if pathKey is *', () => {
      expect(
        matchTimelineRevisionBodyField(mockTimeline, '*', 'replicas: 1'),
      ).toBe(true);
      expect(
        matchTimelineRevisionBodyField(mockTimeline, '*', 'replicas: 2'),
      ).toBe(true);
      expect(
        matchTimelineRevisionBodyField(mockTimeline, '*', 'replicas: 3'),
      ).toBe(false);
    });

    it('should match nested properties in any revision body', () => {
      expect(
        matchTimelineRevisionBodyField(mockTimeline, 'spec.replicas', '1'),
      ).toBe(true);
      expect(
        matchTimelineRevisionBodyField(mockTimeline, 'spec.replicas', '2'),
      ).toBe(true);
      expect(
        matchTimelineRevisionBodyField(mockTimeline, 'spec.replicas', '3'),
      ).toBe(false);
    });

    it('should match with multiple patterns', () => {
      expect(
        matchTimelineRevisionBodyField(mockTimeline, 'spec.replicas', [
          '3',
          '[1-2]',
        ]),
      ).toBe(true);
    });

    it('should return false if path traversal fails in all revisions', () => {
      expect(
        matchTimelineRevisionBodyField(mockTimeline, 'spec.nonexistent', '.*'),
      ).toBe(false);
    });

    it('should return false if regex pattern is invalid', () => {
      expect(
        matchTimelineRevisionBodyField(
          mockTimeline,
          'spec.replicas',
          '[invalid-regex',
        ),
      ).toBe(false);
    });
  });
});
