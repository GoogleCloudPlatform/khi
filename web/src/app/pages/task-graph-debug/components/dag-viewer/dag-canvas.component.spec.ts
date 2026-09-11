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
import { TaskDependencyCardinality } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { DagCanvasComponent } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-canvas.component';
import {
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';

describe('DagCanvasComponent', () => {
  let component: DagCanvasComponent;
  let fixture: ComponentFixture<DagCanvasComponent>;

  const mockNodes: DagViewerNode[] = [
    {
      id: 'task-a',
      referenceId: 'ref-a',
      isFeature: false,
      isFormTask: true,
      topologicalOrder: 0,
      priority: 100,
      labels: {},
      outputType: 'string',
      providedTags: [],
    },
    {
      id: 'task-b',
      referenceId: 'ref-b',
      isFeature: false,
      isFormTask: false,
      topologicalOrder: 1,
      priority: 100,
      labels: {},
      outputType: 'number',
      providedTags: [],
    },
    {
      id: 'task-c',
      referenceId: 'ref-c',
      isFeature: true,
      isFormTask: false,
      topologicalOrder: 2,
      priority: 100,
      labels: {},
      outputType: 'boolean',
      providedTags: [],
    },
  ];

  const mockEdges: DagViewerEdge[] = [
    {
      id: 'task-a->task-b',
      sourceId: 'task-a',
      destinationId: 'task-b',
      sourceReferenceId: 'ref-a',
      cardinality: TaskDependencyCardinality.POINT_TO_POINT,
      tag: '',
      priority: 100,
      outputType: 'string',
    },
    {
      id: 'task-b->task-c',
      sourceId: 'task-b',
      destinationId: 'task-c',
      sourceReferenceId: 'ref-b',
      cardinality: TaskDependencyCardinality.POINT_TO_POINT,
      tag: '',
      priority: 100,
      outputType: 'number',
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DagCanvasComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(DagCanvasComponent);
    component = fixture.componentInstance;
  });

  it('renders canvas toolbar and SVG graph structure', () => {
    fixture.componentRef.setInput('nodes', mockNodes);
    fixture.componentRef.setInput('edges', mockEdges);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.dag-toolbar')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('.dag-svg')).toBeTruthy();

    const renderedNodes =
      fixture.nativeElement.querySelectorAll('g[khi-dag-node]');
    expect(renderedNodes.length).toBe(3);

    const renderedEdges =
      fixture.nativeElement.querySelectorAll('g[khi-dag-edge]');
    expect(renderedEdges.length).toBe(2);
  });

  it('updates zoom on zoomIn, zoomOut, and resetZoom', () => {
    component.zoom.set(1.0);

    component.zoomIn();
    expect(component.zoom()).toBeCloseTo(1.2, 2);

    component.zoomOut();
    expect(component.zoom()).toBeCloseTo(1.0, 2);

    component.zoom.set(1.8);
    component.panX.set(50);
    component.panY.set(50);
    component.resetZoom();
    expect(component.zoom()).toBe(1.0);
    expect(component.panX()).toBe(0);
    expect(component.panY()).toBe(0);
  });

  it('opens and closes detail panel when node is selected and deselected', () => {
    fixture.componentRef.setInput('nodes', mockNodes);
    fixture.componentRef.setInput('edges', mockEdges);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.dag-detail-dock')).toBeNull();

    // Select task-b
    component.onNodeSelected('task-b');
    fixture.detectChanges();

    expect(component.selectedNodeId()).toBe('task-b');
    expect(
      fixture.nativeElement.querySelector('.dag-detail-dock'),
    ).toBeTruthy();

    // Close panel
    component.closeDetailPanel();
    fixture.detectChanges();

    expect(component.selectedNodeId()).toBeNull();
    expect(fixture.nativeElement.querySelector('.dag-detail-dock')).toBeNull();
  });

  it('updates selectedNodeId without rendering detail panel when showDetailPanel is false', () => {
    fixture.componentRef.setInput('nodes', mockNodes);
    fixture.componentRef.setInput('edges', mockEdges);
    fixture.componentRef.setInput('showDetailPanel', false);
    fixture.detectChanges();

    component.onNodeSelected('task-b');
    fixture.detectChanges();

    expect(component.selectedNodeId()).toBe('task-b');
    expect(fixture.nativeElement.querySelector('.dag-detail-dock')).toBeNull();
  });

  it('computes upstream ancestors and downstream descendants for highlighting', () => {
    fixture.componentRef.setInput('nodes', mockNodes);
    fixture.componentRef.setInput('edges', mockEdges);
    fixture.detectChanges();

    // Select middle node B: upstream is A, downstream is C
    component.onNodeSelected('task-b');
    fixture.detectChanges();

    expect(component.upstreamAncestorIds().has('task-a')).toBeTrue();
    expect(component.downstreamDescendantIds().has('task-c')).toBeTrue();
    expect(component.isNodeHighlighted('task-a')).toBeTrue();
    expect(component.isNodeHighlighted('task-b')).toBeTrue();
    expect(component.isNodeHighlighted('task-c')).toBeTrue();
  });

  it('emits nodeClick when node is clicked', () => {
    fixture.componentRef.setInput('nodes', mockNodes);
    fixture.componentRef.setInput('edges', mockEdges);
    fixture.detectChanges();

    let clickedId = '';
    component.nodeClick.subscribe((id) => {
      clickedId = id;
    });

    component.onNodeSelected('task-a');
    expect(clickedId).toBe('task-a');
  });
});
