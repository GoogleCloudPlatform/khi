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
  RegisteredTaskGroupInfo,
  RegisteredTaskInfo,
  TaskDependencyCardinality,
  TaskDependencyScope,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { TASK_DESCRIPTION_LABEL_KEY } from 'src/app/shared/components/dag-viewer/dag-viewer.model';
import { Step1RegistryTableComponent } from './step1-registry-table.component';

describe('Step1RegistryTableComponent', () => {
  let component: Step1RegistryTableComponent;
  let fixture: ComponentFixture<Step1RegistryTableComponent>;

  const mockTasks: RegisteredTaskInfo[] = [
    {
      taskImplementationId: 'parser.k8s.audit-log',
      taskReferenceId: 'parser.k8s.audit',
      priority: 100,
      isFeature: false,
      isDefaultFeature: false,
      featureLabel: '',
      featureDescription: '',
      dependencies: [],
      providedTags: [],
      selectorRequirements: [],
      compatibleInspectionTypes: [],
      labels: {
        log_type: 'audit',
        'khi.google.com/inspection/is-form-task': 'true',
        [TASK_DESCRIPTION_LABEL_KEY]:
          'Parses Kubernetes audit logs into structured events.',
      },
    } as unknown as RegisteredTaskInfo,
    {
      taskImplementationId: 'parser.k8s.audit-log-specialized',
      taskReferenceId: 'parser.k8s.audit',
      priority: 200,
      isFeature: false,
      isDefaultFeature: false,
      featureLabel: '',
      featureDescription: '',
      dependencies: [],
      providedTags: [],
      selectorRequirements: [],
      compatibleInspectionTypes: [],
      labels: {
        log_type: 'audit',
        env: 'prod',
      },
    } as unknown as RegisteredTaskInfo,
  ];

  const mockGroups: RegisteredTaskGroupInfo[] = [
    {
      taskReferenceId: 'parser.k8s.audit',
      tasks: mockTasks,
    } as unknown as RegisteredTaskGroupInfo,
    {
      taskReferenceId: 'timeline.builder',
      tasks: [
        {
          taskImplementationId: 'timeline.builder.default',
          taskReferenceId: 'timeline.builder',
          priority: 50,
          isFeature: false,
          isDefaultFeature: false,
          featureLabel: '',
          featureDescription: '',
          dependencies: [
            {
              cardinality: TaskDependencyCardinality.FAN_IN,
              scope: TaskDependencyScope.ALL,
              targetReferenceId: '',
              targetTag: 'timeline-producer',
            },
          ],
          providedTags: [],
          labels: {},
        } as unknown as RegisteredTaskInfo,
      ],
    } as unknown as RegisteredTaskGroupInfo,
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Step1RegistryTableComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(Step1RegistryTableComponent);
    component = fixture.componentInstance;
  });

  it('renders all task groups initially', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    expect(component.filteredTaskGroups().length).toBe(2);
    expect(component.totalImplementationsCount()).toBe(3);
  });

  it('filters groups by search query matching reference ID', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    component.searchQuery.set('timeline');
    expect(component.filteredTaskGroups().length).toBe(1);
    expect(component.filteredTaskGroups()[0].taskReferenceId).toBe(
      'timeline.builder',
    );
  });

  it('filters groups by search query matching implementation ID', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    component.searchQuery.set('specialized');
    expect(component.filteredTaskGroups().length).toBe(1);
    expect(component.filteredTaskGroups()[0].taskReferenceId).toBe(
      'parser.k8s.audit',
    );
  });

  it('clears search query', () => {
    component.searchQuery.set('test');
    component.clearSearch();
    expect(component.searchQuery()).toBe('');
  });

  it('selects task and opens neighborhood panel on selectTask', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    expect(component.selectedTaskId()).toBeNull();
    expect(
      fixture.nativeElement.querySelector('aside.neighborhood-panel-container'),
    ).toBeNull();

    component.selectTask('parser.k8s.audit-log');
    fixture.detectChanges();

    expect(component.selectedTaskId()).toBe('parser.k8s.audit-log');
    expect(component.history()).toEqual(['parser.k8s.audit-log']);
    expect(component.historyIndex()).toBe(0);
    expect(
      fixture.nativeElement.querySelector('aside.neighborhood-panel-container'),
    ).toBeTruthy();

    const selectedCard = fixture.nativeElement.querySelector(
      '#task-impl-parser\\.k8s\\.audit-log',
    );
    expect(selectedCard?.classList.contains('selected')).toBeTrue();
  });

  it('navigates history back and forward correctly', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    component.selectTask('parser.k8s.audit-log');
    component.selectTask('timeline.builder.default');

    expect(component.canGoBack()).toBeTrue();
    expect(component.canGoForward()).toBeFalse();
    expect(component.selectedTaskId()).toBe('timeline.builder.default');

    component.goBack();
    expect(component.selectedTaskId()).toBe('parser.k8s.audit-log');
    expect(component.canGoBack()).toBeFalse();
    expect(component.canGoForward()).toBeTrue();

    component.goForward();
    expect(component.selectedTaskId()).toBe('timeline.builder.default');
    expect(component.canGoForward()).toBeFalse();
  });

  it('closes neighborhood panel on closePanel', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    component.selectTask('parser.k8s.audit-log');
    fixture.detectChanges();
    expect(component.selectedTaskId()).toBe('parser.k8s.audit-log');

    component.closePanel();
    fixture.detectChanges();

    expect(component.selectedTaskId()).toBeNull();
    expect(
      fixture.nativeElement.querySelector('aside.neighborhood-panel-container'),
    ).toBeNull();
  });

  it('truncates forward history when selecting a new task after navigating back', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    component.selectTask('parser.k8s.audit-log');
    component.selectTask('timeline.builder.default');
    component.selectTask('parser.k8s.audit-log.specialized');

    expect(component.history()).toEqual([
      'parser.k8s.audit-log',
      'timeline.builder.default',
      'parser.k8s.audit-log.specialized',
    ]);
    expect(component.historyIndex()).toBe(2);

    // Navigate back to the first entry
    component.goBack();
    component.goBack();
    expect(component.historyIndex()).toBe(0);
    expect(component.selectedTaskId()).toBe('parser.k8s.audit-log');
    expect(component.canGoForward()).toBeTrue();

    // Select a new task from the back position
    component.selectTask('timeline.builder.default');
    expect(component.history()).toEqual([
      'parser.k8s.audit-log',
      'timeline.builder.default',
    ]);
    expect(component.historyIndex()).toBe(1);
    expect(component.canGoForward()).toBeFalse();
  });

  it('does not duplicate history when re-selecting the same task after closing the panel', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    component.selectTask('parser.k8s.audit-log');
    expect(component.history()).toEqual(['parser.k8s.audit-log']);
    expect(component.historyIndex()).toBe(0);

    // Close panel
    component.closePanel();
    expect(component.selectedTaskId()).toBeNull();

    // Re-select the same task
    component.selectTask('parser.k8s.audit-log');
    expect(component.history()).toEqual(['parser.k8s.audit-log']);
    expect(component.historyIndex()).toBe(0);
    expect(component.selectedTaskId()).toBe('parser.k8s.audit-log');
  });

  it('renders Input badge when task is a form input task', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    const element: HTMLElement = fixture.nativeElement;
    const badge = element.querySelector('.badge-input');
    expect(badge).toBeTruthy();
    expect(badge?.textContent?.trim()).toBe('Input');
  });

  it('renders task description when description label is present', () => {
    fixture.componentRef.setInput('taskGroups', mockGroups);
    fixture.detectChanges();

    const element: HTMLElement = fixture.nativeElement;
    const descEl = element.querySelector('.task-description');
    expect(descEl).toBeTruthy();
    expect(descEl?.textContent?.trim()).toBe(
      'Parses Kubernetes audit logs into structured events.',
    );
  });
});
