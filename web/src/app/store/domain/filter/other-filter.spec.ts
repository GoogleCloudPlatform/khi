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
  IncludeDescendantsFilter,
  IncludeAncestorsFilter,
  ExcludeNoLogsFilter,
} from 'src/app/store/domain/filter/other-filter';
import { LogTimelineFilterContext } from 'src/app/store/domain/filter/types';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { Timeline } from 'src/app/store/domain/timeline';

describe('IncludeDescendantsFilter', () => {
  it('should include all descendants and extract associated logs', async () => {
    const childTimeline = {
      id: 2,
      children: () => [],
      revisions: [{ log: { id: 102 } }],
      events: [],
    };
    const rootTimeline = {
      id: 1,
      children: () => [childTimeline],
      revisions: [{ log: { id: 101 } }],
      events: [],
    };

    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    timelineStoreSpy.getTimeline.and.callFake((id: number) => {
      if (id === 1) {
        return rootTimeline as unknown as ReadonlyDomainElement<Timeline>;
      }
      if (id === 2) {
        return childTimeline as unknown as ReadonlyDomainElement<Timeline>;
      }
      throw new Error('not found');
    });

    const filter = new IncludeDescendantsFilter();
    const context: LogTimelineFilterContext = {
      timelineIds: new Set([1]),
      logIds: new Set(),
    };

    const res = await filter.process(context, timelineStoreSpy);
    expect(res.timelineIds.size).toBe(2);
    expect(res.timelineIds.has(1)).toBe(true);
    expect(res.timelineIds.has(2)).toBe(true);
    expect(res.logIds.size).toBe(2);
    expect(res.logIds.has(101)).toBe(true);
    expect(res.logIds.has(102)).toBe(true);
  });
});

describe('IncludeAncestorsFilter', () => {
  it('should include all ancestors and extract associated logs', async () => {
    const rootTimeline = {
      id: 1,
      parent: null,
      revisions: [{ log: { id: 101 } }],
      events: [],
    };
    const childTimeline = {
      id: 2,
      parent: rootTimeline,
      revisions: [{ log: { id: 102 } }],
      events: [],
    };

    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    timelineStoreSpy.getTimeline.and.callFake((id: number) => {
      if (id === 1) {
        return rootTimeline as unknown as ReadonlyDomainElement<Timeline>;
      }
      if (id === 2) {
        return childTimeline as unknown as ReadonlyDomainElement<Timeline>;
      }
      throw new Error('not found');
    });

    const filter = new IncludeAncestorsFilter();
    const context: LogTimelineFilterContext = {
      timelineIds: new Set([2]),
      logIds: new Set(),
    };

    const res = await filter.process(context, timelineStoreSpy);
    expect(res.timelineIds.size).toBe(2);
    expect(res.timelineIds.has(1)).toBe(true);
    expect(res.timelineIds.has(2)).toBe(true);
    expect(res.logIds.size).toBe(0);
  });
});

describe('ExcludeNoLogsFilter', () => {
  it('should exclude timelines without matching logs', async () => {
    const t1 = {
      id: 1,
      revisions: [{ log: { id: 101 } }],
      events: [],
    };
    const t2 = {
      id: 2,
      revisions: [{ log: { id: 102 } }],
      events: [],
    };

    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    timelineStoreSpy.getTimeline.and.callFake((id: number) => {
      if (id === 1) return t1 as unknown as ReadonlyDomainElement<Timeline>;
      if (id === 2) return t2 as unknown as ReadonlyDomainElement<Timeline>;
      throw new Error('not found');
    });

    const filter = new ExcludeNoLogsFilter();
    filter.setEnabled(true);
    const context: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2]),
      logIds: new Set([101]),
    };

    const res = await filter.process(context, timelineStoreSpy);
    expect(res.timelineIds.size).toBe(1);
    expect(res.timelineIds.has(1)).toBe(true);
    expect(res.timelineIds.has(2)).toBe(false);
  });

  it('should bypass filtering when disabled', async () => {
    const t1 = {
      id: 1,
      revisions: [{ log: { id: 101 } }],
      events: [],
    };
    const t2 = {
      id: 2,
      revisions: [{ log: { id: 102 } }],
      events: [],
    };

    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    timelineStoreSpy.getTimeline.and.callFake((id: number) => {
      if (id === 1) return t1 as unknown as ReadonlyDomainElement<Timeline>;
      if (id === 2) return t2 as unknown as ReadonlyDomainElement<Timeline>;
      throw new Error('not found');
    });

    const filter = new ExcludeNoLogsFilter();
    filter.setEnabled(false);
    const context: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2]),
      logIds: new Set([101]),
    };

    const res = await filter.process(context, timelineStoreSpy);
    expect(res.timelineIds.size).toBe(2);
    expect(res.timelineIds.has(1)).toBe(true);
    expect(res.timelineIds.has(2)).toBe(true);
  });
});
