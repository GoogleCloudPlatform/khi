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
  CELTimelineFilterEnvironment,
  CELLogFilterEnvironment,
} from 'src/app/store/domain/filter/cel-env';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { Timeline } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';

describe('CELTimelineFilterEnvironment', () => {
  let env: CELTimelineFilterEnvironment;

  beforeEach(() => {
    env = new CELTimelineFilterEnvironment();
  });

  it('should successfully compile a valid CEL expression', () => {
    const result = env.compile("name == 'T1'");
    expect(result.success).toBe(true);
    expect(result.error).toBeUndefined();
  });

  it('should return success and disable evaluation for empty or whitespace expression', () => {
    const resultEmpty = env.compile('');
    expect(resultEmpty.success).toBe(true);

    const resultWhitespace = env.compile('   ');
    expect(resultWhitespace.success).toBe(true);

    // Should pass-through all elements
    const mockTimeline = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [],
      events: [],
      revisions: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    expect(env.evaluate(mockTimeline)).toBe(true);
  });

  it('should return failure for syntactically invalid CEL expression', () => {
    const result = env.compile("name == 'T1");
    expect(result.success).toBe(false);
    expect(result.error).toBeDefined();
  });

  it('should evaluate valid CEL expression with basic variables correctly', () => {
    const timeline1 = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [],
      events: [],
      revisions: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    const timeline2 = {
      name: 'T2',
      type: { label: 'Node' },
      path: [],
      events: [],
      revisions: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    env.compile("name == 'T1' && timelineType == 'pod'");

    expect(env.evaluate(timeline1)).toBe(true);
    expect(env.evaluate(timeline2)).toBe(false);
  });

  it('should resolve timeline path map correctly', () => {
    const timeline = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [
        { type: { label: 'Namespace' }, label: 'default' },
        { type: { label: 'Name' }, label: 'pod-a' },
      ],
      events: [],
      revisions: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    env.compile("path['namespace'] == 'default' && path['name'] == 'pod-a'");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("path['namespace'] == 'kube-system'");
    expect(env.evaluate(timeline)).toBe(false);
  });

  it('should support severity constants during evaluation', () => {
    env.compile(
      'ERROR == 3 && WARNING == 2 && INFO == 1 && UNKNOWN == 0 && FATAL == 4',
    );
    const timeline = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [],
      events: [],
      revisions: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    expect(env.evaluate(timeline)).toBe(true);
  });

  it('should support match() and M() functions with a single pattern', () => {
    const timeline = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [
        { type: { label: 'Namespace' }, label: 'kube-system' },
        { type: { label: 'Name' }, label: 'kube-apiserver' },
      ],
      events: [],
      revisions: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    // Match with key and pattern
    env.compile("match('Namespace', 'kube-.*')");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("M('Namespace', 'default')");
    expect(env.evaluate(timeline)).toBe(false);

    // Wildcard match
    env.compile("match('apiserver')");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("M('default')");
    expect(env.evaluate(timeline)).toBe(false);
  });

  it('should support match() and M() functions with a list of patterns', () => {
    const timeline = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [{ type: { label: 'Namespace' }, label: 'kube-system' }],
      events: [],
      revisions: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    env.compile("match('Namespace', ['default', 'kube-.*'])");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("M('Namespace', ['default', 'non-matching'])");
    expect(env.evaluate(timeline)).toBe(false);

    env.compile("match(['default', 'system'])");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("M(['default', 'non-matching'])");
    expect(env.evaluate(timeline)).toBe(false);
  });

  it('should support revision_body() and RB() functions with a single pattern', () => {
    const timeline = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [],
      events: [],
      revisions: [
        {
          id: 1,
          index: 0,
          changedTime: 0n,
          principal: 'user',
          verb: { label: 'create' },
          state: { label: 'success' },
          log: {
            logType: { label: 'k8s' },
            severity: { label: 'INFO' },
            summary: 'L1',
            body: {},
            bodyYAML: '',
          },
          body: { spec: { replicas: 3 } },
          bodyYAML: 'spec:\n  replicas: 3\n',
        },
      ],
    } as unknown as ReadonlyDomainElement<Timeline>;

    env.compile("revision_body('spec.replicas', '3')");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("RB('spec.replicas', '5')");
    expect(env.evaluate(timeline)).toBe(false);

    // Wildcard match
    env.compile("revision_body('replicas')");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("RB('non-existent')");
    expect(env.evaluate(timeline)).toBe(false);
  });

  it('should support revision_body() and RB() functions with a list of patterns', () => {
    const timeline = {
      name: 'T1',
      type: { label: 'Pod' },
      path: [],
      events: [],
      revisions: [
        {
          id: 1,
          index: 0,
          changedTime: 0n,
          principal: 'user',
          verb: { label: 'create' },
          state: { label: 'success' },
          log: {
            logType: { label: 'k8s' },
            severity: { label: 'INFO' },
            summary: 'L1',
            body: {},
            bodyYAML: '',
          },
          body: { spec: { replicas: 3 } },
          bodyYAML: 'spec:\n  replicas: 3\n',
        },
      ],
    } as unknown as ReadonlyDomainElement<Timeline>;

    env.compile("revision_body('spec.replicas', ['1', '3'])");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("RB('spec.replicas', ['1', '5'])");
    expect(env.evaluate(timeline)).toBe(false);

    // Wildcard list match
    env.compile("revision_body(['replicas', 'other'])");
    expect(env.evaluate(timeline)).toBe(true);

    env.compile("RB(['other', 'non-matching'])");
    expect(env.evaluate(timeline)).toBe(false);
  });
});

