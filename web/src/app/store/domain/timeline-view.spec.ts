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
import {
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';

describe('TimelineView', () => {
  async function waitForFiltering(view: TimelineView): Promise<void> {
    // Allow the async pipeline microtask to schedule and run
    await new Promise((resolve) => setTimeout(resolve, 0));
    while (view.isFiltering()) {
      await new Promise((resolve) => setTimeout(resolve, 5));
    }
  }

  it('should process registered filters in order of their priority', async () => {
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
    Object.defineProperty(filter1, 'priority', { get: () => 20 });
    filter1.process.and.callFake((ctx) => Promise.resolve(ctx));

    const filter2 = jasmine.createSpyObj<LogTimelineFilter>('Filter2', [
      'priority',
      'process',
    ]);
    Object.defineProperty(filter2, 'priority', { get: () => 10 });
    filter2.process.and.callFake((ctx) => Promise.resolve(ctx));

    const view = new TimelineView(timelineStoreSpy);

    view.addFilter(filter1);
    view.addFilter(filter2);

    await waitForFiltering(view);

    expect(filter2.process).toHaveBeenCalledBefore(filter1.process);
  });

  it('should allow removing and clearing filters', async () => {
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
    Object.defineProperty(filter, 'priority', { get: () => 10 });
    filter.process.and.callFake((ctx) => Promise.resolve(ctx));

    const view = new TimelineView(timelineStoreSpy);

    view.addFilter(filter);
    await waitForFiltering(view);
    expect(filter.process).toHaveBeenCalledTimes(1);

    filter.process.calls.reset();
    view.removeFilter(filter);
    await waitForFiltering(view);
    expect(filter.process).not.toHaveBeenCalled();

    filter.process.calls.reset();
    view.addFilter(filter);
    await waitForFiltering(view);
    expect(filter.process).toHaveBeenCalledTimes(1);

    filter.process.calls.reset();
    view.clearFilters();
    await waitForFiltering(view);
    expect(filter.process).not.toHaveBeenCalled();
  });

  it('should expose filtered results as signals', async () => {
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

    await waitForFiltering(view);

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

  it('should cancel previous execution if filters change mid-run', async () => {
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

    const processSignals: AbortSignal[] = [];

    const filter = jasmine.createSpyObj<LogTimelineFilter>('Filter', [
      'priority',
      'process',
    ]);
    filter.process.and.callFake((ctx, store, signal) => {
      if (signal) {
        processSignals.push(signal);
      }
      // Simulate slow process
      return new Promise((resolve) => setTimeout(() => resolve(ctx), 50));
    });

    const view = new TimelineView(timelineStoreSpy);

    view.addFilter(filter);

    // Change filters mid-run by clear and re-add
    await new Promise((resolve) => setTimeout(resolve, 10));
    view.clearFilters();
    view.addFilter(filter);

    await waitForFiltering(view);

    expect(processSignals.length).toBe(2);
    expect(processSignals[0].aborted).toBe(true);
    expect(processSignals[1].aborted).toBe(false);
  });

  it('should report filtering progress via progress signal and clear it when finished', async () => {
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

    let resolveProcess: (ctx: LogTimelineFilterContext) => void = () => {};
    const processPromise = new Promise<LogTimelineFilterContext>((resolve) => {
      resolveProcess = resolve;
    });

    const filter = jasmine.createSpyObj<LogTimelineFilter>('ProgressFilter', [
      'priority',
      'process',
    ]);
    Object.defineProperty(filter, 'priority', { get: () => 10 });
    Object.defineProperty(filter, 'displayName', {
      get: () => 'ProgressFilter',
    });
    filter.process.and.callFake((ctx, store, signal, onProgress) => {
      onProgress?.(5, 10);
      return processPromise;
    });

    const view = new TimelineView(timelineStoreSpy);
    expect(view.progress()).toBeNull();

    view.addFilter(filter);

    // Wait for microtask queue to start runPipeline
    await new Promise((resolve) => setTimeout(resolve, 0));

    // The filter.process should be running now, and onProgress has been called
    expect(view.progress()).toEqual({
      filterName: 'ProgressFilter',
      current: 5,
      total: 10,
    });

    // Resolve the filter process
    resolveProcess({ timelineIds: new Set(), logIds: new Set() });

    // Wait for complete pipeline termination
    await waitForFiltering(view);

    // It should be cleared now
    expect(view.progress()).toBeNull();
  });
});
