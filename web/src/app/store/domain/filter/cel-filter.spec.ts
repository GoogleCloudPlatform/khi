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

import { TestBed } from '@angular/core/testing';
import {
  CelTimelineFilter,
  CelTimelineExclusionFilter,
  CelLogFilter,
} from 'src/app/store/domain/filter/cel-filter';
import { LogTimelineFilterContext } from 'src/app/store/domain/filter/types';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { Timeline } from 'src/app/store/domain/timeline';
import { SearchWorkerManager } from 'src/app/services/search-worker-manager.service';

describe('CelTimelineFilter', () => {
  let filter: CelTimelineFilter;
  let searchWorkerManagerSpy: jasmine.SpyObj<SearchWorkerManager>;

  beforeEach(() => {
    searchWorkerManagerSpy = jasmine.createSpyObj<SearchWorkerManager>(
      'SearchWorkerManager',
      ['searchTimelines', 'searchLogs'],
    );

    TestBed.configureTestingModule({
      providers: [
        CelTimelineFilter,
        { provide: SearchWorkerManager, useValue: searchWorkerManagerSpy },
      ],
    });

    filter = TestBed.inject(CelTimelineFilter);
  });

  it('should filter timelines based on configured CEL expression', async () => {
    const timelines = [
      {
        id: 1,
        name: 'T1',
        type: { label: 'type1' },
        events: [],
        revisions: [],
      },
      {
        id: 2,
        name: 'T2',
        type: { label: 'type2' },
        events: [],
        revisions: [],
      },
    ];
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    timelineStoreSpy.getTimeline.and.callFake((id: number) => {
      const found = timelines.find((t) => t.id === id);
      if (!found) {
        throw new Error(`Timeline ${id} not found`);
      }
      return found as unknown as ReadonlyDomainElement<Timeline>;
    });

    searchWorkerManagerSpy.searchTimelines.and.returnValue(
      Promise.resolve(new Set([1])),
    );

    const res = filter.updateFilter("t.name == 'T1'");
    expect(res.success).toBe(true);

    const context: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2]),
      logIds: new Set(),
    };

    const result = await filter.process(context, timelineStoreSpy);
    expect(result.timelineIds.size).toBe(1);
    expect(result.timelineIds.has(1)).toBe(true);
    expect(searchWorkerManagerSpy.searchTimelines).toHaveBeenCalledWith(
      "t.name == 'T1'",
      undefined,
    );
  });

  it('should return original context if filter is not updated with an expression', async () => {
    const context: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2]),
      logIds: new Set(),
    };
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );

    const result = await filter.process(context, timelineStoreSpy);
    expect(result).toBe(context);
  });

  it('should return error and not filter context when updateFilter is called with an invalid expression', async () => {
    const res = filter.updateFilter("t.name == 'T1");
    expect(res.success).toBe(false);
    expect(res.error).toBeDefined();

    const context: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2]),
      logIds: new Set(),
    };
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );

    const result = await filter.process(context, timelineStoreSpy);
    expect(result).toBe(context);
  });

  it('should reset evaluator and return original context if an invalid expression is provided after a valid one', async () => {
    filter.updateFilter("t.name == 'T1'");

    const res = filter.updateFilter("name == 'T1");
    expect(res.success).toBe(false);

    const context: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2]),
      logIds: new Set(),
    };
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );

    const result = await filter.process(context, timelineStoreSpy);
    expect(result).toBe(context);
  });
});

describe('CelLogFilter', () => {
  let filter: CelLogFilter;
  let searchWorkerManagerSpy: jasmine.SpyObj<SearchWorkerManager>;

  beforeEach(() => {
    searchWorkerManagerSpy = jasmine.createSpyObj<SearchWorkerManager>(
      'SearchWorkerManager',
      ['searchTimelines', 'searchLogs'],
    );

    TestBed.configureTestingModule({
      providers: [
        CelLogFilter,
        { provide: SearchWorkerManager, useValue: searchWorkerManagerSpy },
      ],
    });

    filter = TestBed.inject(CelLogFilter);
  });

  it('should return original context if filter is not updated with an expression', async () => {
    const context: LogTimelineFilterContext = {
      timelineIds: new Set(),
      logIds: new Set([1, 2]),
    };
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );

    const result = await filter.process(context, timelineStoreSpy);
    expect(result).toBe(context);
  });

  it('should return error and not filter context when updateFilter is called with an invalid expression', async () => {
    const res = filter.updateFilter("l.summary == 'L1");
    expect(res.success).toBe(false);
    expect(res.error).toBeDefined();

    const context: LogTimelineFilterContext = {
      timelineIds: new Set(),
      logIds: new Set([1, 2]),
    };
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );

    const result = await filter.process(context, timelineStoreSpy);
    expect(result).toBe(context);
  });

  it('should reset evaluator and return original context if an invalid expression is provided after a valid one', async () => {
    filter.updateFilter("l.summary == 'L1'");

    const res = filter.updateFilter("summary == 'L1");
    expect(res.success).toBe(false);

    const context: LogTimelineFilterContext = {
      timelineIds: new Set(),
      logIds: new Set([1, 2]),
    };
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );

    const result = await filter.process(context, timelineStoreSpy);
    expect(result).toBe(context);
  });
});

describe('CelTimelineExclusionFilter', () => {
  let filter: CelTimelineExclusionFilter;
  let searchWorkerManagerSpy: jasmine.SpyObj<SearchWorkerManager>;

  beforeEach(() => {
    searchWorkerManagerSpy = jasmine.createSpyObj<SearchWorkerManager>(
      'SearchWorkerManager',
      ['searchTimelines', 'searchLogs'],
    );

    TestBed.configureTestingModule({
      providers: [
        CelTimelineExclusionFilter,
        { provide: SearchWorkerManager, useValue: searchWorkerManagerSpy },
      ],
    });

    filter = TestBed.inject(CelTimelineExclusionFilter);
  });

  it('should exclude timeline and its descendants when ancestor fails exclusion filter', async () => {
    const rootTimeline = {
      id: 10,
      name: 'Root',
      parent: null,
      events: [],
      revisions: [],
    };
    const childTimeline = {
      id: 11,
      name: 'Child',
      parent: rootTimeline,
      events: [],
      revisions: [],
    };

    const timelines = [rootTimeline, childTimeline];

    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    timelineStoreSpy.getTimeline.and.callFake((id: number) => {
      const found = timelines.find((t) => t.id === id);
      if (!found) {
        throw new Error(`Timeline ${id} not found`);
      }
      return found as unknown as ReadonlyDomainElement<Timeline>;
    });

    searchWorkerManagerSpy.searchTimelines.and.returnValue(
      Promise.resolve(new Set([10])),
    );

    filter.updateFilter('match("kube-system")');

    const context: LogTimelineFilterContext = {
      timelineIds: new Set([10, 11]),
      logIds: new Set(),
    };

    const result = await filter.process(context, timelineStoreSpy);
    expect(result.timelineIds.size).toBe(0);
  });
});
