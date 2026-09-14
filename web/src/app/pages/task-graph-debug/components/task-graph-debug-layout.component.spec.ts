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
  RegisteredTaskGroupInfo,
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';
import { TaskGraphDebugTab } from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { TaskGraphDebugLayoutComponent } from './task-graph-debug-layout.component';

describe('TaskGraphDebugLayoutComponent', () => {
  let component: TaskGraphDebugLayoutComponent;
  let fixture: ComponentFixture<TaskGraphDebugLayoutComponent>;

  const mockInspectionTypes: RegisteredInspectionTypeInfo[] = [
    {
      id: 'gke-standard',
      name: 'GKE Standard Cluster',
      description: 'Standard cluster',
      labels: {},
    } as unknown as RegisteredInspectionTypeInfo,
  ];

  const mockFeatures: FeatureToggleInfo[] = [
    {
      taskImplementationId: 'feature.network',
      label: 'Network',
      description: 'Network features',
      enabled: true,
    } as unknown as FeatureToggleInfo,
  ];

  const mockGroups: RegisteredTaskGroupInfo[] = [
    {
      taskReferenceId: 'parser.audit',
      tasks: [],
    } as unknown as RegisteredTaskGroupInfo,
  ];

  const mockEvaluations: TaskFilterEvaluation[] = [
    {
      taskImplementationId: 'parser.audit.v1',
      taskReferenceId: 'parser.audit',
      isCompatible: true,
      isSelected: true,
      matchReason: 'OK',
      supersededByTaskImplementationId: '',
      priority: 100,
    } as unknown as TaskFilterEvaluation,
  ];

  const mockDagNodes: DagViewerNode[] = [
    {
      id: 'parser.audit.v1',
      referenceId: 'parser.audit',
      priority: 100,
      isFeature: false,
      isFormTask: true,
      topologicalOrder: 0,
      labels: {},
      outputType: 'string',
      providedTags: [],
    },
  ];

  const mockDagEdges: DagViewerEdge[] = [];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TaskGraphDebugLayoutComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(TaskGraphDebugLayoutComponent);
    component = fixture.componentInstance;
  });

  it('renders default layout with registry tab', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.componentRef.setInput('selectedInspectionTypeId', 'gke-standard');
    fixture.componentRef.setInput('evaluations', mockEvaluations);
    fixture.componentRef.setInput('availableFeatures', mockFeatures);
    fixture.componentRef.setInput('dagNodes', mockDagNodes);
    fixture.componentRef.setInput('dagEdges', mockDagEdges);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.header-title')?.textContent).toContain(
      'Task Graph Diagnostics',
    );
    expect(compiled.querySelector('khi-step1-registry-table')).toBeTruthy();
  });

  it('emits tabChange when tab button is clicked', () => {
    const tabSpy = jasmine.createSpy('tabChange');
    component.tabChange.subscribe(tabSpy);

    component.onTabClick(TaskGraphDebugTab.DAG_VIEWER);
    expect(tabSpy).toHaveBeenCalledWith(TaskGraphDebugTab.DAG_VIEWER);

    component.onTabClick(TaskGraphDebugTab.INSPECTION_TYPE_FILTER);
    expect(tabSpy).toHaveBeenCalledWith(
      TaskGraphDebugTab.INSPECTION_TYPE_FILTER,
    );
  });

  it('emits selectInspectionType on selection', () => {
    const typeSpy = jasmine.createSpy('selectInspectionType');
    component.selectInspectionType.subscribe(typeSpy);

    component.onSelectInspectionType('gke-standard');
    expect(typeSpy).toHaveBeenCalledWith('gke-standard');
  });

  it('emits featureToggleChange on toggle', () => {
    const featureSpy = jasmine.createSpy('featureToggleChange');
    component.featureToggleChange.subscribe(featureSpy);

    component.onFeatureToggleChange({
      taskImplementationId: 'feature.network',
      enabled: false,
    });
    expect(featureSpy).toHaveBeenCalledWith({
      taskImplementationId: 'feature.network',
      enabled: false,
    });
  });

  it('renders step 2 filter when activeTab is INSPECTION_TYPE_FILTER', () => {
    fixture.componentRef.setInput(
      'activeTab',
      TaskGraphDebugTab.INSPECTION_TYPE_FILTER,
    );
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.componentRef.setInput('evaluations', mockEvaluations);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('khi-step2-type-filter')).toBeTruthy();
    expect(compiled.querySelector('khi-step1-registry-table')).toBeFalsy();
  });

  it('renders dag canvas when activeTab is DAG_VIEWER', () => {
    fixture.componentRef.setInput('activeTab', TaskGraphDebugTab.DAG_VIEWER);
    fixture.componentRef.setInput('dagNodes', mockDagNodes);
    fixture.componentRef.setInput('dagEdges', mockDagEdges);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('khi-dag-canvas')).toBeTruthy();
    expect(compiled.querySelector('khi-step1-registry-table')).toBeFalsy();
  });

  it('displays error banner in DAG tab when resolution fails', () => {
    fixture.componentRef.setInput('activeTab', TaskGraphDebugTab.DAG_VIEWER);
    fixture.componentRef.setInput('isResolutionSuccess', false);
    fixture.componentRef.setInput(
      'resolutionErrorMessage',
      'Cycle detected: task-a -> task-b -> task-a',
    );
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const banner = compiled.querySelector('.resolution-error-banner');
    expect(banner).toBeTruthy();
    expect(banner?.textContent).toContain('Task Graph Resolution Failed');
    expect(banner?.textContent).toContain('Cycle detected');
  });
});
