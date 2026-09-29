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
import { MatStepper } from '@angular/material/stepper';
import {
  InspectionFeature,
  InspectionType,
} from 'src/app/common/schema/api-types';
import {
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import {
  NewInspectionStepIndex,
  ParameterStepViewModel,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import { FeatureSelectorComponent } from 'src/app/dialogs/new-inspection/components/feature-selector.component';
import { InspectionTypeSelectorComponent } from 'src/app/dialogs/new-inspection/components/inspection-type-selector.component';
import { ParameterInputStepComponent } from 'src/app/dialogs/new-inspection/components/parameter-input-step.component';
import { NewInspectionLayoutComponent } from 'src/app/dialogs/new-inspection/components/new-inspection-layout.component';

describe('NewInspectionLayoutComponent', () => {
  let fixture: ComponentFixture<NewInspectionLayoutComponent>;

  const mockInspectionTypes: readonly InspectionType[] = [
    {
      id: 'gke',
      name: 'Google Kubernetes Engine',
      description: 'Inspect GKE cluster',
      icon: 'assets/gke.svg',
    },
  ];

  const mockFeatures: readonly InspectionFeature[] = [
    {
      id: 'feature-audit',
      label: 'Kubernetes Audit Log',
      description: 'Ingest Kubernetes API audit logs.',
      enabled: true,
    },
  ];

  const mockParameterViewModel: ParameterStepViewModel = {
    rootGroupForm: {
      id: 'root',
      label: 'Root Group',
      type: ParameterInputType.Group,
      description: '',
      hint: '',
      hintType: ParameterHintType.None,
      collapsible: false,
      collapsedByDefault: false,
      children: [],
    },
    queries: [],
    fieldCount: 1,
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule, NewInspectionLayoutComponent],
      providers: [
        {
          provide: PARAMETER_STORE,
          useClass: DefaultParameterStore,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(NewInspectionLayoutComponent);
  });

  function setStandardInputs(
    stepIndex: NewInspectionStepIndex = NewInspectionStepIndex.InspectionType,
  ): void {
    fixture.componentRef.setInput('selectedStepIndex', stepIndex);
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.componentRef.setInput(
      'currentInspectionType',
      mockInspectionTypes[0],
    );
    fixture.componentRef.setInput('features', mockFeatures);
    fixture.componentRef.setInput(
      'parameterStore',
      TestBed.inject(PARAMETER_STORE),
    );
    fixture.componentRef.setInput('parameterViewModel', mockParameterViewModel);
  }

  it('should render dialog title and stepper', () => {
    setStandardInputs();
    fixture.detectChanges();

    const title = fixture.debugElement.query(By.css('h2'));
    expect(title).toBeTruthy();
    expect(title.nativeElement.textContent).toContain('New inspection');

    const stepper = fixture.debugElement.query(By.directive(MatStepper));
    expect(stepper).toBeTruthy();
  });

  it('should propagate selectInspectionType event from InspectionTypeSelectorComponent', () => {
    setStandardInputs();
    fixture.detectChanges();

    let selectedType: InspectionType | undefined;
    fixture.componentInstance.selectInspectionType.subscribe(
      (type: InspectionType) => {
        selectedType = type;
      },
    );

    const typeSelector = fixture.debugElement.query(
      By.directive(InspectionTypeSelectorComponent),
    );
    expect(typeSelector).toBeTruthy();

    typeSelector.componentInstance.selectInspectionType.emit(
      mockInspectionTypes[0],
    );
    expect(selectedType).toEqual(mockInspectionTypes[0]);
  });

  it('should propagate toggleFeature event from FeatureSelectorComponent', () => {
    setStandardInputs();
    fixture.detectChanges();

    let toggledFeature: string | undefined;
    fixture.componentInstance.toggleFeature.subscribe((featureId: string) => {
      toggledFeature = featureId;
    });

    const featureSelector = fixture.debugElement.query(
      By.directive(FeatureSelectorComponent),
    );
    expect(featureSelector).toBeTruthy();

    featureSelector.componentInstance.toggleFeature.emit('feature-audit');
    expect(toggledFeature).toBe('feature-audit');
  });

  it('should emit stepChange with ParameterInput when FeatureSelectorComponent emits nextStep', () => {
    setStandardInputs();
    fixture.detectChanges();

    let changedStep: NewInspectionStepIndex | undefined;
    fixture.componentInstance.stepChange.subscribe(
      (step: NewInspectionStepIndex) => {
        changedStep = step;
      },
    );

    const featureSelector = fixture.debugElement.query(
      By.directive(FeatureSelectorComponent),
    );
    featureSelector.componentInstance.nextStep.emit();

    expect(changedStep).toBe(NewInspectionStepIndex.ParameterInput);
  });

  it('should propagate runInspection event from ParameterInputStepComponent', () => {
    setStandardInputs();
    fixture.detectChanges();

    let runEmitted = false;
    fixture.componentInstance.runInspection.subscribe(() => {
      runEmitted = true;
    });

    const parameterStep = fixture.debugElement.query(
      By.directive(ParameterInputStepComponent),
    );
    expect(parameterStep).toBeTruthy();

    parameterStep.componentInstance.runInspection.emit();
    expect(runEmitted).toBeTrue();
  });

  it('should propagate stepChange when stepper emits selectedIndexChange', () => {
    setStandardInputs();
    fixture.detectChanges();

    let changedStep: NewInspectionStepIndex | undefined;
    fixture.componentInstance.stepChange.subscribe(
      (step: NewInspectionStepIndex) => {
        changedStep = step;
      },
    );

    const stepper = fixture.debugElement.query(By.directive(MatStepper));
    stepper.componentInstance.selectedIndexChange.emit(
      NewInspectionStepIndex.FeatureSelection,
    );

    expect(changedStep).toBe(NewInspectionStepIndex.FeatureSelection);
  });

  it('should compute taskGraphDebugUrl and pass it to ParameterInputStepComponent', () => {
    setStandardInputs();
    fixture.detectChanges();

    const url = fixture.componentInstance.taskGraphDebugUrl();
    expect(url).toContain('tab=DAG_VIEWER');
    expect(url).toContain('inspectionType=gke');
    expect(url).toContain('features=feature-audit');

    const parameterStep = fixture.debugElement.query(
      By.directive(ParameterInputStepComponent),
    );
    expect(parameterStep).toBeTruthy();
    expect(parameterStep.componentInstance.taskGraphDebugUrl()).toBe(url);
  });
});
