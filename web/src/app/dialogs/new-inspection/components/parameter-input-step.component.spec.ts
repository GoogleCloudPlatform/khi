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
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatProgressBar } from '@angular/material/progress-bar';
import {
  ParameterFormValidationTiming,
  ParameterHintType,
  ParameterInputType,
  TextParameterFormField,
} from 'src/app/common/schema/form-types';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
  ParameterStore,
} from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import {
  ParameterStepViewModel,
  TotalEstimatedLogsSeverity,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import { ParameterInputStepComponent } from 'src/app/dialogs/new-inspection/components/parameter-input-step.component';
import { JobCommandComponent } from 'src/app/dialogs/new-inspection/components/job-command.component';

describe('ParameterInputStepComponent', () => {
  let fixture: ComponentFixture<ParameterInputStepComponent>;
  let store: ParameterStore;

  const mockParameterViewModel: ParameterStepViewModel = {
    rootGroupForm: {
      id: 'root',
      label: 'Root Group',
      type: ParameterInputType.Group,
      description: 'Root group description',
      hint: '',
      hintType: ParameterHintType.None,
      collapsible: false,
      collapsedByDefault: false,
      children: [],
    },
    queries: [
      {
        id: 'q1',
        name: 'Audit Log Query',
        query: 'resource.type="k8s_cluster"',
        estimatedCount: 12345,
      },
    ],
    job: {
      command: 'khi run --target gke',
    },
    fieldCount: 5,
    totalEstimatedSummary: {
      knownCount: 12345,
      isComplete: true,
      isEstimating: false,
      isIncomplete: false,
      displayText: '~12,345 total logs estimated',
      severity: TotalEstimatedLogsSeverity.Normal,
    },
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule, ParameterInputStepComponent],
      providers: [
        {
          provide: PARAMETER_STORE,
          useClass: DefaultParameterStore,
        },
      ],
    }).compileComponents();

    store = TestBed.inject(PARAMETER_STORE);
    fixture = TestBed.createComponent(ParameterInputStepComponent);
    fixture.componentRef.setInput('parameterStore', store);
    fixture.componentRef.setInput(
      'taskGraphDebugUrl',
      '/debug/task-graph?tab=DAG_VIEWER&inspectionType=gke&features=feat-1',
    );
  });

  it('should render loading progress bar when parameterViewModel is null', () => {
    fixture.componentRef.setInput('parameterViewModel', null);
    fixture.detectChanges();

    const progressBar = fixture.debugElement.query(
      By.directive(MatProgressBar),
    );
    expect(progressBar).toBeTruthy();

    const message = fixture.debugElement.query(By.css('.message'));
    expect(message.nativeElement.textContent).toContain(
      'Loading a parameter list for the given setting. Wait a few seconds.',
    );

    const parameterView = fixture.debugElement.query(By.css('.parameter-view'));
    expect(parameterView).toBeNull();
  });

  it('should render queries, total estimated logs callout, job command, and task graph diagnostics link when parameterViewModel is populated', () => {
    fixture.componentRef.setInput('parameterViewModel', mockParameterViewModel);
    fixture.detectChanges();

    const progressBar = fixture.debugElement.query(
      By.directive(MatProgressBar),
    );
    expect(progressBar).toBeNull();

    const callout = fixture.debugElement.query(
      By.css('.total-estimated-callout'),
    );
    expect(callout).toBeTruthy();
    expect(callout.nativeElement.textContent).toContain(
      '~12,345 total logs estimated',
    );

    const queryCards = fixture.debugElement.queryAll(By.css('.query-card'));
    expect(queryCards.length).toBe(1);
    expect(queryCards[0].nativeElement.textContent).toContain(
      'Audit Log Query',
    );
    expect(queryCards[0].nativeElement.textContent).toContain(
      'resource.type="k8s_cluster"',
    );

    const jobCommand = fixture.debugElement.query(
      By.directive(JobCommandComponent),
    );
    expect(jobCommand).toBeTruthy();

    const taskGraphLink = fixture.debugElement.query(
      By.css('.task-graph-link-content a'),
    );
    expect(taskGraphLink).toBeTruthy();
    expect(taskGraphLink.nativeElement.getAttribute('href')).toContain(
      '/debug/task-graph?tab=DAG_VIEWER&inspectionType=gke&features=feat-1',
    );
    expect(taskGraphLink.nativeElement.textContent).toContain(
      'Open Task Graph Diagnostics',
    );
  });

  it('should show error count message when errorFieldCount > 0', () => {
    const errorViewModel: ParameterStepViewModel = {
      ...mockParameterViewModel,
      rootGroupForm: {
        ...mockParameterViewModel.rootGroupForm,
        children: [
          {
            id: 'err-field-1',
            label: 'Field 1',
            description: '',
            type: ParameterInputType.Text,
            hint: 'Error 1',
            hintType: ParameterHintType.Error,
            default: '',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Blur,
          } as TextParameterFormField,
          {
            id: 'err-field-2',
            label: 'Field 2',
            description: '',
            type: ParameterInputType.Text,
            hint: 'Error 2',
            hintType: ParameterHintType.Error,
            default: '',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Blur,
          } as TextParameterFormField,
        ],
      },
      fieldCount: 5,
    };
    fixture.componentRef.setInput('parameterViewModel', errorViewModel);
    fixture.detectChanges();

    const errorMessage = fixture.debugElement.query(
      By.css('.errmsg-parameters-error'),
    );
    expect(errorMessage).toBeTruthy();
    expect(errorMessage.nativeElement.textContent).toContain(
      'Fix the validation errors on the input parameters: 2 / 5',
    );

    const runButton = fixture.debugElement.query(By.css('.run-button'));
    expect(runButton.nativeElement.disabled).toBeTrue();
  });

  it('should show pending parameters message when pendingFieldCount > 0 and errorFieldCount is 0', () => {
    const pendingViewModel: ParameterStepViewModel = {
      ...mockParameterViewModel,
      rootGroupForm: {
        ...mockParameterViewModel.rootGroupForm,
        children: [
          {
            id: 'p-field-1',
            label: 'Field 1',
            description: '',
            type: ParameterInputType.Text,
            hint: '',
            hintType: ParameterHintType.None,
            pending: true,
            default: '',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Blur,
          } as TextParameterFormField,
          {
            id: 'p-field-2',
            label: 'Field 2',
            description: '',
            type: ParameterInputType.Text,
            hint: '',
            hintType: ParameterHintType.None,
            pending: true,
            default: '',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Blur,
          } as TextParameterFormField,
          {
            id: 'p-field-3',
            label: 'Field 3',
            description: '',
            type: ParameterInputType.Text,
            hint: '',
            hintType: ParameterHintType.None,
            pending: true,
            default: '',
            readonly: false,
            suggestions: [],
            validationTiming: ParameterFormValidationTiming.Blur,
          } as TextParameterFormField,
        ],
      },
      fieldCount: 5,
    };
    fixture.componentRef.setInput('parameterViewModel', pendingViewModel);
    fixture.detectChanges();

    const pendingSection = fixture.debugElement.query(
      By.css('.pending-parameters'),
    );
    expect(pendingSection).toBeTruthy();
    expect(pendingSection.nativeElement.textContent).toContain(
      'Resolving parameters: 3 / 5',
    );

    const runButton = fixture.debugElement.query(By.css('.run-button'));
    expect(runButton.nativeElement.disabled).toBeTrue();
  });

  it('should emit runInspection when Run button is clicked', () => {
    fixture.componentRef.setInput('parameterViewModel', mockParameterViewModel);
    fixture.detectChanges();

    let emitted = false;
    fixture.componentInstance.runInspection.subscribe(() => {
      emitted = true;
    });

    const runButton = fixture.debugElement.query(By.css('.run-button'));
    expect(runButton.nativeElement.disabled).toBeFalse();
    expect(fixture.componentInstance.hasRun()).toBeFalse();

    runButton.nativeElement.click();
    fixture.detectChanges();

    expect(emitted).toBeTrue();
    expect(fixture.componentInstance.hasRun()).toBeTrue();
    expect(runButton.nativeElement.disabled).toBeTrue();
  });
});
