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

import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { InspectionQueryUrlOpener } from 'src/app/extensions/extension-common/inspection-query-url-opener';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { InspectionData } from 'src/app/store/domain/inspection-data';
import { Log } from 'src/app/store/domain/log';
import { Revision, Timeline } from 'src/app/store/domain/timeline';
import { TimelineView } from 'src/app/store/domain/timeline-view';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

describe('InspectionQueryUrlOpener', () => {
  const fakeRevision0 = {
    id: 100,
  } as unknown as ReadonlyDomainElement<Revision>;
  const fakeRevision1 = {
    id: 101,
  } as unknown as ReadonlyDomainElement<Revision>;
  const fakeTimeline = {
    id: 3,
    revisions: [fakeRevision0, fakeRevision1],
  } as unknown as ReadonlyDomainElement<Timeline>;
  const fakeLog = { id: 42 } as unknown as ReadonlyDomainElement<Log>;

  let mockLoader: jasmine.SpyObj<InspectionDataLoaderService>;
  let mockSelectionManager: jasmine.SpyObj<SelectionManager>;

  beforeEach(() => {
    const inspectionDataSignal = signal<InspectionData | null>(null);
    const timelineViewSignal = signal<TimelineView | null>(null);

    const fakeData = {
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

    const fakeTimelineView = {
      expandAllTimelines: jasmine.createSpy('expandAllTimelines'),
      backendFilter: {
        updateFilterParams: jasmine.createSpy('updateFilterParams'),
      },
    } as unknown as TimelineView;

    mockLoader = jasmine.createSpyObj('InspectionDataLoaderService', [
      'loadInspectionDataFromBackend',
    ]);
    mockLoader.loadInspectionDataFromBackend.and.callFake(async () => {
      inspectionDataSignal.set(fakeData);
      timelineViewSignal.set(fakeTimelineView);
    });

    mockSelectionManager = jasmine.createSpyObj('SelectionManager', [
      'onSelectRevision',
      'onSelectTimeline',
      'onSelectLog',
    ]);
    mockSelectionManager.onSelectLog.and.resolveTo();

    TestBed.configureTestingModule({
      providers: [
        { provide: InspectionDataLoaderService, useValue: mockLoader },
        {
          provide: InspectionDataStore,
          useValue: {
            inspectionData: inspectionDataSignal,
            timelineView: timelineViewSignal,
          },
        },
        { provide: SelectionManager, useValue: mockSelectionManager },
      ],
    });
  });

  it('returns false when inspectionId query parameter is absent', () => {
    const opener = new InspectionQueryUrlOpener(() => '?foo=bar');
    const opened = TestBed.runInInjectionContext(() => opener.tryOpen());
    expect(opened).toBeFalse();
    expect(mockLoader.loadInspectionDataFromBackend).not.toHaveBeenCalled();
  });

  it('loads inspection and selects target revision when timelineId and revisionIndex are provided', async () => {
    const opener = new InspectionQueryUrlOpener(
      () => '?inspectionId=insp-1&timelineId=3&revisionIndex=1',
    );
    const opened = TestBed.runInInjectionContext(() => opener.tryOpen());
    expect(opened).toBeTrue();

    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledWith(
      'insp-1',
    );
    expect(mockSelectionManager.onSelectRevision).toHaveBeenCalledWith(
      fakeRevision1,
    );
  });

  it('loads inspection and selects target log when logId is provided', async () => {
    const opener = new InspectionQueryUrlOpener(
      () => '?inspectionId=insp-1&timelineId=3&logId=42',
    );
    const opened = TestBed.runInInjectionContext(() => opener.tryOpen());
    expect(opened).toBeTrue();

    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(mockSelectionManager.onSelectTimeline).toHaveBeenCalledWith(
      fakeTimeline,
    );
    expect(mockSelectionManager.onSelectLog).toHaveBeenCalledWith(fakeLog);
  });
});
