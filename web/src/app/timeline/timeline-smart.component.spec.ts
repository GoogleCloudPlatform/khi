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

  beforeEach(async () => {
    timeRangeFilterSignal = signal<TimeRangeFilter | null>(null);

    mockViewStateService = jasmine.createSpyObj(
      'ViewStateService',
      [
        'isScaleInitializedForData',
        'setScaleInitializedForData',
        'setPixelPerTime',
        'setTimeOffset',
      ],
      {
        pixelPerTime: of(0.01),
        timeOffset: of(0),
        timezoneShift: of(0),
        timeRangeFilter: timeRangeFilterSignal,
      },
    );

    mockInspectionDataStore = jasmine.createSpyObj('InspectionDataStore', [], {
      inspectionData: signal(null),
      timelineView: signal(null),
    });

    mockSelectionManager = jasmine.createSpyObj(
      'SelectionManager',
      [
        'onHighlightTimeline',
        'onSelectTimeline',
        'onHighlightLog',
        'onSelectRevision',
        'onSelectEvent',
      ],
      {
        highlightedTimeline: signal(null),
        selectedTimeline: signal(null),
        highlightedChildrenOfSelectedTimeline: signal(null),
        selectedLogIndex: signal(undefined),
        highlightedLogIndices: signal(null),
        selectedLog: signal(null),
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
});
