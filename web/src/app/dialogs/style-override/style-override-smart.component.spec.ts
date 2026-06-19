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
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StyleOverrideService } from 'src/app/services/style-override.service';
import { StyleOverrideSmartComponent } from 'src/app/dialogs/style-override/style-override-smart.component';
import {
  RevisionStateStyle,
  TimelineType,
  LogType,
} from 'src/app/store/domain/style';
import { MatDialogModule } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('StyleOverrideSmartComponent', () => {
  let component: StyleOverrideSmartComponent;
  let fixture: ComponentFixture<StyleOverrideSmartComponent>;
  let mockStyleOverrideService: jasmine.SpyObj<StyleOverrideService>;

  const originalColor = { r: 1, g: 0, b: 0, a: 1 };
  const mockState = {
    id: 1,
    label: 'State 1',
    icon: 'check',
    description: 'First State',
    backgroundColor: originalColor,
    style: RevisionStateStyle.NORMAL,
  };

  const mockTimelineType: TimelineType = {
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
    height: 1.2,
  };

  const mockLogType: LogType = {
    id: 1,
    label: 'Audit',
    description: 'Audit events',
    backgroundColor: originalColor,
    foregroundColor: originalColor,
  };

  beforeEach(async () => {
    mockStyleOverrideService = jasmine.createSpyObj<StyleOverrideService>(
      'StyleOverrideService',
      [
        'overrideRevisionState',
        'resetRevisionState',
        'isRevisionStateOverridden',
        'getRevisionState',
        'overrideTimelineType',
        'resetTimelineType',
        'isTimelineTypeOverridden',
        'getTimelineType',
        'overrideLogType',
        'resetLogType',
        'isLogTypeOverridden',
        'getLogType',
      ],
      {
        stylesUpdated: signal(0),
        revisionStates: [mockState],
        timelineTypes: [mockTimelineType],
        logTypes: [mockLogType],
      },
    );

    mockStyleOverrideService.getRevisionState.and.returnValue(mockState);
    mockStyleOverrideService.getTimelineType.and.returnValue(mockTimelineType);
    mockStyleOverrideService.getLogType.and.returnValue(mockLogType);
    mockStyleOverrideService.isRevisionStateOverridden.and.returnValue(false);
    mockStyleOverrideService.isTimelineTypeOverridden.and.returnValue(false);
    mockStyleOverrideService.isLogTypeOverridden.and.returnValue(false);

    await TestBed.configureTestingModule({
      imports: [
        StyleOverrideSmartComponent,
        MatDialogModule,
        NoopAnimationsModule,
      ],
      providers: [
        {
          provide: StyleOverrideService,
          useValue: mockStyleOverrideService,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(StyleOverrideSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should map revision states to view models correctly', () => {
    const vms = component['revisionStateViewModels']();
    expect(vms.length).toBe(1);
    expect(vms[0]).toEqual({
      id: 1,
      label: 'State 1',
      icon: 'check',
      description: 'First State',
      hexColor: '#ff0000',
      isOverridden: false,
      style: RevisionStateStyle.NORMAL,
      goColorCode: 'style.Color{R: 1.000, G: 0.000, B: 0.000, A: 1.0}',
    });
  });

  it('should map timeline types to view models correctly', () => {
    const vms = component['timelineTypeViewModels']();
    expect(vms.length).toBe(1);
    expect(vms[0]).toEqual({
      id: 1,
      label: 'Pod',
      icon: 'pod',
      description: 'Pod resources',
      hexColor: '#ff0000',
      hexForegroundColor: '#ff0000',
      hexChipBackgroundColor: '#ff0000',
      hexChipForegroundColor: '#ff0000',
      height: 1.2,
      isOverridden: false,
      goColorCode: 'style.Color{R: 1.000, G: 0.000, B: 0.000, A: 1.0}',
    });
  });

  it('should map log types to view models correctly', () => {
    const vms = component['logTypeViewModels']();
    expect(vms.length).toBe(1);
    expect(vms[0]).toEqual({
      id: 1,
      label: 'Audit',
      description: 'Audit events',
      hexColor: '#ff0000',
      hexForegroundColor: '#ff0000',
      isOverridden: false,
      goColorCode: 'style.Color{R: 1.000, G: 0.000, B: 0.000, A: 1.0}',
    });
  });

  it('should call overrideRevisionState onRevisionStateColorChange', () => {
    component['onRevisionStateColorChange']({ id: 1, hexColor: '#00ff00' });
    expect(mockStyleOverrideService.overrideRevisionState).toHaveBeenCalledWith(
      {
        ...mockState,
        backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
      },
    );
  });

  it('should call resetRevisionState onRevisionStateResetColor', () => {
    component['onRevisionStateResetColor'](1);
    expect(mockStyleOverrideService.resetRevisionState).toHaveBeenCalledWith(1);
  });

  it('should call overrideTimelineType onTimelineTypePropertyChange', () => {
    component['onTimelineTypePropertyChange']({
      id: 1,
      backgroundColor: '#00ff00',
      height: 1.5,
    });
    expect(mockStyleOverrideService.overrideTimelineType).toHaveBeenCalledWith({
      ...mockTimelineType,
      backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
      height: 1.5,
    });
  });

  it('should call resetTimelineType onTimelineTypeResetColor', () => {
    component['onTimelineTypeResetColor'](1);
    expect(mockStyleOverrideService.resetTimelineType).toHaveBeenCalledWith(1);
  });

  it('should call overrideLogType onLogTypePropertyChange', () => {
    component['onLogTypePropertyChange']({
      id: 1,
      backgroundColor: '#00ff00',
    });
    expect(mockStyleOverrideService.overrideLogType).toHaveBeenCalledWith({
      ...mockLogType,
      backgroundColor: { r: 0, g: 1, b: 0, a: 1 },
    });
  });

  it('should call resetLogType onLogTypeResetColor', () => {
    component['onLogTypeResetColor'](1);
    expect(mockStyleOverrideService.resetLogType).toHaveBeenCalledWith(1);
  });
});