describe('CELLogFilterEnvironment', () => {
  let env: CELLogFilterEnvironment;

  beforeEach(() => {
    env = new CELLogFilterEnvironment();
  });

  it('should successfully compile a valid CEL expression', () => {
    const result = env.compile("summary == 'L1'");
    expect(result.success).toBe(true);
    expect(result.error).toBeUndefined();
  });

  it('should return success and disable evaluation for empty or whitespace expression', () => {
    const resultEmpty = env.compile('');
    expect(resultEmpty.success).toBe(true);

    const resultWhitespace = env.compile('   ');
    expect(resultWhitespace.success).toBe(true);

    const mockLog = {
      logType: { label: 'k8s' },
      severity: { label: 'INFO' },
      summary: 'L1',
      body: {},
      bodyYAML: '',
    } as ReadonlyDomainElement<Log>;

    expect(env.evaluate(mockLog)).toBe(true);
  });

  it('should return failure for syntactically invalid CEL expression', () => {
    const result = env.compile("summary == 'L1");
    expect(result.success).toBe(false);
    expect(result.error).toBeDefined();
  });

  it('should evaluate valid CEL expression with basic variables correctly', () => {
    const log1 = {
      logType: { label: 'k8s' },
      severity: { label: 'INFO' },
      summary: 'L1',
      body: {},
      bodyYAML: '',
    } as ReadonlyDomainElement<Log>;

    const log2 = {
      logType: { label: 'audit' },
      severity: { label: 'WARNING' },
      summary: 'L2',
      body: {},
      bodyYAML: '',
    } as ReadonlyDomainElement<Log>;

    env.compile("logType == 'k8s' && severity == INFO");
    expect(env.evaluate(log1)).toBe(true);
    expect(env.evaluate(log2)).toBe(false);
  });

  it('should resolve log body and bodyYAML correctly', () => {
    const log = {
      logType: { label: 'k8s' },
      severity: { label: 'INFO' },
      summary: 'L1',
      body: { verb: 'create' },
      bodyYAML: 'verb: create\n',
    } as unknown as ReadonlyDomainElement<Log>;

    env.compile("body['verb'] == 'create'");
    expect(env.evaluate(log)).toBe(true);

    env.compile("bodyYAML.contains('verb: create')");
    expect(env.evaluate(log)).toBe(true);
  });

  it('should support body() and B() functions with single patterns', () => {
    const log = {
      logType: { label: 'k8s' },
      severity: { label: 'INFO' },
      summary: 'L1',
      body: { metadata: { name: 'pod-a' } },
      bodyYAML: 'metadata:\n  name: pod-a\n',
    } as unknown as ReadonlyDomainElement<Log>;

    env.compile("body('metadata.name', 'pod-.*')");
    expect(env.evaluate(log)).toBe(true);

    env.compile("B('metadata.name', 'pod-b')");
    expect(env.evaluate(log)).toBe(false);

    // Wildcard match
    env.compile("body('pod-a')");
    expect(env.evaluate(log)).toBe(true);

    env.compile("B('pod-b')");
    expect(env.evaluate(log)).toBe(false);
  });

  it('should support body() and B() functions with a list of patterns', () => {
    const log = {
      logType: { label: 'k8s' },
      severity: { label: 'INFO' },
      summary: 'L1',
      body: { metadata: { name: 'pod-a' } },
      bodyYAML: 'metadata:\n  name: pod-a\n',
    } as unknown as ReadonlyDomainElement<Log>;

    env.compile("body('metadata.name', ['pod-b', 'pod-.*'])");
    expect(env.evaluate(log)).toBe(true);

    env.compile("B('metadata.name', ['pod-b', 'pod-c'])");
    expect(env.evaluate(log)).toBe(false);

    // Wildcard list match
    env.compile("body(['pod-a', 'other'])");
    expect(env.evaluate(log)).toBe(true);

    env.compile("B(['pod-b', 'other'])");
    expect(env.evaluate(log)).toBe(false);
  });
});
