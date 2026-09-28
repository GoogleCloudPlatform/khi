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
import { ActivatedRoute, convertToParamMap } from '@angular/router';
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
import { TaskGraphDebugSmartComponent } from 'src/app/pages/task-graph-debug/task-graph-debug-smart.component';

describe('TaskGraphDebugSmartComponent', () => {
  let component: TaskGraphDebugSmartComponent;
  let fixture: ComponentFixture<TaskGraphDebugSmartComponent>;

  const mockInspectionTypes: RegisteredInspectionTypeInfo[] = [
    {
      id: 'type-a',
      name: 'Type A Cluster',
      description: 'Type A description',
      labels: {},
    } as unknown as RegisteredInspectionTypeInfo,
    {
      id: 'type-b',
      name: 'Type B Cluster',
      description: 'Type B description',
      labels: {},
    } as unknown as RegisteredInspectionTypeInfo,
  ];

  const mockGroups: RegisteredTaskGroupInfo[] = [
    {
      taskReferenceId: 'ref-group',
      tasks: [
        {
          taskImplementationId: 'feat-1',
          taskReferenceId: 'ref-group',
          isFeature: true,
        },
        {
          taskImplementationId: 'feat-2',
          taskReferenceId: 'ref-group',
          isFeature: true,
        },
        {
          taskImplementationId: 'non-feat',
          taskReferenceId: 'ref-group',
          isFeature: false,
        },
      ],
    } as unknown as RegisteredTaskGroupInfo,
  ];

  const mockRegistryResponse: GetInspectionTaskRegistryResponse = {
    taskGroups: mockGroups,
    inspectionTypes: mockInspectionTypes,
  } as unknown as GetInspectionTaskRegistryResponse;

  const mockEvaluations: TaskFilterEvaluation[] = [
    {
      taskImplementationId: 'feat-1',
      taskReferenceId: 'ref-group',
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
        taskImplementationId: 'feat-1',
        taskReferenceId: 'ref-group',
        isFeature: true,
        topologicalOrder: 0,
        priority: 100,
        labels: {},
      },
    ],
    edges: [
      {
        sourceImplementationId: 'feat-1',
        destinationImplementationId: 'feat-2',
        sourceReferenceId: 'ref-group',
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

  const createComponentWithQueryParams = async (
    queryParams: Record<string, string>,
  ): Promise<{
    fixture: ComponentFixture<TaskGraphDebugSmartComponent>;
    component: TaskGraphDebugSmartComponent;
  }> => {
    getRegistrySpy = jasmine
      .createSpy('getInspectionTaskRegistry')
      .and.resolveTo(mockRegistryResponse);
    resolveGraphSpy = jasmine
      .createSpy('resolveInspectionTaskGraph')
      .and.resolveTo(mockResolveResponse);

    const mockConnectClientService = {
      inspectionTaskGraphClient: {
        getInspectionTaskRegistry: getRegistrySpy,
        resolveInspectionTaskGraph: resolveGraphSpy,
      },
    };

    await TestBed.configureTestingModule({
      imports: [TaskGraphDebugSmartComponent, NoopAnimationsModule],
      providers: [
        {
          provide: ConnectClientService,
          useValue: mockConnectClientService,
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParamMap: convertToParamMap(queryParams),
            },
          },
        },
      ],
    }).compileComponents();

    const createdFixture = TestBed.createComponent(
      TaskGraphDebugSmartComponent,
    );
    return {
      fixture: createdFixture,
      component: createdFixture.componentInstance,
    };
  };

  describe('default initialization without query parameters', () => {
    beforeEach(async () => {
      const created = await createComponentWithQueryParams({});
      fixture = created.fixture;
      component = created.component;
    });

    it('defaults to REGISTRY tab, selects first inspection type, and resolves with empty feature overrides', async () => {
      expect(component.activeTab()).toBe(TaskGraphDebugTab.REGISTRY);

      fixture.detectChanges();
      await fixture.whenStable();

      expect(getRegistrySpy).toHaveBeenCalled();
      expect(component.selectedInspectionTypeId()).toBe('type-a');
      expect(component.featureOverrides()).toEqual({});
      expect(resolveGraphSpy).toHaveBeenCalledWith({
        inspectionTypeId: 'type-a',
        featureOverrides: {},
      });
      expect(component.taskGroups().length).toBe(1);
      expect(component.inspectionTypes().length).toBe(2);
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

      component.onSelectInspectionType('type-b');
      await fixture.whenStable();

      expect(component.selectedInspectionTypeId()).toBe('type-b');
      expect(resolveGraphSpy).toHaveBeenCalledWith({
        inspectionTypeId: 'type-b',
        featureOverrides: {},
      });
    });

    it('resolves graph with overrides when feature is toggled', async () => {
      fixture.detectChanges();
      await fixture.whenStable();
      resolveGraphSpy.calls.reset();

      component.onFeatureToggleChange({
        taskImplementationId: 'feat-1',
        enabled: false,
      });
      await fixture.whenStable();

      expect(resolveGraphSpy).toHaveBeenCalledWith({
        inspectionTypeId: 'type-a',
        featureOverrides: { 'feat-1': false },
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

      component.selectedInspectionTypeId.set('type-a');
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

      component.selectedInspectionTypeId.set('type-a');
      await component.resolveGraph();

      expect(component.isResolutionSuccess()).toBeFalse();
      expect(component.resolutionErrorMessage()).toContain(
        'Network connection refused',
      );
      expect(component.isLoading()).toBeFalse();
    });
  });

  describe('initialization with URL query parameters', () => {
    it('initializes with tab=DAG_VIEWER, inspectionType=type-b, and features=feat-2', async () => {
      const created = await createComponentWithQueryParams({
        tab: 'DAG_VIEWER',
        inspectionType: 'type-b',
        features: 'feat-2',
      });
      fixture = created.fixture;
      component = created.component;

      fixture.detectChanges();
      await fixture.whenStable();

      expect(component.activeTab()).toBe(TaskGraphDebugTab.DAG_VIEWER);

      expect(component.selectedInspectionTypeId()).toBe('type-b');
      expect(component.featureOverrides()).toEqual({
        'feat-1': false,
        'feat-2': true,
      });
      expect(resolveGraphSpy).toHaveBeenCalledWith({
        inspectionTypeId: 'type-b',
        featureOverrides: {
          'feat-1': false,
          'feat-2': true,
        },
      });
    });
  });
});
