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
  GetInspectionTaskRegistryResponse,
  RegisteredInspectionTypeInfo,
  RegisteredTaskGroupInfo,
  ResolveInspectionTaskGraphResponse,
  TaskDAGInfo,
  TaskDependencyCardinality,
  TaskFilterEvaluation,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { TaskGraphDebugTab } from 'src/app/pages/task-graph-debug/types/task-graph-debug.model';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { TaskGraphDebugSmartComponent } from './task-graph-debug-smart.component';

describe('TaskGraphDebugSmartComponent', () => {
  let component: TaskGraphDebugSmartComponent;
  let fixture: ComponentFixture<TaskGraphDebugSmartComponent>;

  const mockInspectionTypes: RegisteredInspectionTypeInfo[] = [
    {
      id: 'gke-standard',
      name: 'GKE Standard Cluster',
      description: 'Standard cluster',
      labels: {},
    } as unknown as RegisteredInspectionTypeInfo,
  ];

  const mockGroups: RegisteredTaskGroupInfo[] = [
    {
      taskReferenceId: 'parser.audit',
      tasks: [],
    } as unknown as RegisteredTaskGroupInfo,
  ];

  const mockRegistryResponse: GetInspectionTaskRegistryResponse = {
    taskGroups: mockGroups,
    inspectionTypes: mockInspectionTypes,
  } as unknown as GetInspectionTaskRegistryResponse;

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

  const mockDag: TaskDAGInfo = {
    isSuccess: true,
    errorMessage: '',
    nodes: [
      {
        taskImplementationId: 'parser.audit.v1',
        taskReferenceId: 'parser.audit',
        isFeature: false,
        topologicalOrder: 0,
        priority: 100,
        labels: {},
      },
    ],
    edges: [
      {
        sourceImplementationId: 'parser.audit.v1',
        destinationImplementationId: 'worker.v1',
        sourceReferenceId: 'parser.audit',
        cardinality: TaskDependencyCardinality.POINT_TO_POINT,
        tag: '',
        priority: 0,
      },
    ],
  } as unknown as TaskDAGInfo;

  const mockResolveResponse: ResolveInspectionTaskGraphResponse = {
    filteringEvaluations: mockEvaluations,
    dag: mockDag,
    availableFeatures: [],
  } as unknown as ResolveInspectionTaskGraphResponse;

  let getRegistrySpy: jasmine.Spy;
  let resolveGraphSpy: jasmine.Spy;

  beforeEach(async () => {
    getRegistrySpy = jasmine
      .createSpy('getInspectionTaskRegistry')
      .and.resolveTo(mockRegistryResponse);
    resolveGraphSpy = jasmine
      .createSpy('resolveInspectionTaskGraph')
      .and.resolveTo(mockResolveResponse);

    const mockInspectionTaskGraphClient = {
      getInspectionTaskRegistry: getRegistrySpy,
      resolveInspectionTaskGraph: resolveGraphSpy,
    };

    const mockConnectClientService = {
      inspectionTaskGraphClient: mockInspectionTaskGraphClient,
    };

    await TestBed.configureTestingModule({
      imports: [TaskGraphDebugSmartComponent, NoopAnimationsModule],
      providers: [
        {
          provide: ConnectClientService,
          useValue: mockConnectClientService,
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(TaskGraphDebugSmartComponent);
    component = fixture.componentInstance;
  });

  it('loads registry and resolves initial graph on initialization', async () => {
    fixture.detectChanges();
    await fixture.whenStable();

    expect(getRegistrySpy).toHaveBeenCalled();
    expect(resolveGraphSpy).toHaveBeenCalledWith({
      inspectionTypeId: 'gke-standard',
      featureOverrides: {},
    });

    expect(component.taskGroups().length).toBe(1);
    expect(component.inspectionTypes().length).toBe(1);
    expect(component.selectedInspectionTypeId()).toBe('gke-standard');
    expect(component.dagNodes().length).toBe(1);
    expect(component.dagEdges().length).toBe(1);
  });

  it('switches active tab', () => {
    component.onTabChange(TaskGraphDebugTab.INSPECTION_TYPE_FILTER);
    expect(component.activeTab()).toBe(
      TaskGraphDebugTab.INSPECTION_TYPE_FILTER,
    );

    component.onTabChange(TaskGraphDebugTab.DAG_VIEWER);
    expect(component.activeTab()).toBe(TaskGraphDebugTab.DAG_VIEWER);
  });

  it('resolves graph when inspection type changes', async () => {
    fixture.detectChanges();
    await fixture.whenStable();
    resolveGraphSpy.calls.reset();

    component.onSelectInspectionType('gke-autopilot');
    await fixture.whenStable();

    expect(component.selectedInspectionTypeId()).toBe('gke-autopilot');
    expect(resolveGraphSpy).toHaveBeenCalledWith({
      inspectionTypeId: 'gke-autopilot',
      featureOverrides: {},
    });
  });

  it('resolves graph with overrides when feature is toggled', async () => {
    fixture.detectChanges();
    await fixture.whenStable();
    resolveGraphSpy.calls.reset();

    component.onFeatureToggleChange({
      taskImplementationId: 'feature.network',
      enabled: false,
    });
    await fixture.whenStable();

    expect(resolveGraphSpy).toHaveBeenCalledWith({
      inspectionTypeId: 'gke-standard',
      featureOverrides: { 'feature.network': false },
    });
  });

  it('handles DAG resolution cycle failure gracefully', async () => {
    resolveGraphSpy.and.returnValue(
      Promise.resolve({
        filteringEvaluations: mockEvaluations,
        availableFeatures: [],
        dag: {
          isSuccess: false,
          errorMessage: 'Cycle detected between task-a and task-b',
          nodes: [],
          edges: [],
        },
      } as unknown as ResolveInspectionTaskGraphResponse),
    );

    component.selectedInspectionTypeId.set('gke-standard');
    await component.resolveGraph();

    expect(component.isResolutionSuccess()).toBeFalse();
    expect(component.resolutionErrorMessage()).toBe(
      'Cycle detected between task-a and task-b',
    );
  });

  it('handles RPC rejection gracefully during graph resolution', async () => {
    resolveGraphSpy.and.returnValue(
      Promise.reject(new Error('Network connection refused')),
    );

    component.selectedInspectionTypeId.set('gke-standard');
    await component.resolveGraph();

    expect(component.isResolutionSuccess()).toBeFalse();
    expect(component.resolutionErrorMessage()).toContain(
      'Network connection refused',
    );
    expect(component.isLoading()).toBeFalse();
  });
});
