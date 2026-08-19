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
import { BreakpointObserver } from '@angular/cdk/layout';
import { ViewStateService } from 'src/app/services/view-state.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { CelValidationClientService } from 'src/app/services/api/cel/cel-validation-client.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import {
  compileLogFiltersToCel,
  compileFiltersToCel,
  compileExclusionFiltersToCel,
  TimelineToolbarSmartComponent,
} from './timeline-toolbar-smart.component';
import { TimelineFilterConfig } from 'src/app/timeline-toolbar/types/filter-config';

describe('TimelineToolbarSmart compilation helpers', () => {
  describe('compileLogFiltersToCel', () => {
    it('should return empty string when severity is ANY and searchQuery is empty', () => {
      expect(compileLogFiltersToCel('ANY', '')).toBe('');
      expect(compileLogFiltersToCel('ANY', '   ')).toBe('');
    });

    it('should compile severity filter when severity is not ANY', () => {
      expect(compileLogFiltersToCel('INFO', '')).toBe('severity >= INFO');
      expect(compileLogFiltersToCel('ERROR', '')).toBe('severity >= ERROR');
    });

    it('should compile search query filter when searchQuery is provided', () => {
      expect(compileLogFiltersToCel('ANY', 'hello')).toBe('body("hello")');
    });

    it('should join severity and search query with &&', () => {
      expect(compileLogFiltersToCel('WARNING', 'my-query')).toBe(
        'severity >= WARNING && body("my-query")',
      );
    });

    it('should escape double quotes and backslashes in search query', () => {
      expect(compileLogFiltersToCel('ANY', 'hello "world"')).toBe(
        'body("hello \\"world\\"")',
      );
      expect(compileLogFiltersToCel('ANY', 'path\\to\\file')).toBe(
        'body("path\\\\to\\\\file")',
      );
    });
  });

  describe('compileFiltersToCel', () => {
    it('should return empty string when no filters are provided', () => {
      expect(compileFiltersToCel([])).toBe('');
    });

    it('should compile regex filter with * type', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: '*',
          mode: 'regex',
          value: 'pod-.*',
          action: 'include',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe('match("pod-.*")');
    });

    it('should compile regex filter with specific type', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'regex',
          value: 'pod-.*',
          action: 'include',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "pod-.*")',
      );
    });

    it('should escape quotes and backslashes in regex mode', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'regex',
          value: 'test"val\\ue',
          action: 'include',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "test\\"val\\\\ue")',
      );
    });

    it('should compile selection filter by escaping special regex characters and wrapping in anchor', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'selection',
          value: 'pod-a|pod.b|pod+c',
          action: 'include',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "^(?:pod-a|pod\\.b|pod\\+c)$")',
      );
    });

    it('should escape quotes and backslashes in selection mode', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'selection',
          value: 'val"ue\\1|val"ue\\2',
          action: 'include',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "^(?:val\\"ue\\\\\\\\1|val\\"ue\\\\\\\\2)$")',
      );
    });

    it('should ignore exclude filters in compileFiltersToCel', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'regex',
          value: 'pod-.*',
          action: 'include',
        },
        {
          id: '2',
          timelineType: '*',
          mode: 'selection',
          value: 'ns-1|ns-2',
          action: 'exclude',
        },
      ];
      expect(compileFiltersToCel(filters)).toBe(
        'match("K8sResource", "pod-.*")',
      );
    });

    it('should prepend minSeverity when severity is not ANY', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: '*',
          mode: 'regex',
          value: 'pod-.*',
          action: 'include',
        },
      ];
      expect(compileFiltersToCel(filters, 'ERROR')).toBe(
        'minSeverity(ERROR) && match("pod-.*")',
      );
    });

    it('should return minSeverity only when filters is empty and severity is not ANY', () => {
      expect(compileFiltersToCel([], 'ERROR')).toBe('minSeverity(ERROR)');
    });
  });

  describe('compileExclusionFiltersToCel', () => {
    it('should return empty string when no exclude filters exist', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: '*',
          mode: 'regex',
          value: 'pod-.*',
          action: 'include',
        },
      ];
      expect(compileExclusionFiltersToCel(filters)).toBe('');
    });

    it('should compile exclude filters to match', () => {
      const filters: TimelineFilterConfig[] = [
        {
          id: '1',
          timelineType: 'K8sResource',
          mode: 'regex',
          value: 'pod-.*',
          action: 'exclude',
        },
        {
          id: '2',
          timelineType: '*',
          mode: 'selection',
          value: 'ns-1',
          action: 'exclude',
        },
      ];
      expect(compileExclusionFiltersToCel(filters)).toBe(
        'match("K8sResource", "pod-.*") || match("^(?:ns-1)$")',
      );
    });
  });
});

