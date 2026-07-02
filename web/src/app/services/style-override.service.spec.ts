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
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { StyleStore } from 'src/app/store/domain/style-store';
import { StyleOverrideService } from 'src/app/services/style-override.service';
import { RevisionStateStyle } from 'src/app/store/domain/style';

import { InspectionData } from 'src/app/store/domain/inspection-data';

describe('StyleOverrideService', () => {
  let service: StyleOverrideService;
  let mockInspectionDataStore: jasmine.SpyObj<InspectionDataStore>;
  let styleStore: StyleStore;

  const originalColor = { r: 1, g: 0, b: 0, a: 1 };
  const mockState = {
    id: 1,
    label: 'Terminated',
    icon: 'cancel',
    description: 'Resource instance terminated',
    backgroundColor: originalColor,
    style: RevisionStateStyle.NORMAL,
  };

  const mockTimelineType = {
    id: 1,
    label: 'Pod',
    description: 'Pod resources',
    icon: 'pod',
    backgroundColor: originalColor,
    foregroundColor: originalColor,
    typeChipBackgroundColor: originalColor,
    typeChipForegroundColor: originalColor,
    visible: true,
    sortPriority: 10,
    height: 30,
  };

  const mockLogType = {
    id: 1,
    label: 'Audit',
    description: 'Audit log events',
    backgroundColor: originalColor,
    foregroundColor: originalColor,
  };

  beforeEach(() => {
    styleStore = new StyleStore();
    styleStore.addRevisionStates([mockState]);
    styleStore.addTimelineTypes([mockTimelineType]);
    styleStore.addLogTypes([mockLogType]);

    mockInspectionDataStore = jasmine.createSpyObj<InspectionDataStore>(
      'InspectionDataStore',
      [],
      {
        inspectionData: signal({
          styleStore,
        } as unknown as InspectionData),
      },
    );

    TestBed.configureTestingModule({
      providers: [
        StyleOverrideService,
        {
          provide: InspectionDataStore,
          useValue: mockInspectionDataStore,
        },
      ],
    });

    service = TestBed.inject(StyleOverrideService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should return base styles when no overrides are applied', () => {
    expect(service.revisionStates.length).toBe(1);
    expect(service.revisionStates[0]).toEqual(mockState);
    expect(service.getRevisionState(1)).toEqual(mockState);
    expect(service.isRevisionStateOverridden(1)).toBeFalse();

    expect(service.timelineTypes.length).toBe(1);
    expect(service.timelineTypes[0]).toEqual(mockTimelineType);
    expect(service.getTimelineType(1)).toEqual(mockTimelineType);
    expect(service.isTimelineTypeOverridden(1)).toBeFalse();

    expect(service.logTypes.length).toBe(1);
    expect(service.logTypes[0]).toEqual(mockLogType);
    expect(service.getLogType(1)).toEqual(mockLogType);
    expect(service.isLogTypeOverridden(1)).toBeFalse();
  });

  it('should override and reset revision state styles correctly', () => {
    const overriddenState = {
      ...mockState,
      backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
    };

    service.overrideRevisionState(overriddenState);

    expect(service.isRevisionStateOverridden(1)).toBeTrue();
    expect(service.revisionStates[0]).toEqual(overriddenState);
    expect(service.getRevisionState(1)).toEqual(overriddenState);
    expect(service.stylesUpdated()).toBe(1);

    service.resetRevisionState(1);

    expect(service.isRevisionStateOverridden(1)).toBeFalse();
    expect(service.revisionStates[0]).toEqual(mockState);
    expect(service.getRevisionState(1)).toEqual(mockState);
    expect(service.stylesUpdated()).toBe(2);
  });

  it('should override and reset timeline type styles correctly', () => {
    const overriddenType = {
      ...mockTimelineType,
      backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
    };

    service.overrideTimelineType(overriddenType);

    expect(service.isTimelineTypeOverridden(1)).toBeTrue();
    expect(service.timelineTypes[0]).toEqual(overriddenType);
    expect(service.getTimelineType(1)).toEqual(overriddenType);
    expect(service.stylesUpdated()).toBe(1);

    service.resetTimelineType(1);

    expect(service.isTimelineTypeOverridden(1)).toBeFalse();
    expect(service.timelineTypes[0]).toEqual(mockTimelineType);
    expect(service.getTimelineType(1)).toEqual(mockTimelineType);
    expect(service.stylesUpdated()).toBe(2);
  });

  it('should override and reset log type styles correctly', () => {
    const overriddenType = {
      ...mockLogType,
      backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
      foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
    };

    service.overrideLogType(overriddenType);

    expect(service.isLogTypeOverridden(1)).toBeTrue();
    expect(service.logTypes[0]).toEqual(overriddenType);
    expect(service.getLogType(1)).toEqual(overriddenType);
    expect(service.stylesUpdated()).toBe(1);

    service.resetLogType(1);

    expect(service.isLogTypeOverridden(1)).toBeFalse();
    expect(service.logTypes[0]).toEqual(mockLogType);
    expect(service.getLogType(1)).toEqual(mockLogType);
    expect(service.stylesUpdated()).toBe(2);
  });
});
