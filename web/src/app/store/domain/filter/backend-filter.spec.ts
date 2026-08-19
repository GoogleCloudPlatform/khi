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

import { BackendFilter } from 'src/app/store/domain/filter/backend-filter';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { LogTimelineFilterContext } from 'src/app/store/domain/filter/types';

describe('BackendFilter', () => {
  let mockWorkbenchClient: jasmine.SpyObj<WorkbenchClientService>;
  let timelineStoreDummy: TimelineStore;

  beforeEach(() => {
    mockWorkbenchClient = jasmine.createSpyObj<WorkbenchClientService>(
      'WorkbenchClientService',
      ['isWorkbenchActive', 'filterTimeline'],
    );
    timelineStoreDummy = {} as TimelineStore;
  });

  it('should initialize with default parameter values', () => {
    const filter = new BackendFilter(mockWorkbenchClient);
    expect(filter.displayName).toBe('Backend filter');
    expect(filter.priority).toBe(10);
    expect(filter.timelineQuery()).toBe('');
    expect(filter.timelineExclusionQuery()).toBe('');
    expect(filter.logQuery()).toBe('');
    expect(filter.excludeNoLogs()).toBeFalse();
  });

  it('should update filter parameters and emit onChanged', () => {
    const filter = new BackendFilter(mockWorkbenchClient);
    let changed = false;
    filter.onChanged.subscribe(() => {
      changed = true;
    });

    filter.updateFilterParams({
      timelineQuery: 'match("Pod", ".*")',
      excludeNoLogs: true,
    });

    expect(filter.timelineQuery()).toBe('match("Pod", ".*")');
    expect(filter.excludeNoLogs()).toBeTrue();
    expect(changed).toBeTrue();
  });

  it('should return original context if workbench is not active', async () => {
    mockWorkbenchClient.isWorkbenchActive.and.returnValue(false);
    const filter = new BackendFilter(mockWorkbenchClient);

    const initialContext: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2, 3]),
      logIds: new Set([10, 20]),
    };

    const res = await filter.process(initialContext, timelineStoreDummy);
    expect(res).toBe(initialContext);
    expect(mockWorkbenchClient.filterTimeline).not.toHaveBeenCalled();
  });

  it('should call workbenchClient.filterTimeline and cache results', async () => {
    mockWorkbenchClient.isWorkbenchActive.and.returnValue(true);
    mockWorkbenchClient.filterTimeline.and.resolveTo({
      timelineIds: [1, 2],
      logIds: [10],
    });

    const filter = new BackendFilter(mockWorkbenchClient);
    filter.updateFilterParams({
      timelineQuery: 'match("Pod", ".*")',
    });

    const initialContext: LogTimelineFilterContext = {
      timelineIds: new Set([1, 2, 3]),
      logIds: new Set([10, 20]),
    };

    const res1 = await filter.process(initialContext, timelineStoreDummy);
    expect(res1.timelineIds).toEqual(new Set([1, 2]));
    expect(res1.logIds).toEqual(new Set([10]));
    expect(mockWorkbenchClient.filterTimeline).toHaveBeenCalledTimes(1);

    // Second call with same parameters should return cached result without calling RPC
    const res2 = await filter.process(initialContext, timelineStoreDummy);
    expect(res2).toBe(res1);
    expect(mockWorkbenchClient.filterTimeline).toHaveBeenCalledTimes(1);

    // Invalidate cache and verify RPC is called again
    filter.invalidateCache();
    await filter.process(initialContext, timelineStoreDummy);
    expect(mockWorkbenchClient.filterTimeline).toHaveBeenCalledTimes(2);
  });
});
