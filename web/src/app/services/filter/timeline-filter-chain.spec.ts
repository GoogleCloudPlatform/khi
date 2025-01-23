/**
 * Copyright 2025 Google LLC
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

import { TimelineEntry } from 'src/app/store/timeline';
import {
  FilterNamepaceOrKindWithoutResource,
  FilterSubresourceWithoutParent,
  FilterTimelinesOnlyWithFilteredLogs,
} from './timeline-filter-chain';
import { BehaviorSubject, of } from 'rxjs';
import {
  ParentRelationship,
  RevisionState,
  RevisionVerb,
} from 'src/app/generated';
import { ResourceRevision } from 'src/app/store/revision';

function generateTestTimeline(resourcePath: string): TimelineEntry {
  return new TimelineEntry(
    resourcePath,
    [],
    [],
    ParentRelationship.RelationshipChild,
  );
}

function generateTestTimelineWithLogIndices(
  resourcePath: string,
  logIndices: number[],
): TimelineEntry {
  return new TimelineEntry(
    resourcePath,
    logIndices.map(
      (index) =>
        new ResourceRevision(
          0,
          1,
          RevisionState.RevisionStateExisting,
          RevisionVerb.RevisionVerbCreate,
          '',
          '',
          false,
          false,
          index,
        ),
    ),
    [],
    ParentRelationship.RelationshipChild,
  );
}

describe('FilterNamepaceOrKindWithoutResource', () => {
  it('should remove kind/namespace timelines when there is no name layer timeline', () => {
    const filter = new FilterNamepaceOrKindWithoutResource();
    const kindTimeline = generateTestTimeline('apps/v1#Deployment');
    const namespaceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default',
    );
    const timelines = [kindTimeline, namespaceTimeline];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([]);
    });
  });

  it('should keep kind/namespace timelines when there is a name layer timeline', () => {
    const filter = new FilterNamepaceOrKindWithoutResource();
    const kindTimeline = generateTestTimeline('apps/v1#Deployment');
    const namespaceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default',
    );
    const nameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx',
    );

    const timelines = [kindTimeline, namespaceTimeline, nameTimeline];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual(timelines);
    });
  });

  it('should keep name layer timelines and later', () => {
    const filter = new FilterNamepaceOrKindWithoutResource();
    const kindTimeline = generateTestTimeline('apps/v1#Deployment');
    const namespaceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default',
    );
    const nameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx',
    );
    const subresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx#scale',
    );

    const timelines = [
      kindTimeline,
      namespaceTimeline,
      nameTimeline,
      subresourceTimeline,
    ];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual(timelines);
    });
  });

  it('should remove kind timelines when there is no namespace layer timelines', () => {
    const filter = new FilterNamepaceOrKindWithoutResource();
    const kindTimeline = generateTestTimeline('apps/v1#Deployment');
    const timelines = [kindTimeline];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([]);
    });
  });

  it('should filter multiple sets of timelines', () => {
    const filter = new FilterNamepaceOrKindWithoutResource();
    const deploymentKindTimeline = generateTestTimeline('apps/v1#Deployment');
    const deploymentNamespaceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default',
    );
    const deploymentNameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx',
    );
    const podKindTimeline = generateTestTimeline('apps/v1#Pod');
    const podNamespaceTimeline = generateTestTimeline('apps/v1#Pod#default');

    const timelines = [
      deploymentKindTimeline,
      deploymentNamespaceTimeline,
      deploymentNameTimeline,
      podKindTimeline,
      podNamespaceTimeline,
    ];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([
        deploymentKindTimeline,
        deploymentNamespaceTimeline,
        deploymentNameTimeline,
      ]);
    });
  });
});

describe('FilterSubresourceWithoutParent', () => {
  it('should remove subresource timelines when there is no name layer timeline', () => {
    const filter = new FilterSubresourceWithoutParent();
    const subresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx#scale',
    );
    const timelines = [subresourceTimeline];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([]);
    });
  });

  it('should keep subresource timelines when there is a name layer timeline', () => {
    const filter = new FilterSubresourceWithoutParent();
    const nameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx',
    );
    const subresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx#scale',
    );

    const timelines = [nameTimeline, subresourceTimeline];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual(timelines);
    });
  });

  it('should remove subresource timelines when parent resource name does not match', () => {
    const filter = new FilterSubresourceWithoutParent();
    const nameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx',
    );
    const subresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx-2#scale',
    );

    const timelines = [nameTimeline, subresourceTimeline];

    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([nameTimeline]);
    });
  });

  it('should handle multiple sets of timelines', () => {
    const filter = new FilterSubresourceWithoutParent();

    const nginxNameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx',
    );
    const nginxSubresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx#scale',
    );
    const nginx2NameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx-2',
    );
    const nginx2SubresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx-2#scale',
    );

    const timelines = [
      nginxNameTimeline,
      nginxSubresourceTimeline,
      nginx2NameTimeline,
      nginx2SubresourceTimeline,
    ];

    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([
        nginxNameTimeline,
        nginxSubresourceTimeline,
        nginx2NameTimeline,
        nginx2SubresourceTimeline,
      ]);
    });
  });

  it('should filter subresource timelines when its parent resource name does not match in complex cases', () => {
    const filter = new FilterSubresourceWithoutParent();

    const nginxNameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx',
    );
    const nginxSubresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx#scale',
    );
    const deploymentNamespaceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default',
    );
    const nginx2NameTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx-2',
    );
    const nginx2SubresourceTimeline = generateTestTimeline(
      'apps/v1#Deployment#default#nginx-3#scale',
    );

    const timelines = [
      nginxNameTimeline,
      nginxSubresourceTimeline,
      deploymentNamespaceTimeline,
      nginx2NameTimeline,
      nginx2SubresourceTimeline,
    ];

    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([
        nginxNameTimeline,
        nginxSubresourceTimeline,
        deploymentNamespaceTimeline,
        nginx2NameTimeline,
      ]);
    });
  });
});

describe('FilterTimelinesOnlyWithFilteredLogs', () => {
  it('should not filter timelines when there are no filtered logs', () => {
    const filteredOutLogIndicesSet = of(new Set<number>());
    const hideTimelinesWithoutMatchingLogs = new BehaviorSubject(true);
    const filter = new FilterTimelinesOnlyWithFilteredLogs(
      filteredOutLogIndicesSet,
      hideTimelinesWithoutMatchingLogs,
    );
    const timeline1 = generateTestTimelineWithLogIndices(
      'apps/v1#Deployment#default#nginx',
      [1, 2, 3],
    );
    const timeline2 = generateTestTimelineWithLogIndices(
      'apps/v1#Pod#default#my-pod',
      [4, 5],
    );

    const timelines = [timeline1, timeline2];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual(timelines);
    });
  });

  it('should filter out timelines with all logs filtered out', () => {
    const filteredOutLogIndicesSet = of(new Set<number>([1, 2, 3]));
    const hideTimelinesWithoutMatchingLogs = new BehaviorSubject(true);

    const filter = new FilterTimelinesOnlyWithFilteredLogs(
      filteredOutLogIndicesSet,
      hideTimelinesWithoutMatchingLogs,
    );
    const timeline1 = generateTestTimelineWithLogIndices(
      'apps/v1#Deployment#default#nginx',
      [1, 2, 3],
    );
    const timeline2 = generateTestTimelineWithLogIndices(
      'apps/v1#Pod#default#my-pod',
      [4, 5],
    );

    const timelines = [timeline1, timeline2];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([timeline2]);
    });
  });

  it('should keep timelines that has at least one log not filtered out.', () => {
    const filteredOutLogIndicesSet = of(new Set<number>([1, 2, 5]));
    const hideTimelinesWithoutMatchingLogs = new BehaviorSubject(true);

    const filter = new FilterTimelinesOnlyWithFilteredLogs(
      filteredOutLogIndicesSet,
      hideTimelinesWithoutMatchingLogs,
    );
    const timeline1 = generateTestTimelineWithLogIndices(
      'apps/v1#Deployment#default#nginx',
      [1, 2, 3],
    );
    const timeline2 = generateTestTimelineWithLogIndices(
      'apps/v1#Pod#default#my-pod',
      [4, 5],
    );

    const timelines = [timeline1, timeline2];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([timeline1, timeline2]);
    });
  });

  it('should not filter out timelines without name layer', () => {
    const filteredOutLogIndicesSet = of(new Set<number>([1, 2, 3]));
    const hideTimelinesWithoutMatchingLogs = new BehaviorSubject(true);

    const filter = new FilterTimelinesOnlyWithFilteredLogs(
      filteredOutLogIndicesSet,
      hideTimelinesWithoutMatchingLogs,
    );
    const timeline1 = generateTestTimelineWithLogIndices(
      'apps/v1#Deployment',
      [1, 2, 3],
    );
    const timeline2 = generateTestTimelineWithLogIndices(
      'apps/v1#Deployment#default#nginx',
      [1, 2, 3],
    );

    const timelines = [timeline1, timeline2];
    filter.chain(of(timelines)).subscribe((filtered) => {
      expect(filtered).toEqual([timeline1]);
    });
  });
});
