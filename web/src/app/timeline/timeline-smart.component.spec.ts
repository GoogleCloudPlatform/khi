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

import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import {
  TimeRangeFilter,
  ViewStateService,
} from 'src/app/services/view-state.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { StyleOverrideService } from 'src/app/services/style-override.service';
import { TimelineSmartComponent } from './timeline-smart.component';

describe('TimelineSmartComponent', () => {
  let component: TimelineSmartComponent;
  let fixture: ComponentFixture<TimelineSmartComponent>;

  let mockViewStateService: jasmine.SpyObj<ViewStateService>;
  let mockInspectionDataStore: jasmine.SpyObj<InspectionDataStore>;
  let mockSelectionManager: jasmine.SpyObj<SelectionManager>;
  let mockStyleOverrideService: jasmine.SpyObj<StyleOverrideService>;

  let timeRangeFilterSignal: WritableSignal<TimeRangeFilter | null>;
  let inspectionDataSignal: WritableSignal<unknown>;
  let selectedLogSignal: WritableSignal<unknown>;
  let selectedRevisionSignal: WritableSignal<unknown>;

  beforeEach(async () => {
    timeRangeFilterSignal = signal<TimeRangeFilter | null>(null);
    inspectionDataSignal = signal<unknown>(null);
    selectedLogSignal = signal<unknown>(null);
    selectedRevisionSignal = signal<unknown>(null);

    mockViewStateService = jasmine.createSpyObj(
      'ViewStateService',
      [
        'isScaleInitializedForData',
        'setScaleInitializedForData',
        'setPixelPerTime',
        'setTimeOffset',
      ],
      {
        pixelPerTime: of(1e-8),
        timeOffset: of(0n),
        timezoneShift: of(0),
        timeRangeFilter: timeRangeFilterSignal,
      },
    );

    mockInspectionDataStore = jasmine.createSpyObj('InspectionDataStore', [], {
      inspectionData: inspectionDataSignal,
      timelineView: signal(null),
    });

    mockSelectionManager = jasmine.createSpyObj(
      'SelectionManager',
      [
        'onHighlightTimeline',
        'onSelectTimeline',
        'onHighlightLog',
        'onHighlightRevision',
        'onSelectRevision',
        'onSelectEvent',
      ],
      {
        highlightedTimeline: signal(null),
        selectedTimeline: signal(null),
        highlightedChildrenOfSelectedTimeline: signal(null),
        selectedLogIndex: signal(undefined),
        highlightedLogIndices: signal(null),
        selectedLog: selectedLogSignal,
        selectedRevision: selectedRevisionSignal,
        highlightedRevision: signal(null),
        highlightedLogs: signal([]),
      },
    );

    mockStyleOverrideService = jasmine.createSpyObj(
      'StyleOverrideService',
      [],
      {
        severities: [],
        timelineTypes: [],
        subtypes: [],
      },
    );

    await TestBed.configureTestingModule({
      imports: [TimelineSmartComponent],
      providers: [
        { provide: ViewStateService, useValue: mockViewStateService },
        { provide: InspectionDataStore, useValue: mockInspectionDataStore },
        { provide: SelectionManager, useValue: mockSelectionManager },
        { provide: StyleOverrideService, useValue: mockStyleOverrideService },
      ],
    })
      .overrideComponent(TimelineSmartComponent, {
        set: {
          imports: [],
          template: '',
        },
      })
      .compileComponents();

    fixture = TestBed.createComponent(TimelineSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should expose timeRangeFilter signal from ViewStateService', () => {
    expect(component['timeRangeFilter']()).toBeNull();

    timeRangeFilterSignal.set({ startTime: 100n, endTime: 200n });
    expect(component['timeRangeFilter']()).toEqual({
      startTime: 100n,
      endTime: 200n,
    });
  });

  it('should update viewStateService.timeRangeFilter when onTimeRangeSelected is invoked', () => {
    const testRange: TimeRangeFilter = {
      startTime: 1000n,
      endTime: 2000n,
    };
    component['onTimeRangeSelected'](testRange);

    expect(timeRangeFilterSignal()).toEqual(testRange);
  });

  it('should set viewStateService.timeRangeFilter to null when onTimeRangeCleared is invoked', () => {
    timeRangeFilterSignal.set({ startTime: 1000n, endTime: 2000n });
    component['onTimeRangeCleared']();

    expect(timeRangeFilterSignal()).toBeNull();
  });

  it('should exclude events from lookupPrecedingLogs and lookupSucceedingLogs when Alt is held', () => {
    const mockRevision = { changedTime: 1000n } as never;
    const mockEvent1 = { timestamp: 1500n } as never;
    const mockEvent2 = { timestamp: 1600n } as never;
    const mockNextRevision = { changedTime: 2500n } as never;
    const mockNextEvent = { timestamp: 2200n } as never;

    const mockTimeline = {
      id: 1,
      events: [mockEvent1, mockEvent2],
      revisions: [mockRevision],
      lookupEventsInRangeNs: () => [mockNextEvent],
      lookupRevisionsInRangeNs: () => [mockNextRevision],
      lookupRevisionAtNs: () => null,
    } as never;

    expect(
      component['lookupPrecedingLogs'](mockTimeline, 2000n, false),
    ).toEqual([mockEvent1, mockEvent2]);
    expect(component['lookupPrecedingLogs'](mockTimeline, 2000n, true)).toEqual(
      [mockRevision],
    );

    expect(
      component['lookupSucceedingLogs'](mockTimeline, 2000n, 10, false),
    ).toEqual([mockNextEvent, mockNextRevision]);
    expect(
      component['lookupSucceedingLogs'](mockTimeline, 2000n, 10, true),
    ).toEqual([mockNextRevision]);
  });

  it('should compute cursorTimeNs from selectedRevision.changedTime when a revision is selected', () => {
    selectedLogSignal.set({ timestamp: 5000000000n });
    expect(component['cursorTimeNs']()).toBe(5000000000n);
    expect(component['secondaryCursorTimeNs']()).toBe(0n);

    selectedRevisionSignal.set({ changedTime: 1200000000n });
    expect(component['cursorTimeNs']()).toBe(1200000000n);
    expect(component['secondaryCursorTimeNs']()).toBe(5000000000n);

    selectedRevisionSignal.set({ changedTime: 5000000000n });
    expect(component['cursorTimeNs']()).toBe(5000000000n);
    expect(component['secondaryCursorTimeNs']()).toBe(0n);
  });

  it('should set hoveredTime to revision.changedTime when hovering over a revision inside the viewport', () => {
    inspectionDataSignal.set({
      metadata: {
        header: {
          startTimeUnixSeconds: 1,
          endTimeUnixSeconds: 10,
        },
      },
    });
    const mockLog = { timestamp: 5000000000n };
    const mockRevision = {
      changedTime: 1200000000n,
      getEndNs: () => 6000000000n,
      log: mockLog,
    };
    const mockTimeline = {
      id: 1,
      revisions: [mockRevision],
      events: [],
    };

    component['hoverOnTimelineItem']({
      event: new MouseEvent('mousemove'),
      timeline: mockTimeline as never,
      revisionIndex: 0,
      timeNs: 1_500_000_000n,
      clientX: 100,
      clientY: 200,
    });

    expect(mockSelectionManager.onHighlightRevision).toHaveBeenCalledWith(
      mockRevision as never,
    );
    expect(component['hoveredTime']()).toBe(1200000000n);
  });
});
