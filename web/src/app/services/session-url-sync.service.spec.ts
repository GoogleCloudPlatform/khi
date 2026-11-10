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

import { WritableSignal, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import {
  SESSION_LOCATION_ADAPTER,
  SessionLocationAdapter,
  SessionUrlSyncService,
} from 'src/app/services/session-url-sync.service';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { InspectionData } from 'src/app/store/domain/inspection-data';
import { Log } from 'src/app/store/domain/log';
import { Revision, Timeline } from 'src/app/store/domain/timeline';
import { TimelineView } from 'src/app/store/domain/timeline-view';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

describe('SessionUrlSyncService', () => {
  const fakeRevision0 = {
    id: 100,
    timelineId: 3,
    index: 0,
  } as unknown as ReadonlyDomainElement<Revision>;
  const fakeRevision1 = {
    id: 101,
    timelineId: 3,
    index: 1,
  } as unknown as ReadonlyDomainElement<Revision>;
  const fakeTimeline = {
    id: 3,
    revisions: [fakeRevision0, fakeRevision1],
  } as unknown as ReadonlyDomainElement<Timeline>;
  const fakeLog = { id: 42 } as unknown as ReadonlyDomainElement<Log>;

  let inspectionIdSignal: WritableSignal<string | null>;
  let inspectionDataSignal: WritableSignal<InspectionData | null>;
  let timelineViewSignal: WritableSignal<TimelineView | null>;
  let timelineQuerySignal: WritableSignal<string>;
  let logQuerySignal: WritableSignal<string>;
  let selectedTimelineSignal: WritableSignal<ReadonlyDomainElement<Timeline> | null>;
  let selectedRevisionSignal: WritableSignal<ReadonlyDomainElement<Revision> | null>;
  let selectedLogSignal: WritableSignal<ReadonlyDomainElement<Log> | null>;
  let fakeData: InspectionData;
  let fakeTimelineView: TimelineView;

  let mockLoader: jasmine.SpyObj<InspectionDataLoaderService>;
  let mockSelectionManager: jasmine.SpyObj<SelectionManager>;
  let currentSearch: string;
  let locationAdapter: SessionLocationAdapter;

  beforeEach(() => {
    inspectionIdSignal = signal<string | null>(null);
    inspectionDataSignal = signal<InspectionData | null>(null);
    timelineViewSignal = signal<TimelineView | null>(null);
    timelineQuerySignal = signal<string>('');
    logQuerySignal = signal<string>('');
    selectedTimelineSignal = signal<ReadonlyDomainElement<Timeline> | null>(
      null,
    );
    selectedRevisionSignal = signal<ReadonlyDomainElement<Revision> | null>(
      null,
    );
    selectedLogSignal = signal<ReadonlyDomainElement<Log> | null>(null);
    currentSearch = '';

    locationAdapter = {
      getSearch: () => currentSearch,
      replaceSearch: (search: string) => {
        currentSearch = search;
      },
    };

    fakeData = {
      timelineStore: {
        getTimeline: (id: number) => {
          if (id === 3) {
            return fakeTimeline;
          }
          throw new Error('not found');
        },
      },
      logStore: {
        getLog: (id: number) => {
          if (id === 42) {
            return fakeLog;
          }
          throw new Error('not found');
        },
      },
    } as unknown as InspectionData;

    fakeTimelineView = {
      expandAllTimelines: jasmine.createSpy('expandAllTimelines'),
      timelineQuery: timelineQuerySignal,
      logQuery: logQuerySignal,
      backendFilter: {
        updateFilterParams: jasmine
          .createSpy('updateFilterParams')
          .and.callFake(
            (params: { timelineQuery?: string; logQuery?: string }) => {
              if (params.timelineQuery !== undefined) {
                timelineQuerySignal.set(params.timelineQuery);
              }
              if (params.logQuery !== undefined) {
                logQuerySignal.set(params.logQuery);
              }
            },
          ),
      },
    } as unknown as TimelineView;

    mockLoader = jasmine.createSpyObj('InspectionDataLoaderService', [
      'loadInspectionDataFromBackend',
    ]);
    mockLoader.loadInspectionDataFromBackend.and.callFake(
      async (id: string) => {
        inspectionIdSignal.set(id);
        inspectionDataSignal.set(fakeData);
        timelineViewSignal.set(fakeTimelineView);
      },
    );

    mockSelectionManager = jasmine.createSpyObj(
      'SelectionManager',
      ['onSelectRevision', 'onSelectTimeline', 'onSelectLog'],
      {
        selectedTimeline: selectedTimelineSignal,
        selectedRevision: selectedRevisionSignal,
        selectedLog: selectedLogSignal,
      },
    );
    mockSelectionManager.onSelectRevision.and.callFake(
      (rev: ReadonlyDomainElement<Revision> | null) => {
        selectedRevisionSignal.set(rev);
        if (rev) {
          selectedTimelineSignal.set(fakeTimeline);
        }
      },
    );
    mockSelectionManager.onSelectTimeline.and.callFake(
      (timeline?: ReadonlyDomainElement<Timeline> | null) => {
        selectedTimelineSignal.set(timeline ?? null);
        if (!timeline) {
          selectedRevisionSignal.set(null);
          selectedLogSignal.set(null);
        }
      },
    );
    mockSelectionManager.onSelectLog.and.callFake(
      async (log: ReadonlyDomainElement<Log> | null) => {
        selectedLogSignal.set(log);
      },
    );

    TestBed.configureTestingModule({
      providers: [
        SessionUrlSyncService,
        { provide: SESSION_LOCATION_ADAPTER, useValue: locationAdapter },
        { provide: InspectionDataLoaderService, useValue: mockLoader },
        {
          provide: InspectionDataStore,
          useValue: {
            inspectionId: inspectionIdSignal,
            inspectionData: inspectionDataSignal,
            timelineView: timelineViewSignal,
          },
        },
        { provide: SelectionManager, useValue: mockSelectionManager },
      ],
    });
  });

  it('returns false when inspectionId query parameter is absent', () => {
    currentSearch = '?foo=bar';
    const service = TestBed.inject(SessionUrlSyncService);
    const opened = service.openFromQueryParams();
    expect(opened).toBeFalse();
    expect(mockLoader.loadInspectionDataFromBackend).not.toHaveBeenCalled();
  });

  it('loads inspection and selects target revision when timelineId and revisionIndex are provided', async () => {
    currentSearch = '?inspectionId=insp-1&timelineId=3&revisionIndex=1';
    const service = TestBed.inject(SessionUrlSyncService);
    const opened = service.openFromQueryParams();
    expect(opened).toBeTrue();

    await new Promise((resolve) => setTimeout(resolve, 0));
    TestBed.flushEffects();

    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledWith(
      'insp-1',
    );
    expect(mockSelectionManager.onSelectRevision).toHaveBeenCalledWith(
      fakeRevision1,
    );
    expect(currentSearch).toBe(
      '?inspectionId=insp-1&timelineId=3&revisionIndex=1',
    );
  });

  it('loads inspection and selects target log when logId is provided', async () => {
    currentSearch = '?inspectionId=insp-1&timelineId=3&logId=42';
    const service = TestBed.inject(SessionUrlSyncService);
    const opened = service.openFromQueryParams();
    expect(opened).toBeTrue();

    await new Promise((resolve) => setTimeout(resolve, 0));
    TestBed.flushEffects();

    expect(mockSelectionManager.onSelectTimeline).toHaveBeenCalledWith(
      fakeTimeline,
    );
    expect(mockSelectionManager.onSelectLog).toHaveBeenCalledWith(fakeLog);
    expect(currentSearch).toBe('?inspectionId=insp-1&timelineId=3&logId=42');
  });

  it('ignores non-decimal numeric strings in timelineId and logId query parameters', async () => {
    currentSearch = '?inspectionId=insp-1&timelineId=0x3&logId=1e1';
    const service = TestBed.inject(SessionUrlSyncService);
    const opened = service.openFromQueryParams();
    expect(opened).toBeTrue();

    await new Promise((resolve) => setTimeout(resolve, 0));
    TestBed.flushEffects();

    expect(mockSelectionManager.onSelectTimeline).not.toHaveBeenCalled();
    expect(mockSelectionManager.onSelectLog).not.toHaveBeenCalled();
    expect(currentSearch).toBe('?inspectionId=insp-1');
  });

  it('synchronizes inspection, selection, and filter changes back to URL query parameters', () => {
    const service = TestBed.inject(SessionUrlSyncService);
    expect(service.openFromQueryParams()).toBeFalse();

    inspectionIdSignal.set('insp-sync');
    inspectionDataSignal.set(fakeData);
    timelineViewSignal.set(fakeTimelineView);
    TestBed.flushEffects();
    expect(currentSearch).toBe('?inspectionId=insp-sync');

    selectedTimelineSignal.set(fakeTimeline);
    TestBed.flushEffects();
    expect(currentSearch).toBe('?inspectionId=insp-sync&timelineId=3');

    selectedRevisionSignal.set(fakeRevision1);
    TestBed.flushEffects();
    expect(currentSearch).toBe(
      '?inspectionId=insp-sync&timelineId=3&revisionIndex=1',
    );

    selectedRevisionSignal.set(null);
    selectedLogSignal.set(fakeLog);
    TestBed.flushEffects();
    expect(currentSearch).toBe('?inspectionId=insp-sync&timelineId=3&logId=42');

    timelineQuerySignal.set('match("Pod")');
    logQuerySignal.set('severity >= ERROR');
    TestBed.flushEffects();
    const params = new URLSearchParams(currentSearch);
    expect(params.get('inspectionId')).toBe('insp-sync');
    expect(params.get('timelineQuery')).toBe('match("Pod")');
    expect(params.get('logQuery')).toBe('severity >= ERROR');
    expect(params.get('timelineId')).toBe('3');
    expect(params.get('logId')).toBe('42');
    expect(params.get('revisionIndex')).toBeNull();

    selectedLogSignal.set(null);
    selectedTimelineSignal.set(null);
    timelineQuerySignal.set('');
    logQuerySignal.set('');
    TestBed.flushEffects();
    expect(currentSearch).toBe('?inspectionId=insp-sync');
  });
});
