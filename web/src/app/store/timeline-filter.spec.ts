/**
 * Copyright 2024 Google LLC
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

import { ParentRelationship } from '../generated';
import { ResourceTimeline } from './timeline';
import { TimelineFilterFacade } from './timeline-filter';

interface TimelineFilterTestCase {
  name: string;
  resourcePath: string;
  expected: boolean;
}

describe('TimelineFilter#isNodeOrNodeChildren', () => {
  const testCases: TimelineFilterTestCase[] = [
    {
      name: 'returns true on node resource',
      resourcePath: 'core/v1#node#cluster-scope#node-1',
      expected: true,
    },
    {
      name: 'returns false on non-node resource',
      resourcePath: 'core/v1#pod#kube-system#kubedns',
      expected: false,
    },
  ];
  for (const testCase of testCases) {
    it(testCase.name, () => {
      const timeline = new ResourceTimeline(
        'test',
        testCase.resourcePath,
        [],
        [],
        ParentRelationship.RelationshipChild,
      );
      expect(TimelineFilterFacade.isNodeOrNodeChildren(timeline)).toBe(
        testCase.expected,
      );
    });
  }
});

describe('TimelineFilter#isPodOrChildren', () => {
  const testCases: TimelineFilterTestCase[] = [
    {
      name: 'returns true on pod resource',
      resourcePath: 'core/v1#pod#kube-system#kubedns',
      expected: true,
    },
    {
      name: 'returns false on non-pod resource',
      resourcePath: 'core/v1#services#kube-system#kubedns',
      expected: false,
    },
  ];
  for (const testCase of testCases) {
    it(testCase.name, () => {
      const timeline = new ResourceTimeline(
        'test',
        testCase.resourcePath,
        [],
        [],
        ParentRelationship.RelationshipChild,
      );
      expect(TimelineFilterFacade.isPodOrPodChildren(timeline)).toBe(
        testCase.expected,
      );
    });
  }
});

describe('TimelineFilter#isPodBindingForNode', () => {
  const testCases: TimelineFilterTestCase[] = [
    {
      name: 'returns true on pod binding resource',
      resourcePath: 'core/v1#pod#kube-system#kubedns#binding',
      expected: true,
    },
    {
      name: 'returns false on non-pod binding resource',
      resourcePath: 'core/v1#pod#kube-system#kubedns#eviction',
      expected: false,
    },
    {
      name: 'returns false on non-pod resource',
      resourcePath: 'core/v1#services#kube-system#kubedns',
      expected: false,
    },
  ];
  for (const testCase of testCases) {
    it(testCase.name, () => {
      const timeline = new ResourceTimeline(
        'test',
        testCase.resourcePath,
        [],
        [],
        ParentRelationship.RelationshipChild,
      );
      expect(TimelineFilterFacade.isPodBindingForNode(timeline)).toBe(
        testCase.expected,
      );
    });
  }
});
