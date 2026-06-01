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

import { TimelineView } from 'src/app/store/domain/timeline-view';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { LogTimelineFilter } from 'src/app/store/domain/filter/types';

describe('TimelineView', () => {
  it('should process registered filters in order of their priority', () => {
    const logStoreSpy = jasmine.createSpyObj<LogStore>('LogStore', [
      'getLog',
      'logs',
    ]);
    logStoreSpy.logs.and.returnValue([][Symbol.iterator]());
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    Object.defineProperty(timelineStoreSpy, 'logStore', {
      get: () => logStoreSpy,
    });
    Object.defineProperty(timelineStoreSpy, 'timelines', {
      get: () => [],
    });

    const filter1 = jasmine.createSpyObj<LogTimelineFilter>('Filter1', [
      'priority',
      'process',
    ]);
    filter1.priority.and.returnValue(20);
    filter1.process.and.callFake((ctx) => ctx);

    const filter2 = jasmine.createSpyObj<LogTimelineFilter>('Filter2', [
      'priority',
      'process',
    ]);
    filter2.priority.and.returnValue(10);
    filter2.process.and.callFake((ctx) => ctx);

    const view = new TimelineView(timelineStoreSpy);
    view.addFilter(filter1);
    view.addFilter(filter2);

    // Accessing the signal triggers lazy evaluation
    view.filteredTimelines();

    expect(filter2.process).toHaveBeenCalledBefore(filter1.process);
  });

  it('should allow removing and clearing filters', () => {
    const logStoreSpy = jasmine.createSpyObj<LogStore>('LogStore', [
      'getLog',
      'logs',
    ]);
    logStoreSpy.logs.and.returnValue([][Symbol.iterator]());
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    Object.defineProperty(timelineStoreSpy, 'logStore', {
      get: () => logStoreSpy,
    });
    Object.defineProperty(timelineStoreSpy, 'timelines', {
      get: () => [],
    });

    const filter = jasmine.createSpyObj<LogTimelineFilter>('Filter', [
      'priority',
      'process',
    ]);
    filter.priority.and.returnValue(10);
    filter.process.and.callFake((ctx) => ctx);

    const view = new TimelineView(timelineStoreSpy);
    view.addFilter(filter);
    view.removeFilter(filter);
    view.filteredTimelines();
    expect(filter.process).not.toHaveBeenCalled();

    view.addFilter(filter);
    view.clearFilters();
    view.filteredTimelines();
    expect(filter.process).not.toHaveBeenCalled();
  });

  it('should expose filtered results as signals', () => {
    const logStoreSpy = jasmine.createSpyObj<LogStore>('LogStore', [
      'getLog',
      'logs',
    ]);
    logStoreSpy.logs.and.returnValue([][Symbol.iterator]());
    const timelineStoreSpy = jasmine.createSpyObj<TimelineStore>(
      'TimelineStore',
      ['getTimeline'],
    );
    Object.defineProperty(timelineStoreSpy, 'logStore', {
      get: () => logStoreSpy,
    });
    Object.defineProperty(timelineStoreSpy, 'timelines', {
      get: () => [],
    });

    const view = new TimelineView(timelineStoreSpy);

    const timelinesSignal = view.filteredTimelines;
    const logsSignal = view.filteredLogs;
    const logIdsSignal = view.filteredLogIds;

    expect(typeof timelinesSignal).toBe('function');
    expect(typeof logsSignal).toBe('function');
    expect(typeof logIdsSignal).toBe('function');

    expect(timelinesSignal()).toEqual([]);
    expect(logsSignal()).toEqual([]);
    expect(logIdsSignal().size).toBe(0);
  });
});
