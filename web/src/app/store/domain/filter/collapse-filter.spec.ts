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

import { CollapseTimelineFilter } from 'src/app/store/domain/filter/collapse-filter';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { LogTimelineFilterContext } from 'src/app/store/domain/filter/types';

describe('CollapseTimelineFilter', () => {
  let filter: CollapseTimelineFilter;

  beforeEach(() => {
    filter = new CollapseTimelineFilter();
  });

  it('should initialize with empty collapsed set', () => {
    expect(filter.collapsedTimelineIds.size).toBe(0);
  });

  it('should toggle timeline collapse state and emit onChanged', (done) => {
    filter.onChanged.subscribe(() => {
      expect(filter.collapsedTimelineIds.has(1)).toBeTrue();
      done();
    });
    filter.toggleTimelineCollapse(1);
  });

  it('should remove descendant timelines when process is called', async () => {
    const parentTimeline = {
      id: 1,
      descendants: function* () {
        yield { id: 2 } as unknown as Timeline;
        yield { id: 3 } as unknown as Timeline;
      },
    } as unknown as Timeline;

    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    timelineStoreSpy.getTimeline.and.callFake((id: number) => {
      if (id === 1) return parentTimeline;
      throw new Error('Not found');
    });

    filter.setCollapsedTimelineIds(new Set([1]));

    const initialContext: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2, 3, 4]),
      logIds: new Set([10, 20]),
    };

    const result = await filter.process(initialContext, timelineStoreSpy);

    expect(result.timelineIds.has(1)).toBeTrue();
    expect(result.timelineIds.has(2)).toBeFalse();
    expect(result.timelineIds.has(3)).toBeFalse();
    expect(result.timelineIds.has(4)).toBeTrue();
  });

  it('should expand direct children timelines', () => {
    const childTimeline = { id: 2 } as unknown as Timeline;
    const parentTimeline = {
      children: function* () {
        yield childTimeline;
      },
    } as unknown as Timeline;

    filter.setCollapsedTimelineIds(new Set([1, 2, 5]));
    filter.expandChildren(parentTimeline);

    expect(filter.collapsedTimelineIds.has(1)).toBeTrue();
    expect(filter.collapsedTimelineIds.has(2)).toBeFalse();
    expect(filter.collapsedTimelineIds.has(5)).toBeTrue();
  });

  it('should collapse direct children timelines with children', () => {
    const childWithChildren = {
      id: 2,
      childrenCount: 1,
    } as unknown as Timeline;
    const parentTimeline = {
      children: function* () {
        yield childWithChildren;
      },
    } as unknown as Timeline;

    filter.collapseChildren(parentTimeline);

    expect(filter.collapsedTimelineIds.has(1)).toBeFalse();
    expect(filter.collapsedTimelineIds.has(2)).toBeTrue();
  });

  it('should expand descendants timelines recursively', () => {
    const childTimeline = { id: 2 } as unknown as Timeline;
    const parentTimeline = {
      id: 1,
      descendants: function* () {
        yield childTimeline;
      },
    } as unknown as Timeline;

    filter.setCollapsedTimelineIds(new Set([1, 2, 5]));
    filter.expandDescendants(parentTimeline);

    expect(filter.collapsedTimelineIds.has(1)).toBeFalse();
    expect(filter.collapsedTimelineIds.has(2)).toBeFalse();
    expect(filter.collapsedTimelineIds.has(5)).toBeTrue();
  });

  it('should collapse descendants timelines recursively', () => {
    const childWithChildren = {
      id: 2,
      childrenCount: 1,
    } as unknown as Timeline;
    const parentTimeline = {
      id: 1,
      descendants: function* () {
        yield childWithChildren;
      },
    } as unknown as Timeline;

    filter.collapseDescendants(parentTimeline);

    expect(filter.collapsedTimelineIds.has(1)).toBeTrue();
    expect(filter.collapsedTimelineIds.has(2)).toBeTrue();
  });
});