describe('TimelineToolbarSmartComponent', () => {
  let component: TimelineToolbarSmartComponent;
  let fixture: ComponentFixture<TimelineToolbarSmartComponent>;

  let mockCelValidationClient: jasmine.SpyObj<CelValidationClientService>;
  let mockViewStateService: jasmine.SpyObj<ViewStateService>;
  let mockInspectionDataStore: jasmine.SpyObj<InspectionDataStore>;
  let mockBackendFilter: { updateFilterParams: jasmine.Spy };

  let isAdvancedModeSignal: WritableSignal<boolean>;
  let advancedTimelineIncludeCelSignal: WritableSignal<string>;
  let advancedTimelineExcludeCelSignal: WritableSignal<string>;
  let advancedLogCelSignal: WritableSignal<string>;
  let standardTimelineFiltersSignal: WritableSignal<TimelineFilterConfig[]>;
  let standardSelectedSeveritySignal: WritableSignal<string>;
  let standardLogSearchQuerySignal: WritableSignal<string>;

  beforeEach(async () => {
    mockCelValidationClient = jasmine.createSpyObj(
      'CelValidationClientService',
      ['validateTimelineQuery', 'validateLogQuery'],
    );
    mockCelValidationClient.validateTimelineQuery.and.returnValue(
      Promise.resolve({ valid: true, errorMessage: '' }),
    );
    mockCelValidationClient.validateLogQuery.and.returnValue(
      Promise.resolve({ valid: true, errorMessage: '' }),
    );

    isAdvancedModeSignal = signal(false);
    advancedTimelineIncludeCelSignal = signal('');
    advancedTimelineExcludeCelSignal = signal('');
    advancedLogCelSignal = signal('');
    standardTimelineFiltersSignal = signal([]);
    standardSelectedSeveritySignal = signal('ANY');
    standardLogSearchQuerySignal = signal('');

    mockBackendFilter = {
      updateFilterParams: jasmine.createSpy('updateFilterParams'),
    };

    const mockTimelineView = {
      isFiltering: signal(false),
      progress: signal(null),
      backendFilter: mockBackendFilter,
    };

    mockInspectionDataStore = jasmine.createSpyObj('InspectionDataStore', [], {
      timelineView: signal(mockTimelineView),
      inspectionData: signal(null),
    });

    mockViewStateService = jasmine.createSpyObj(
      'ViewStateService',
      ['setTimezoneShift', 'setHideTimelinesWithoutMatchingLogs'],
      {
        isAdvancedMode: isAdvancedModeSignal,
        activeSearchScope: signal(0),
        timezoneShift: of(0),
        standardSelectedSeverity: standardSelectedSeveritySignal,
        standardLogSearchQuery: standardLogSearchQuerySignal,
        standardTimelineFilters: standardTimelineFiltersSignal,
        advancedTimelineIncludeCel: advancedTimelineIncludeCelSignal,
        advancedTimelineExcludeCel: advancedTimelineExcludeCelSignal,
        advancedLogCel: advancedLogCelSignal,
        hideTimelinesWithoutMatchingLogs: of(true),
      },
    );

    await TestBed.configureTestingModule({
      imports: [TimelineToolbarSmartComponent],
      providers: [
        { provide: ViewStateService, useValue: mockViewStateService },
        { provide: InspectionDataStore, useValue: mockInspectionDataStore },
        {
          provide: CelValidationClientService,
          useValue: mockCelValidationClient,
        },
        {
          provide: SelectionManager,
          useValue: {
            selectedTimeline: signal(null),
            selectedLog: signal(null),
          },
        },
        {
          provide: BreakpointObserver,
          useValue: {
            observe: () => of({ matches: true, breakpoints: {} }),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TimelineToolbarSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should synchronize standard filter changes in standard mode', () => {
    mockBackendFilter.updateFilterParams.calls.reset();
    standardSelectedSeveritySignal.set('ERROR');
    fixture.detectChanges();

    expect(mockBackendFilter.updateFilterParams).toHaveBeenCalledWith(
      jasmine.objectContaining({
        timelineQuery: 'minSeverity(ERROR)',
        logQuery: 'severity >= ERROR',
      }),
    );
  });

  it('should validate and apply valid timeline query in advanced mode after debounce', async () => {
    isAdvancedModeSignal.set(true);
    fixture.detectChanges();
    mockBackendFilter.updateFilterParams.calls.reset();

    mockCelValidationClient.validateTimelineQuery.and.returnValue(
      Promise.resolve({ valid: true, errorMessage: '' }),
    );

    advancedTimelineIncludeCelSignal.set('match("pod-1")');
    fixture.detectChanges();
    await new Promise((resolve) => setTimeout(resolve, 250));

    expect(mockCelValidationClient.validateTimelineQuery).toHaveBeenCalledWith(
      'match("pod-1")',
    );
    expect(mockBackendFilter.updateFilterParams).toHaveBeenCalledWith({
      timelineQuery: 'match("pod-1")',
    });
  });

  it('should not apply invalid timeline query to backend filter in advanced mode', async () => {
    isAdvancedModeSignal.set(true);
    fixture.detectChanges();
    mockBackendFilter.updateFilterParams.calls.reset();

    mockCelValidationClient.validateTimelineQuery.and.returnValue(
      Promise.resolve({ valid: false, errorMessage: 'Syntax error at 1:5' }),
    );

    advancedTimelineIncludeCelSignal.set('invalid(');
    fixture.detectChanges();
    await new Promise((resolve) => setTimeout(resolve, 250));

    expect(mockCelValidationClient.validateTimelineQuery).toHaveBeenCalledWith(
      'invalid(',
    );
    expect(mockBackendFilter.updateFilterParams).not.toHaveBeenCalledWith(
      jasmine.objectContaining({ timelineQuery: 'invalid(' }),
    );
  });

  it('should validate and apply valid log query in advanced mode after debounce', async () => {
    isAdvancedModeSignal.set(true);
    fixture.detectChanges();
    mockBackendFilter.updateFilterParams.calls.reset();

    mockCelValidationClient.validateLogQuery.and.returnValue(
      Promise.resolve({ valid: true, errorMessage: '' }),
    );

    advancedLogCelSignal.set('severity >= ERROR');
    fixture.detectChanges();
    await new Promise((resolve) => setTimeout(resolve, 250));

    expect(mockCelValidationClient.validateLogQuery).toHaveBeenCalledWith(
      'severity >= ERROR',
    );
    expect(mockBackendFilter.updateFilterParams).toHaveBeenCalledWith({
      logQuery: 'severity >= ERROR',
    });
  });

  it('should not apply invalid log query to backend filter in advanced mode', async () => {
    isAdvancedModeSignal.set(true);
    fixture.detectChanges();
    mockBackendFilter.updateFilterParams.calls.reset();

    mockCelValidationClient.validateLogQuery.and.returnValue(
      Promise.resolve({ valid: false, errorMessage: 'Syntax error at 1:1' }),
    );

    advancedLogCelSignal.set('invalid[');
    fixture.detectChanges();
    await new Promise((resolve) => setTimeout(resolve, 250));

    expect(mockCelValidationClient.validateLogQuery).toHaveBeenCalledWith(
      'invalid[',
    );
    expect(mockBackendFilter.updateFilterParams).not.toHaveBeenCalledWith(
      jasmine.objectContaining({ logQuery: 'invalid[' }),
    );
  });
});
