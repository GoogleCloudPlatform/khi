/**
 * Copyright 2024 Google LLC
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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef } from '@angular/material/dialog';
import { signal } from '@angular/core';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

import {
  NewInspectionDialogComponent,
  computeTotalEstimatedLogs,
  TotalEstimatedLogsSeverity,
} from './new-inspection.component';
import { BACKEND_API } from 'src/app/services/api/backend-api-interface';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { InspectionMetadataQuery } from 'src/app/common/schema/metadata-types';

import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';

describe('NewInspectionDialogTest', () => {
  let component: NewInspectionDialogComponent;
  let fixture: ComponentFixture<NewInspectionDialogComponent>;
  beforeEach(async () => {
    const inspectionTypesSignal = signal({ types: [] });
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      providers: [
        {
          provide: MatDialogRef,
          useValue: null,
        },
        {
          provide: BACKEND_API,
          useValue: {},
        },
        {
          provide: BACKEND_SYNC,
          useValue: {
            inspectionTypes: {
              value: inspectionTypesSignal,
            },
          },
        },
        {
          provide: EXTENSION_STORE,
          useValue: new ExtensionStore(),
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(NewInspectionDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('computeTotalEstimatedLogs', () => {
    it('should return undefined for empty or undefined query list', () => {
      expect(computeTotalEstimatedLogs(undefined)).toBeUndefined();
      expect(computeTotalEstimatedLogs([])).toBeUndefined();
    });

    it('should calculate complete total with Normal severity when < 1,000,000', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 500 },
        { id: 'q2', name: 'q2', query: 'query2', estimatedCount: 1500 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 2000,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~2,000 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should calculate complete total with 0 logs', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 0 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~0 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should format partial estimate with > prefix when some queries are in-flight', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1250 },
        { id: 'q2', name: 'q2', query: 'query2' },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1250,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: '>1,250 logs estimated so far',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should display Estimating total logs... when all queries are unestimated', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1' },
        { id: 'q2', name: 'q2', query: 'query2' },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: 'Estimating total logs...',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should assign Warning severity for counts between 1,000,000 and 4,999,999', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1200000 },
        { id: 'q2', name: 'q2', query: 'query2', estimatedCount: 300000 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1500000,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~1,500,000 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Warning,
      });
    });

    it('should assign Danger severity for counts >= 5,000,000', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 5000000 },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 5000000,
        isComplete: true,
        isEstimating: false,
        isIncomplete: false,
        displayText: '~5,000,000 total logs estimated',
        severity: TotalEstimatedLogsSeverity.Danger,
      });
    });

    it('should assign Danger severity for partial estimates >= 5,000,000', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 6500000 },
        { id: 'q2', name: 'q2', query: 'query2' },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 6500000,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: '>6,500,000 logs estimated so far',
        severity: TotalEstimatedLogsSeverity.Danger,
      });
    });

    it('should display Incomplete parameters when all queries are incomplete with no counts', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', incomplete: true },
        { id: 'q2', name: 'q2', query: 'query2', incomplete: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 0,
        isComplete: false,
        isEstimating: false,
        isIncomplete: true,
        displayText: 'Incomplete parameters',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should display >N logs estimated (some parameters incomplete) when some queries are estimated and others are incomplete', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 2500 },
        { id: 'q2', name: 'q2', query: 'query2', incomplete: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 2500,
        isComplete: false,
        isEstimating: false,
        isIncomplete: true,
        displayText: '>2,500 logs estimated (some parameters incomplete)',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });

    it('should set isEstimating to true when some queries are incomplete and others are still estimating', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1000 },
        { id: 'q2', name: 'q2', query: 'query2' },
        { id: 'q3', name: 'q3', query: 'query3', incomplete: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1000,
        isComplete: false,
        isEstimating: true,
        isIncomplete: true,
        displayText: '>1,000 logs estimated (some parameters incomplete)',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });
  });
});
