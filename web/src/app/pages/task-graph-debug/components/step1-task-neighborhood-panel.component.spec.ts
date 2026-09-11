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
import {
  RegisteredTaskGroupInfo,
  RegisteredTaskInfo,
  TaskDependencyCardinality,
  TaskDependencyScope,
} from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  Step1TaskNeighborhoodPanelComponent,
  computeTaskNeighborhoodGraph,
} from 'src/app/pages/task-graph-debug/components/step1-task-neighborhood-panel.component';
import { TASK_DESCRIPTION_LABEL_KEY } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';

describe('Step1TaskNeighborhoodPanelComponent', () => {
  const mockTaskGroups: readonly RegisteredTaskGroupInfo[] = [
    {
      $typeName: 'api.v1.RegisteredTaskGroupInfo',
      taskReferenceId: 'ref-entry',
      tasks: [
        {
          $typeName: 'api.v1.RegisteredTaskInfo',
          taskImplementationId: 'impl-entry-1',
          taskReferenceId: 'ref-entry',
          priority: 100,
          isFeature: false,
          isDefaultFeature: false,
          featureDescription: '',
          outputType: 'test/RawLog',
          dependencies: [],
          providedTags: [
            {
              $typeName: 'api.v1.ProvidedTagInfo',
              tag: 'raw-logs',
              priority: 100,
              outputType: 'test/RawLog',
            },
          ],
          labels: {},
        } as unknown as RegisteredTaskInfo,
      ],
    } as unknown as RegisteredTaskGroupInfo,
    {
      $typeName: 'api.v1.RegisteredTaskGroupInfo',
      taskReferenceId: 'ref-parser',
      tasks: [
        {
          $typeName: 'api.v1.RegisteredTaskInfo',
          taskImplementationId: 'impl-parser-primary',
          taskReferenceId: 'ref-parser',
          priority: 100,
          isFeature: false,
          isDefaultFeature: false,
          featureDescription: '',
          outputType: 'test/ParsedEvent',
          dependencies: [
            {
              $typeName: 'api.v1.TaskDependencyInfo',
              cardinality: TaskDependencyCardinality.POINT_TO_POINT,
              scope: TaskDependencyScope.ACTIVE_GRAPH,
              targetReferenceId: 'ref-entry',
              targetTag: '',
            },
          ],
          providedTags: [
            {
              $typeName: 'api.v1.ProvidedTagInfo',
              tag: 'parsed-events',
              priority: 100,
              outputType: 'test/ParsedEvent',
            },
          ],
          labels: {
            [TASK_DESCRIPTION_LABEL_KEY]: 'Parses logs into events',
          },
        } as unknown as RegisteredTaskInfo,
        {
          $typeName: 'api.v1.RegisteredTaskInfo',
          taskImplementationId: 'impl-parser-alternative',
          taskReferenceId: 'ref-parser',
          priority: 50,
          isFeature: true,
          isDefaultFeature: false,
          featureDescription: 'Alternative experimental parser',
          outputType: 'test/ParsedEvent',
          dependencies: [
            {
              $typeName: 'api.v1.TaskDependencyInfo',
              cardinality: TaskDependencyCardinality.FAN_IN,
              scope: TaskDependencyScope.ALL,
              targetReferenceId: '',
              targetTag: 'raw-logs',
            },
          ],
          providedTags: [
            {
              $typeName: 'api.v1.ProvidedTagInfo',
              tag: 'parsed-events',
              priority: 50,
              outputType: 'test/ParsedEvent',
            },
          ],
          labels: {},
        } as unknown as RegisteredTaskInfo,
      ],
    } as unknown as RegisteredTaskGroupInfo,
    {
      $typeName: 'api.v1.RegisteredTaskGroupInfo',
      taskReferenceId: 'ref-aggregator',
      tasks: [
        {
          $typeName: 'api.v1.RegisteredTaskInfo',
          taskImplementationId: 'impl-aggregator',
          taskReferenceId: 'ref-aggregator',
          priority: 100,
          isFeature: false,
          isDefaultFeature: false,
          featureDescription: '',
          outputType: 'test/AggregatedResult',
          dependencies: [
            {
              $typeName: 'api.v1.TaskDependencyInfo',
              cardinality: TaskDependencyCardinality.FAN_IN,
              scope: TaskDependencyScope.ACTIVE_GRAPH,
              targetReferenceId: '',
              targetTag: 'parsed-events',
            },
            {
              $typeName: 'api.v1.TaskDependencyInfo',
              cardinality: TaskDependencyCardinality.POINT_TO_POINT,
              scope: TaskDependencyScope.ACTIVE_GRAPH,
              targetReferenceId: 'ref-parser',
              targetTag: '',
            },
          ],
          labels: {},
        } as unknown as RegisteredTaskInfo,
      ],
    } as unknown as RegisteredTaskGroupInfo,
    {
      $typeName: 'api.v1.RegisteredTaskGroupInfo',
      taskReferenceId: 'ref-sink',
      tasks: [
        {
          $typeName: 'api.v1.RegisteredTaskInfo',
          taskImplementationId: 'impl-sink',
          taskReferenceId: 'ref-sink',
          priority: 100,
          isFeature: false,
          isDefaultFeature: false,
          featureDescription: '',
          outputType: '',
          dependencies: [],
          providedTags: [],
          labels: {},
        } as unknown as RegisteredTaskInfo,
      ],
    } as unknown as RegisteredTaskGroupInfo,
  ];

  describe('computeTaskNeighborhoodGraph', () => {
    it('returns empty structure when centerTaskId is null or non-existent', () => {
      const nullResult = computeTaskNeighborhoodGraph(mockTaskGroups, null);
      expect(nullResult.centerTask).toBeNull();
      expect(nullResult.nodes.length).toBe(0);
      expect(nullResult.edges.length).toBe(0);

      const notFoundResult = computeTaskNeighborhoodGraph(
        mockTaskGroups,
        'non-existent-task',
      );
      expect(notFoundResult.centerTask).toBeNull();
      expect(notFoundResult.nodes.length).toBe(0);
      expect(notFoundResult.edges.length).toBe(0);
    });

    it('resolves 3 layers for a middle focal task with both parents and children', () => {
      // Choose impl-parser-primary:
      // Parent: impl-entry-1 (P2P on ref-entry)
      // Children: impl-aggregator (both Fan-in on parsed-events and P2P on ref-parser)
      const result = computeTaskNeighborhoodGraph(
        mockTaskGroups,
        'impl-parser-primary',
      );

      expect(result.centerTask?.taskImplementationId).toBe(
        'impl-parser-primary',
      );
      expect(result.parentCount).toBe(1);
      expect(result.childCount).toBe(1);

      const nodeIds = result.nodes.map((n) => n.id);
      expect(nodeIds).toContain('impl-entry-1');
      expect(nodeIds).toContain('impl-parser-primary');
      expect(nodeIds).toContain('impl-aggregator');

      // Check layer assignments (topologicalOrder)
      const parentNode = result.nodes.find((n) => n.id === 'impl-entry-1');
      const centerNode = result.nodes.find(
        (n) => n.id === 'impl-parser-primary',
      );
      const childNode = result.nodes.find((n) => n.id === 'impl-aggregator');

      expect(parentNode?.topologicalOrder).toBe(0);
      expect(centerNode?.topologicalOrder).toBe(1);
      expect(childNode?.topologicalOrder).toBe(2);

      // Check edges connecting parent to center and center to child
      const parentEdge = result.edges.find(
        (e) =>
          e.sourceId === 'impl-entry-1' &&
          e.destinationId === 'impl-parser-primary',
      );
      expect(parentEdge).toBeTruthy();
      expect(parentEdge?.cardinality).toBe(
        TaskDependencyCardinality.POINT_TO_POINT,
      );
      expect(parentEdge?.outputType).toBe('test/RawLog');

      const childP2PEdge = result.edges.find(
        (e) =>
          e.sourceId === 'impl-parser-primary' &&
          e.destinationId === 'impl-aggregator' &&
          e.cardinality === TaskDependencyCardinality.POINT_TO_POINT,
      );
      expect(childP2PEdge).toBeTruthy();
      expect(childP2PEdge?.outputType).toBe('test/ParsedEvent');

      const childFanInEdge = result.edges.find(
        (e) =>
          e.sourceId === 'impl-parser-primary' &&
          e.destinationId === 'impl-aggregator' &&
          e.cardinality === TaskDependencyCardinality.FAN_IN,
      );
      expect(childFanInEdge).toBeTruthy();
      expect(childFanInEdge?.tag).toBe('parsed-events');
      expect(childFanInEdge?.outputType).toBe('test/ParsedEvent');
    });

    it('resolves fan-in parent candidates correctly based on providedTags', () => {
      // impl-parser-alternative has Fan-in dependency on tag 'raw-logs'
      // impl-entry-1 provides tag 'raw-logs'
      const result = computeTaskNeighborhoodGraph(
        mockTaskGroups,
        'impl-parser-alternative',
      );

      expect(result.parentCount).toBe(1);
      const parentEdge = result.edges.find(
        (e) =>
          e.sourceId === 'impl-entry-1' &&
          e.destinationId === 'impl-parser-alternative',
      );
      expect(parentEdge).toBeTruthy();
      expect(parentEdge?.cardinality).toBe(TaskDependencyCardinality.FAN_IN);
      expect(parentEdge?.tag).toBe('raw-logs');
    });

    it('includes all implementations under the same target reference ID as parents', () => {
      // impl-aggregator depends on ref-parser (P2P).
      // Both impl-parser-primary and impl-parser-alternative share ref-parser.
      const result = computeTaskNeighborhoodGraph(
        mockTaskGroups,
        'impl-aggregator',
      );

      const parentIds = result.edges
        .filter((e) => e.destinationId === 'impl-aggregator')
        .map((e) => e.sourceId);

      expect(parentIds).toContain('impl-parser-primary');
      expect(parentIds).toContain('impl-parser-alternative');
      expect(result.childCount).toBe(0);
    });
  });

  describe('Component interaction and rendering', () => {
    let component: Step1TaskNeighborhoodPanelComponent;
    let fixture: ComponentFixture<Step1TaskNeighborhoodPanelComponent>;

    beforeEach(async () => {
      await TestBed.configureTestingModule({
        imports: [Step1TaskNeighborhoodPanelComponent],
      }).compileComponents();

      fixture = TestBed.createComponent(Step1TaskNeighborhoodPanelComponent);
      component = fixture.componentInstance;
    });

    it('renders header, identity badges, and summary counts for selected center task', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-parser-primary');
      fixture.detectChanges();

      const titleEl = fixture.nativeElement.querySelector('.panel-title');
      expect(titleEl.textContent).toContain('Neighborhood');

      const refPill = fixture.nativeElement.querySelector('.ref-pill');
      expect(refPill.textContent).toBe('ref-parser');

      const implPill = fixture.nativeElement.querySelector('.impl-pill');
      expect(implPill.textContent).toBe('impl-parser-primary');

      const upstreamCount = fixture.nativeElement.querySelector(
        '.summary-item.upstream .summary-count',
      );
      expect(upstreamCount.textContent.trim()).toBe('1');

      const downstreamCount = fixture.nativeElement.querySelector(
        '.summary-item.downstream .summary-count',
      );
      expect(downstreamCount.textContent.trim()).toBe('1');
    });

    it('disables back and forward navigation buttons when cannot navigate', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-parser-primary');
      fixture.componentRef.setInput('canGoBack', false);
      fixture.componentRef.setInput('canGoForward', false);
      fixture.detectChanges();

      const navButtons = fixture.nativeElement.querySelectorAll(
        '.history-nav button',
      );
      expect(navButtons[0].disabled).toBeTrue();
      expect(navButtons[1].disabled).toBeTrue();
    });

    it('enables navigation buttons and emits events when clicked', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-parser-primary');
      fixture.componentRef.setInput('canGoBack', true);
      fixture.componentRef.setInput('canGoForward', true);
      fixture.detectChanges();

      let backEmitted = false;
      let forwardEmitted = false;
      let closeEmitted = false;

      component.goBack.subscribe(() => {
        backEmitted = true;
      });
      component.goForward.subscribe(() => {
        forwardEmitted = true;
      });
      component.closePanel.subscribe(() => {
        closeEmitted = true;
      });

      const navButtons = fixture.nativeElement.querySelectorAll(
        '.history-nav button',
      );
      expect(navButtons[0].disabled).toBeFalse();
      expect(navButtons[1].disabled).toBeFalse();

      navButtons[0].click();
      expect(backEmitted).toBeTrue();

      navButtons[1].click();
      expect(forwardEmitted).toBeTrue();

      const closeButton = fixture.nativeElement.querySelector('.close-btn');
      closeButton.click();
      expect(closeEmitted).toBeTrue();
    });

    it('emits selectTask when neighbor node is clicked', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-parser-primary');
      fixture.detectChanges();

      let selectedId: string | null = null;
      component.selectTask.subscribe((id) => {
        selectedId = id;
      });

      // Clicking non-center node emits selectTask
      component.onNodeClick('impl-entry-1');
      expect(selectedId as string | null).toBe('impl-entry-1');

      // Clicking center node does NOT emit selectTask
      selectedId = null;
      component.onNodeClick('impl-parser-primary');
      expect(selectedId as string | null).toBeNull();
    });

    it('renders task description when center task has description label', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-parser-primary');
      fixture.detectChanges();

      const descEl: HTMLElement | null = fixture.nativeElement.querySelector(
        '.task-description-row',
      );
      expect(descEl).toBeTruthy();
      expect(descEl?.textContent?.trim()).toBe('Parses logs into events');
    });

    it('does not render task description row when center task lacks description label', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-entry-1');
      fixture.detectChanges();

      const descEl = fixture.nativeElement.querySelector(
        '.task-description-row',
      );
      expect(descEl).toBeNull();
    });

    it('renders output type pill when center task has outputType', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-parser-primary');
      fixture.detectChanges();

      const outputTypeEl: HTMLElement | null =
        fixture.nativeElement.querySelector('.output-type-pill');
      expect(outputTypeEl).toBeTruthy();
      expect(outputTypeEl?.textContent?.trim()).toBe(
        'Output: test/ParsedEvent',
      );
      expect(outputTypeEl?.getAttribute('title')).toBe(
        'Output: test/ParsedEvent',
      );
    });

    it('does not render output type pill when center task lacks outputType', () => {
      fixture.componentRef.setInput('taskGroups', mockTaskGroups);
      fixture.componentRef.setInput('centerTaskId', 'impl-sink');
      fixture.detectChanges();

      const outputTypeEl =
        fixture.nativeElement.querySelector('.output-type-pill');
      expect(outputTypeEl).toBeNull();
    });
  });
});
