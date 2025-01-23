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

import {
  LogType,
  ParentRelationship,
  RevisionState,
  RevisionVerb,
  Severity,
} from '../generated';
import { ResourceEvent } from './event';
import { ResourceRevision } from './revision';
import { TimelineEntry, TimelineLayer } from './timeline';

function generateTestTimeline(resourcePath: string): TimelineEntry {
  return new TimelineEntry(
    resourcePath,
    [],
    [],
    ParentRelationship.RelationshipChild,
  );
}

describe('TimelineEntry', () => {
  it('initialize parent field in child with addChildTimeline', () => {
    const child1 = new TimelineEntry(
      'core/v1#pod',
      [],
      [],
      ParentRelationship.RelationshipChild,
    );
    const p = new TimelineEntry(
      'core/v1',
      [],
      [],
      ParentRelationship.RelationshipChild,
    );
    p.addChildTimeline(child1);

    expect(p.parent).toBe(null);
    expect(child1.parent).toBe(p);
  });
  it('parses layer from resourcePath', () => {
    const testCases: {
      resourcePath: string;
      expectedLayer: TimelineLayer;
    }[] = [
      {
        resourcePath: 'core/v1',
        expectedLayer: TimelineLayer.APIVersion,
      },
      {
        resourcePath: 'core/v1#pod',
        expectedLayer: TimelineLayer.Kind,
      },
      {
        resourcePath: 'core/v1#pod#kube-system',
        expectedLayer: TimelineLayer.Namespace,
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns',
        expectedLayer: TimelineLayer.Name,
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns#binding',
        expectedLayer: TimelineLayer.Subresource,
      },
    ];
    for (const tc of testCases) {
      const tl = new TimelineEntry(
        tc.resourcePath,
        [],
        [],
        ParentRelationship.RelationshipChild,
      );

      expect(tl.layer).toBe(tc.expectedLayer);
    }
  });

  it('parses name from resourcePath', () => {
    const testCases: {
      resourcePath: string;
      expectedName: string;
    }[] = [
      {
        resourcePath: 'core/v1',
        expectedName: 'core/v1',
      },
      {
        resourcePath: 'core/v1#pod',
        expectedName: 'pod',
      },
      {
        resourcePath: 'core/v1#pod#kube-system',
        expectedName: 'kube-system',
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns',
        expectedName: 'kube-dns',
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns#binding',
        expectedName: 'binding',
      },
    ];
    for (const tc of testCases) {
      const tl = new TimelineEntry(
        tc.resourcePath,
        [],
        [],
        ParentRelationship.RelationshipChild,
      );

      expect(tl.name).toBe(tc.expectedName);
    }
  });

  it('returns name of layers from resourcePath with getNameOfLayer', () => {
    const testCases: {
      resourcePath: string;
      targetLayer: TimelineLayer;
      expectedName: string;
    }[] = [
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns#binding',
        targetLayer: TimelineLayer.APIVersion,
        expectedName: 'core/v1',
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns#binding',
        targetLayer: TimelineLayer.Kind,
        expectedName: 'pod',
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns#binding',
        targetLayer: TimelineLayer.Namespace,
        expectedName: 'kube-system',
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns#binding',
        targetLayer: TimelineLayer.Name,
        expectedName: 'kube-dns',
      },
      {
        resourcePath: 'core/v1#pod#kube-system#kube-dns#binding',
        targetLayer: TimelineLayer.Subresource,
        expectedName: 'binding',
      },
      {
        resourcePath: 'core/v1#pod#kube-system',
        targetLayer: TimelineLayer.Subresource,
        expectedName: '',
      },
    ];
    for (const tc of testCases) {
      const tl = new TimelineEntry(
        tc.resourcePath,
        [],
        [],
        ParentRelationship.RelationshipChild,
      );

      expect(tl.getNameOfLayer(tc.targetLayer)).toBe(tc.expectedName);
    }
  });

  describe('queryEventsInRange', () => {
    it('returns the list of events in the range', () => {
      const eventTimes = [0, 1, 2, 3, 4, 5];
      const tl = new TimelineEntry(
        'core/v1',
        [],
        eventTimes.map(
          (t) =>
            new ResourceEvent(
              t,
              t,
              LogType.LogTypeAudit,
              Severity.SeverityError,
            ),
        ),
        ParentRelationship.RelationshipChild,
      );

      const queried = tl.queryEventsInRange(2, 4);

      expect(queried.length).toBe(2);
      expect(queried[0].logIndex).toBe(2);
      expect(queried[1].logIndex).toBe(3);
    });
  });

  describe('pickEventNearCenterOfRange', () => {
    it('returns a event at the center of specified range', () => {
      const eventTimes = [0, 1, 2, 3, 4, 5];
      const tl = new TimelineEntry(
        'core/v1',
        [],
        eventTimes.map(
          (t) =>
            new ResourceEvent(
              t,
              t,
              LogType.LogTypeAudit,
              Severity.SeverityError,
            ),
        ),
        ParentRelationship.RelationshipChild,
      );

      const center = tl.pickEventNearCenterOfRange(1.5, 4); // center is 2.75

      expect(center).not.toBeNull();
      expect(center?.logIndex).toBe(3);
    });

    it('returns null when the range is outside of events', () => {
      const eventTimes = [0, 1, 2, 3, 4, 5];
      const tl = new TimelineEntry(
        'core/v1',
        [],
        eventTimes.map(
          (t) =>
            new ResourceEvent(
              t,
              t,
              LogType.LogTypeAudit,
              Severity.SeverityError,
            ),
        ),
        ParentRelationship.RelationshipChild,
      );

      const center = tl.pickEventNearCenterOfRange(10, 20);

      expect(center).toBeNull();
    });
  });

  describe('queryRevisionsInRange', () => {
    it('returns the list of revisions in the range', () => {
      const revisionTimes = [0, 1, 2, 3, 4, 5];
      const tl = new TimelineEntry(
        'core/v1',
        revisionTimes.map(
          (t) =>
            new ResourceRevision(
              t,
              t + 1,
              RevisionState.RevisionStateComposerTiDeferred,
              RevisionVerb.RevisionVerbComposerTaskInstanceDeferred,
              '',
              '',
              false,
              false,
              t,
            ),
        ),
        [],
        ParentRelationship.RelationshipChild,
      );

      const queried = tl.queryRevisionsInRange(2, 4);

      expect(queried.length).toBe(2);
      expect(queried[0].logIndex).toBe(2);
      expect(queried[1].logIndex).toBe(3);
    });

    it('returns the revision overwrapping the given range', () => {
      const revisionTimes = [0, 1, 2, 3, 4, 5];
      const tl = new TimelineEntry(
        'core/v1',
        revisionTimes.map(
          (t) =>
            new ResourceRevision(
              t,
              t + 1,
              RevisionState.RevisionStateComposerTiDeferred,
              RevisionVerb.RevisionVerbComposerTaskInstanceDeferred,
              '',
              '',
              false,
              false,
              t,
            ),
        ),
        [],
        ParentRelationship.RelationshipChild,
      );

      const queried = tl.queryRevisionsInRange(2.25, 2.75); // overwrapping the revision with [2,3]

      expect(queried.length).toBe(1);
      expect(queried[0].logIndex).toBe(2);
    });
  });

  describe('getLatestRevisionOfTime', () => {
    it('returns the revision not reaching the given endtime', () => {
      const revisionTimes = [0, 1, 2, 3, 4, 5];
      const tl = new TimelineEntry(
        'core/v1',
        revisionTimes.map(
          (t) =>
            new ResourceRevision(
              t,
              t + 1,
              RevisionState.RevisionStateComposerTiDeferred,
              RevisionVerb.RevisionVerbComposerTaskInstanceDeferred,
              '',
              '',
              false,
              false,
              t,
            ),
        ),
        [],
        ParentRelationship.RelationshipChild,
      );

      const revision = tl.getLatestRevisionOfTime(3.5);

      expect(revision).not.toBeNull();
      expect(revision?.startAt).toBe(3);
    });

    it('returns null when no revision started yet at the given time', () => {
      const revisionTimes = [2, 3, 4, 5];
      const tl = new TimelineEntry(
        'core/v1',
        revisionTimes.map(
          (t) =>
            new ResourceRevision(
              t,
              t + 1,
              RevisionState.RevisionStateComposerTiDeferred,
              RevisionVerb.RevisionVerbComposerTaskInstanceDeferred,
              '',
              '',
              false,
              false,
              t,
            ),
        ),
        [],
        ParentRelationship.RelationshipChild,
      );

      const revision = tl.getLatestRevisionOfTime(1);

      expect(revision).toBeNull();
    });
  });

  describe('getAllChildrenRecursive', () => {
    it('returns all the children', () => {
      const p1 = generateTestTimeline('core/v1');
      const p1c1 = generateTestTimeline('core/v1#pod');
      const p1c2 = generateTestTimeline('core/v1#replicaset');
      const p1c1c1 = generateTestTimeline('core/v1#pod#kube-sytem');
      const p1c1c2 = generateTestTimeline('core/v1#pod#istio-sytem');
      const p1c2c1 = generateTestTimeline('core/v1#replicaset#kube-system');
      p1.addChildTimeline(p1c1);
      p1.addChildTimeline(p1c2);
      p1c1.addChildTimeline(p1c1c1);
      p1c1.addChildTimeline(p1c1c2);
      p1c2.addChildTimeline(p1c2c1);

      const timelines = p1.getAllChildrenRecursive();
      const setFromResult = new Set(timelines);

      expect(timelines.length).toBe(5);
      expect(setFromResult.has(p1c1)).toBeTrue();
      expect(setFromResult.has(p1c2)).toBeTrue();
      expect(setFromResult.has(p1c1c1)).toBeTrue();
      expect(setFromResult.has(p1c1c2)).toBeTrue();
      expect(setFromResult.has(p1c2c1)).toBeTrue();
    });
  });

  describe('getRevisionPairByLogId', () => {
    it('returns the pair when the revision having the log index found in the timeline', () => {
      const logIndices = [0, 3, 5, 7, 9];
      const timeline = new TimelineEntry(
        'core/v1',
        logIndices.map(
          (li, ri) =>
            new ResourceRevision(
              ri,
              ri + 1,
              RevisionState.RevisionStateExisting,
              RevisionVerb.RevisionVerbUpdate,
              '',
              '',
              false,
              false,
              li,
            ),
        ),
        [],
        ParentRelationship.RelationshipChild,
      );

      const pair = timeline.getRevisionPairByLogId(5);
      expect(pair).not.toBeNull();
      expect(pair?.current?.logIndex).toBe(5);
      expect(pair?.previous?.logIndex).toBe(3);
    });

    it('returns the pair without the previous field when there is no older revision', () => {
      const logIndices = [0, 3, 5, 7, 9];
      const timeline = new TimelineEntry(
        'core/v1',
        logIndices.map(
          (li, ri) =>
            new ResourceRevision(
              ri,
              ri + 1,
              RevisionState.RevisionStateExisting,
              RevisionVerb.RevisionVerbUpdate,
              '',
              '',
              false,
              false,
              li,
            ),
        ),
        [],
        ParentRelationship.RelationshipChild,
      );

      const pair = timeline.getRevisionPairByLogId(0);
      expect(pair).not.toBeNull();
      expect(pair?.current?.logIndex).toBe(0);
      expect(pair?.previous).toBeNull();
    });

    it('returns null when no revisions are included in the timeline', () => {
      const timeline = new TimelineEntry(
        'core/v1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      );

      const pair = timeline.getRevisionPairByLogId(5);
      expect(pair).toBeNull();
    });
  });

  describe('hasNonFilteredOutIndices', () => {
    it('returns true when there is a revision not included in the filtered indices', () => {
      const timeline = new TimelineEntry(
        'core/v1',
        [
          new ResourceRevision(
            0,
            1,
            RevisionState.RevisionStateExisting,
            RevisionVerb.RevisionVerbUpdate,
            '',
            '',
            false,
            false,
            0,
          ),
          new ResourceRevision(
            1,
            2,
            RevisionState.RevisionStateExisting,
            RevisionVerb.RevisionVerbUpdate,
            '',
            '',
            false,
            false,
            1,
          ),
        ],
        [],
        ParentRelationship.RelationshipChild,
      );
      const filteredOut = new Set([0]);
      expect(timeline.hasNonFilteredOutIndices(filteredOut)).toBeTrue();
    });
    it('returns false when all revisions are included in the filtered indices', () => {
      const timeline = new TimelineEntry(
        'core/v1',
        [
          new ResourceRevision(
            0,
            1,
            RevisionState.RevisionStateExisting,
            RevisionVerb.RevisionVerbUpdate,
            '',
            '',
            false,
            false,
            0,
          ),
          new ResourceRevision(
            1,
            2,
            RevisionState.RevisionStateExisting,
            RevisionVerb.RevisionVerbUpdate,
            '',
            '',
            false,
            false,
            1,
          ),
        ],
        [],
        ParentRelationship.RelationshipChild,
      );
      const filteredOut = new Set([0, 1]);
      expect(timeline.hasNonFilteredOutIndices(filteredOut)).toBeFalse();
    });
    it('returns true when there is an event not included in the filtered indices', () => {
      const timeline = new TimelineEntry(
        'core/v1',
        [],
        [
          new ResourceEvent(0, 0, LogType.LogTypeAudit, Severity.SeverityError),
          new ResourceEvent(1, 1, LogType.LogTypeAudit, Severity.SeverityError),
        ],
        ParentRelationship.RelationshipChild,
      );
      const filteredOut = new Set([0]);
      expect(timeline.hasNonFilteredOutIndices(filteredOut)).toBeTrue();
    });
    it('returns false when all events are included in the filtered indices', () => {
      const timeline = new TimelineEntry(
        'core/v1',
        [],
        [
          new ResourceEvent(0, 0, LogType.LogTypeAudit, Severity.SeverityError),
          new ResourceEvent(1, 1, LogType.LogTypeAudit, Severity.SeverityError),
        ],
        ParentRelationship.RelationshipChild,
      );
      const filteredOut = new Set([0, 1]);
      expect(timeline.hasNonFilteredOutIndices(filteredOut)).toBeFalse();
    });
    it('returns false when there is no revision and event', () => {
      const timeline = new TimelineEntry(
        'core/v1',
        [],
        [],
        ParentRelationship.RelationshipChild,
      );
      const filteredOut = new Set([0, 1]);
      expect(timeline.hasNonFilteredOutIndices(filteredOut)).toBeFalse();
    });
  });

  describe('hasNonFilteredOutIndicesRecursive', () => {
    it('returns true when there is a revision not included in the filtered indices in its children', () => {
      const p1 = generateTestTimeline('core/v1');
      const p1c1 = new TimelineEntry(
        'core/v1#pod',
        [
          new ResourceRevision(
            0,
            1,
            RevisionState.RevisionStateExisting,
            RevisionVerb.RevisionVerbUpdate,
            '',
            '',
            false,
            false,
            0,
          ),
        ],
        [],
        ParentRelationship.RelationshipChild,
      );
      p1.addChildTimeline(p1c1);
      const filteredOut = new Set([1]);
      expect(p1.hasNonFilteredOutIndicesRecursive(filteredOut)).toBeTrue();
    });
    it('returns false when all revisions are included in the filtered indices in its children', () => {
      const p1 = generateTestTimeline('core/v1');
      const p1c1 = new TimelineEntry(
        'core/v1#pod',
        [
          new ResourceRevision(
            0,
            1,
            RevisionState.RevisionStateExisting,
            RevisionVerb.RevisionVerbUpdate,
            '',
            '',
            false,
            false,
            0,
          ),
        ],
        [],
        ParentRelationship.RelationshipChild,
      );
      p1.addChildTimeline(p1c1);
      const filteredOut = new Set([0]);
      expect(p1.hasNonFilteredOutIndicesRecursive(filteredOut)).toBeFalse();
    });
  });
});
