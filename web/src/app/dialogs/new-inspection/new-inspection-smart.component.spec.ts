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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { signal, WritableSignal } from '@angular/core';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { Observable, of, Subject } from 'rxjs';

import { ParameterInputStepComponent } from 'src/app/dialogs/new-inspection/components/parameter-input-step.component';
import { NewInspectionDialogComponent } from 'src/app/dialogs/new-inspection/new-inspection-smart.component';
import { BACKEND_API } from 'src/app/services/api/backend-api-interface';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import {
  InspectionType,
  InspectionDryRunResponse,
} from 'src/app/common/schema/api-types';
import {
  ParameterHintType,
  ParameterInputType,
  ParameterFormValidationTiming,
} from 'src/app/common/schema/form-types';
import {
  PARAMETER_STORE,
  ParameterStore,
} from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';
import {
  NewInspectionDialogData,
  NewInspectionStepIndex,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';

describe('NewInspectionDialogTest', () => {
  let component: NewInspectionDialogComponent;
  let fixture: ComponentFixture<NewInspectionDialogComponent>;
  let mockDialogRef: { close: jasmine.Spy };

  beforeEach(async () => {
    mockDialogRef = {
      close: jasmine.createSpy('close'),
    };
    const inspectionTypesSignal = signal({ types: [] });
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      providers: [
        {
          provide: MatDialogRef,
          useValue: mockDialogRef,
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

  it('setInspectionType should update currentInspectionType and transition to FeatureSelection step', () => {
    const testType: InspectionType = {
      id: 'test-gke',
      name: 'Test GKE',
      icon: '',
      description: '',
    };
    component.setInspectionType(testType);
    expect(component.currentInspectionType()).toEqual(testType);
    expect(component.selectedStepIndex()).toBe(
      NewInspectionStepIndex.FeatureSelection,
    );
  });

  it('toggleFeature should call client.setFeatures with toggled boolean value', async () => {
    const mockInspectionClient = {
      features: of([
        { id: 'feat-1', name: 'Feature 1', enabled: true },
        { id: 'feat-2', name: 'Feature 2', enabled: false },
      ]),
      setFeatures: jasmine.createSpy('setFeatures'),
      dryrunDirect: jasmine.createSpy('dryrunDirect').and.returnValue(of({})),
      run: jasmine.createSpy('run').and.returnValue(of({})),
    };
    const mockApi = TestBed.inject(BACKEND_API) as unknown as {
      createInspection: jasmine.Spy;
    };
    mockApi.createInspection = jasmine
      .createSpy('createInspection')
      .and.returnValue(of(mockInspectionClient));

    component.setInspectionType({
      id: 'test-gke',
      name: 'Test GKE',
      icon: '',
      description: '',
    });
    fixture.detectChanges();
    await fixture.whenStable();

    component.toggleFeature('feat-1');
    expect(mockInspectionClient.setFeatures).toHaveBeenCalledWith({
      'feat-1': false,
    });

    component.toggleFeature('feat-2');
    expect(mockInspectionClient.setFeatures).toHaveBeenCalledWith({
      'feat-2': true,
    });
  });

  it('onRunButtonClick should call client.run, extension.notifyLifecycleOnInspectionStart, and close dialog', async () => {
    const mockInspectionClient = {
      features: of([]),
      setFeatures: jasmine.createSpy('setFeatures'),
      dryrunDirect: jasmine.createSpy('dryrunDirect').and.returnValue(of({})),
      run: jasmine.createSpy('run').and.returnValue(of({})),
    };
    const mockApi = TestBed.inject(BACKEND_API) as unknown as {
      createInspection: jasmine.Spy;
    };
    mockApi.createInspection = jasmine
      .createSpy('createInspection')
      .and.returnValue(of(mockInspectionClient));

    const extensionStore = TestBed.inject(EXTENSION_STORE);
    spyOn(extensionStore, 'notifyLifecycleOnInspectionStart');

    component.setInspectionType({
      id: 'test-gke',
      name: 'Test GKE',
      icon: '',
      description: '',
    });
    fixture.detectChanges();
    await fixture.whenStable();

    component.onRunButtonClick();
    await fixture.whenStable();

    expect(mockInspectionClient.run).toHaveBeenCalled();
    expect(extensionStore.notifyLifecycleOnInspectionStart).toHaveBeenCalled();
    expect(mockDialogRef.close).toHaveBeenCalledWith({
      inspectionTaskStarted: true,
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

      component.selectedStepChange(NewInspectionStepIndex.ParameterInput);

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

      component.selectedStepChange(NewInspectionStepIndex.ParameterInput);
      await fixture.whenStable();
      fixture.detectChanges();

      const parameterStep = fixture.debugElement.query(
        By.directive(ParameterInputStepComponent),
      ).componentInstance as ParameterInputStepComponent;

      expect(parameterStep.pendingFieldCount()).toBe(1);
      expect(parameterStep.isRunButtonDisabled()).toBe(true);
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

      component.selectedStepChange(NewInspectionStepIndex.ParameterInput);
      await fixture.whenStable();
      fixture.detectChanges();

      const parameterStep = fixture.debugElement.query(
        By.directive(ParameterInputStepComponent),
      ).componentInstance as ParameterInputStepComponent;

      expect(parameterStep.errorFieldCount()).toBe(1);
      expect(parameterStep.pendingFieldCount()).toBe(0);

      // User changes the value, making it validating on client side
      store.set('text-param', 'new-text');
      fixture.detectChanges();

      // Validating field should suppress the stale error and increase pendingFieldCount
      expect(parameterStep.errorFieldCount()).toBe(0);
      expect(parameterStep.pendingFieldCount()).toBe(1);
      expect(parameterStep.isRunButtonDisabled()).toBe(true);
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

      component.selectedStepChange(NewInspectionStepIndex.ParameterInput);
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

      component.selectedStepChange(NewInspectionStepIndex.ParameterInput);
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

      component.selectedStepChange(NewInspectionStepIndex.ParameterInput);
      await fixture.whenStable();

      const initialCalls = mockDryrunDirect.calls.count();
      component.selectedStepChange(NewInspectionStepIndex.FeatureSelection);

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

      component.selectedStepChange(NewInspectionStepIndex.ParameterInput);
      await fixture.whenStable();
      await new Promise((resolve) => setTimeout(resolve, 20));

      expect(unsubscribed).toBe(false);

      component.selectedStepChange(NewInspectionStepIndex.FeatureSelection);
      await fixture.whenStable();
      await new Promise((resolve) => setTimeout(resolve, 20));

      expect(unsubscribed).toBe(true);
    });
  });

  describe('with NewInspectionDialogData', () => {
    let customFixture: ComponentFixture<NewInspectionDialogComponent>;
    let customComponent: NewInspectionDialogComponent;
    let mockClient: {
      features: unknown;
      setFeatures: jasmine.Spy;
      dryrunDirect: jasmine.Spy;
      run: jasmine.Spy;
    };
    let inspectionTypesSignal: WritableSignal<{ types: InspectionType[] }>;

    beforeEach(async () => {
      TestBed.resetTestingModule();
      mockClient = {
        features: of([
          { id: 'feature-1', enabled: true },
          { id: 'feature-2', enabled: true },
        ]),
        setFeatures: jasmine.createSpy('setFeatures'),
        dryrunDirect: jasmine.createSpy('dryrunDirect').and.returnValue(
          of({
            metadata: {
              form: [],
              query: [],
              plan: { taskGraph: '' },
            },
          }),
        ),
        run: jasmine.createSpy('run'),
      };
      const mockApi = {
        createInspection: jasmine
          .createSpy('createInspection')
          .and.returnValue(of(mockClient)),
      };
      inspectionTypesSignal = signal({
        types: [
          {
            id: 'gke',
            name: 'GKE Inspection',
            icon: '',
            description: '',
          },
        ],
      });

      const dialogData: NewInspectionDialogData = {
        initialInspectionTypeId: 'gke',
        initialFeatureIds: ['feature-1', 'feature-2'],
        initialParameters: { cluster: 'cluster-1' },
      };

      await TestBed.configureTestingModule({
        imports: [NoopAnimationsModule],
        providers: [
          {
            provide: MatDialogRef,
            useValue: null,
          },
          {
            provide: MAT_DIALOG_DATA,
            useValue: dialogData,
          },
          {
            provide: BACKEND_API,
            useValue: mockApi,
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

      customFixture = TestBed.createComponent(NewInspectionDialogComponent);
      customComponent = customFixture.componentInstance;
      spyOn(
        customComponent as unknown as { startDryrunLoop: () => void },
        'startDryrunLoop',
      ).and.stub();
      customFixture.detectChanges();
    });

    afterEach(() => {
      customFixture?.destroy();
      TestBed.resetTestingModule();
    });

    it('should preselect inspection type, prefill parameters, and enable features', async () => {
      await customFixture.whenStable();
      const currentType = customComponent.currentInspectionType();
      expect(currentType?.id).toBe('gke');

      const store = customFixture.debugElement.injector.get(PARAMETER_STORE);
      expect(store.currentParameters()['cluster']).toBe('cluster-1');
      expect(mockClient.setFeatures).toHaveBeenCalledWith({
        'feature-1': true,
        'feature-2': true,
      });
      expect(customComponent.selectedStepIndex()).toBe(
        NewInspectionStepIndex.ParameterInput,
      );
    });
  });
});
