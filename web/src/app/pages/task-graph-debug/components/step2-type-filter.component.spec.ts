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
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import {
  FeatureToggleInfo,
  RegisteredInspectionTypeInfo,
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { TaskFilterStatus } from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import {
  EvaluationStatusFilter,
  Step2TypeFilterComponent,
} from './step2-type-filter.component';

describe('Step2TypeFilterComponent', () => {
  let component: Step2TypeFilterComponent;
  let fixture: ComponentFixture<Step2TypeFilterComponent>;

  const mockInspectionTypes: RegisteredInspectionTypeInfo[] = [
    {
      id: 'gke-standard',
      name: 'GKE Standard Cluster',
      description: 'Standard cluster with user managed node pools',
      labels: {
        platform: 'gke',
      },
    } as unknown as RegisteredInspectionTypeInfo,
    {
      id: 'gke-autopilot',
      name: 'GKE Autopilot Cluster',
      description: 'Fully automated cluster',
      labels: {},
    } as unknown as RegisteredInspectionTypeInfo,
  ];

  const mockFeatures: FeatureToggleInfo[] = [
    {
      taskImplementationId: 'feature.network.flow',
      label: 'Network Flow Logging',
      description: 'Capture container network traffic metrics',
      enabled: true,
    } as unknown as FeatureToggleInfo,
  ];

  const mockEvaluations: TaskFilterEvaluation[] = [
    {
      taskImplementationId: 'parser.audit.v1',
      taskReferenceId: 'parser.audit',
      isCompatible: true,
      isSelected: true,
      matchReason: 'Matched platform=gke label',
      supersededByTaskImplementationId: '',
      priority: 100,
    } as unknown as TaskFilterEvaluation,
    {
      taskImplementationId: 'parser.audit.legacy',
      taskReferenceId: 'parser.audit',
      isCompatible: true,
      isSelected: false,
      matchReason: 'Lower priority than parser.audit.v1',
      supersededByTaskImplementationId: 'parser.audit.v1',
      priority: 50,
    } as unknown as TaskFilterEvaluation,
    {
      taskImplementationId: 'parser.aws.cloudtrail',
      taskReferenceId: 'parser.audit',
      isCompatible: false,
      isSelected: false,
      matchReason: 'Label platform=aws did not match requirement platform=gke',
      supersededByTaskImplementationId: '',
      priority: 100,
    } as unknown as TaskFilterEvaluation,
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Step2TypeFilterComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(Step2TypeFilterComponent);
    component = fixture.componentInstance;
  });

  it('classifies evaluations accurately', () => {
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.componentRef.setInput('selectedInspectionTypeId', 'gke-standard');
    fixture.componentRef.setInput('availableFeatures', mockFeatures);
    fixture.componentRef.setInput('evaluations', mockEvaluations);
    fixture.detectChanges();

    expect(component.selectedCount()).toBe(1);
    expect(component.supersededCount()).toBe(1);
    expect(component.incompatibleCount()).toBe(1);

    const items = component.classifiedEvaluations();
    expect(items[0].status).toBe(TaskFilterStatus.SELECTED);
    expect(items[1].status).toBe(TaskFilterStatus.SUPERSEDED);
    expect(items[2].status).toBe(TaskFilterStatus.INCOMPATIBLE);
  });

  it('filters evaluations by status filter', () => {
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.componentRef.setInput('selectedInspectionTypeId', 'gke-standard');
    fixture.componentRef.setInput('evaluations', mockEvaluations);
    fixture.detectChanges();

    component.setStatusFilter(EvaluationStatusFilter.SELECTED_ONLY);
    expect(component.filteredEvaluations().length).toBe(1);
    expect(component.filteredEvaluations()[0].status).toBe(
      TaskFilterStatus.SELECTED,
    );

    component.setStatusFilter(EvaluationStatusFilter.SUPERSEDED_ONLY);
    expect(component.filteredEvaluations().length).toBe(1);
    expect(component.filteredEvaluations()[0].status).toBe(
      TaskFilterStatus.SUPERSEDED,
    );

    component.setStatusFilter(EvaluationStatusFilter.INCOMPATIBLE_ONLY);
    expect(component.filteredEvaluations().length).toBe(1);
    expect(component.filteredEvaluations()[0].status).toBe(
      TaskFilterStatus.INCOMPATIBLE,
    );
  });

  it('filters evaluations by search query', () => {
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.componentRef.setInput('evaluations', mockEvaluations);
    fixture.detectChanges();

    component.searchQuery.set('cloudtrail');
    expect(component.filteredEvaluations().length).toBe(1);
    expect(
      component.filteredEvaluations()[0].evaluation.taskImplementationId,
    ).toBe('parser.aws.cloudtrail');
  });

  it('emits selectInspectionType when type is changed', () => {
    const spy = jasmine.createSpy('selectInspectionType');
    component.selectInspectionType.subscribe(spy);

    component.onInspectionTypeSelect('gke-autopilot');
    expect(spy).toHaveBeenCalledWith('gke-autopilot');
  });

  it('emits featureToggleChange when feature checkbox is toggled', () => {
    const spy = jasmine.createSpy('featureToggleChange');
    component.featureToggleChange.subscribe(spy);

    component.onFeatureToggle('feature.network.flow', false);
    expect(spy).toHaveBeenCalledWith({
      taskImplementationId: 'feature.network.flow',
      enabled: false,
    });
  });
});
