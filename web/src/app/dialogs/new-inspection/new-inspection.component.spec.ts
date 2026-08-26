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
import { Observable, of, Subject } from 'rxjs';

import {
  NewInspectionDialogComponent,
  computeTotalEstimatedLogs,
  TotalEstimatedLogsSeverity,
} from './new-inspection.component';
import { BACKEND_API } from 'src/app/services/api/backend-api-interface';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import {
  InspectionType,
  InspectionDryRunResponse,
} from 'src/app/common/schema/api-types';
import { InspectionMetadataQuery } from 'src/app/common/schema/metadata-types';
import {
  ParameterHintType,
  ParameterInputType,
  ParameterFormValidationTiming,
} from 'src/app/common/schema/form-types';
import {
  PARAMETER_STORE,
  ParameterStore,
} from './components/service/parameter-store';

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

    it('should treat queries with pending = true as estimating', () => {
      const queries: InspectionMetadataQuery[] = [
        { id: 'q1', name: 'q1', query: 'query1', estimatedCount: 1000 },
        { id: 'q2', name: 'q2', query: 'query2', pending: true },
      ];
      const result = computeTotalEstimatedLogs(queries);
      expect(result).toEqual({
        knownCount: 1000,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: '>1,000 logs estimated so far',
        severity: TotalEstimatedLogsSeverity.Normal,
      });
    });
  });

  describe('dryrunLoop', () => {
    let mockDryrunDirect: jasmine.Spy;
    let mockInspectionClient: {
      features: unknown;
      dryrunDirect: jasmine.Spy;
      run: jasmine.Spy;
    };
    let mockApiClient: {
      createInspection: jasmine.Spy;
    };
    let store: ParameterStore;

    beforeEach(() => {
      mockDryrunDirect = jasmine.createSpy('dryrunDirect');
      mockInspectionClient = {
        features: of([]),
        dryrunDirect: mockDryrunDirect,
        run: jasmine.createSpy('run').and.returnValue(of({})),
      };
      mockApiClient = TestBed.inject(BACKEND_API) as unknown as {
        createInspection: jasmine.Spy;
      };
      mockApiClient.createInspection = jasmine
        .createSpy('createInspection')
        .and.returnValue(of(mockInspectionClient));

      store = fixture.debugElement.injector.get(PARAMETER_STORE);
      const testType: InspectionType = {
        id: 'test-type',
        name: 'Test Type',
        icon: '',
        description: '',
      };
      component.setInspectionType(testType);
    });

    it('should start dryrun loop and update parameterViewModel on response with default values not validating', async () => {
      const dryrunResponse: InspectionDryRunResponse = {
        metadata: {
          form: [
            {
              id: 'text-param',
              type: ParameterInputType.Text,
              label: 'Text',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              default: 'default-text',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            },
            {
              id: 'set-param',
              type: ParameterInputType.Set,
              label: 'Set',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              options: [],
              default: ['opt1', 'opt2'],
              allowAddAll: false,
              allowRemoveAll: false,
              allowCustomValue: true,
            },
          ],
          query: [],
          plan: { taskGraph: '' },
          jobCommand: { command: 'test-cmd' },
        },
      };
      mockDryrunDirect.and.returnValue(of(dryrunResponse));

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT,
      );

      await fixture.whenStable();

      expect(mockDryrunDirect).toHaveBeenCalledTimes(2);
      const vm = component.parameterViewModel();
      expect(vm).toBeTruthy();
      expect(vm?.job?.command).toBe('test-cmd');
      expect(store.isValidating('text-param')()).toBe(false);
      expect(store.isValidating('set-param')()).toBe(false);
      expect(store.isDirty('text-param')()).toBe(false);
      expect(store.isDirty('set-param')()).toBe(false);
    });

    it('should track pendingFieldCount and disable run button when a field is server-pending or validating', async () => {
      const dryrunResponse: InspectionDryRunResponse = {
        metadata: {
          form: [
            {
              id: 'text-param',
              type: ParameterInputType.Text,
              label: 'Text',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              default: 'default-text',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
              pending: true,
            },
          ],
          query: [],
          plan: { taskGraph: '' },
          jobCommand: { command: 'test-cmd' },
        },
      };
      mockDryrunDirect.and.returnValue(of(dryrunResponse));

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT,
      );
      await fixture.whenStable();

      expect(component.pendingFieldCount()).toBe(1);
      expect(component.isRunButtonDisabled()).toBe(true);
    });

    it('should suppress stale errors when field is validating', async () => {
      const dryrunResponse: InspectionDryRunResponse = {
        metadata: {
          form: [
            {
              id: 'text-param',
              type: ParameterInputType.Text,
              label: 'Text',
              description: '',
              hint: 'Error message',
              hintType: ParameterHintType.Error,
              default: 'default-text',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            },
          ],
          query: [],
          plan: { taskGraph: '' },
          jobCommand: { command: 'test-cmd' },
        },
      };
      mockDryrunDirect.and.returnValue(of(dryrunResponse));

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT,
      );
      await fixture.whenStable();

      expect(component.errorFieldCount()).toBe(1);
      expect(component.pendingFieldCount()).toBe(0);

      // User changes the value, making it validating on client side
      store.set('text-param', 'new-text');
      fixture.detectChanges();

      // Validating field should suppress the stale error and increase pendingFieldCount
      expect(component.errorFieldCount()).toBe(0);
      expect(component.pendingFieldCount()).toBe(1);
      expect(component.isRunButtonDisabled()).toBe(true);
    });

    it('should keep fields in validating state after defaults are assigned until the next dryrun completes', async () => {
      const dryrunResponse: InspectionDryRunResponse = {
        metadata: {
          form: [
            {
              id: 'text-param',
              type: ParameterInputType.Text,
              label: 'Text',
              description: '',
              hint: '',
              hintType: ParameterHintType.None,
              default: 'default-text',
              readonly: false,
              suggestions: [],
              validationTiming: ParameterFormValidationTiming.Blur,
            },
          ],
          query: [],
          plan: { taskGraph: '' },
          jobCommand: { command: 'test-cmd' },
        },
      };
      const subject1 = new Subject<InspectionDryRunResponse>();
      const subject2 = new Subject<InspectionDryRunResponse>();

      let callCount = 0;
      mockDryrunDirect.and.callFake(() => {
        callCount++;
        if (callCount === 1) {
          return subject1;
        }
        return subject2;
      });

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));
      expect(mockDryrunDirect).toHaveBeenCalledTimes(1);

      // Dryrun 1 response provides defaults
      subject1.next(dryrunResponse);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // After defaults are assigned, text-param must be validating for the next dryrun
      expect(store.isValidating('text-param')()).toBe(true);
      expect(mockDryrunDirect).toHaveBeenCalledTimes(2);

      // Second dryrun completes, validating the assigned defaults
      subject2.next(dryrunResponse);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Once the next dryrun completes, text-param is no longer validating
      expect(store.isValidating('text-param')()).toBe(false);
    });

    it('should discard stale response when parameters change while request is in flight', async () => {
      const subject1 = new Subject<InspectionDryRunResponse>();
      const subject2 = new Subject<InspectionDryRunResponse>();

      let callCount = 0;
      mockDryrunDirect.and.callFake(() => {
        callCount++;
        if (callCount === 1) {
          return subject1;
        }
        return subject2;
      });

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(mockDryrunDirect).toHaveBeenCalledTimes(1);

      // User changes parameter while request 1 is in-flight
      store.set('param1', 'updated-value');

      // Request 1 responds with stale job command
      subject1.next({
        metadata: {
          form: [],
          query: [],
          plan: { taskGraph: '' },
          jobCommand: { command: 'stale-cmd' },
        },
      });
      subject1.complete();

      await new Promise((resolve) => setTimeout(resolve, 10));

      // Loop should have immediately triggered second request with updated params
      expect(mockDryrunDirect).toHaveBeenCalledTimes(2);
      expect(mockDryrunDirect.calls.mostRecent().args[0]).toEqual({
        param1: 'updated-value',
      });
      // The stale result should NOT have been set
      expect(component.parameterViewModel()).toBeNull();

      // Request 2 responds with updated job command
      subject2.next({
        metadata: {
          form: [],
          query: [],
          plan: { taskGraph: '' },
          jobCommand: { command: 'updated-cmd' },
        },
      });
      subject2.complete();

      await new Promise((resolve) => setTimeout(resolve, 10));

      const vm = component.parameterViewModel();
      expect(vm?.job?.command).toBe('updated-cmd');
      expect(store.validatedParameters()['param1']).toBe('updated-value');
    });

    it('should stop dryrun loop when leaving parameter step', async () => {
      mockDryrunDirect.and.returnValue(
        of({
          metadata: {
            form: [],
            query: [],
            plan: { taskGraph: '' },
            jobCommand: { command: 'cmd' },
          },
        }),
      );

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT,
      );
      await fixture.whenStable();

      const initialCalls = mockDryrunDirect.calls.count();
      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_FEATURE_SELECTION,
      );

      await new Promise((resolve) => setTimeout(resolve, 50));
      expect(mockDryrunDirect.calls.count()).toBe(initialCalls);
    });

    it('should unsubscribe from in-flight dryrun request when leaving parameter step', async () => {
      let unsubscribed = false;
      const observable = new Observable<InspectionDryRunResponse>(() => {
        return () => {
          unsubscribed = true;
        };
      });
      mockDryrunDirect.and.returnValue(observable);

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT,
      );
      await fixture.whenStable();
      await new Promise((resolve) => setTimeout(resolve, 20));

      expect(unsubscribed).toBe(false);

      component.selectedStepChange(
        NewInspectionDialogComponent.STEP_INDEX_FEATURE_SELECTION,
      );
      await fixture.whenStable();
      await new Promise((resolve) => setTimeout(resolve, 20));

      expect(unsubscribed).toBe(true);
    });
  });
});
