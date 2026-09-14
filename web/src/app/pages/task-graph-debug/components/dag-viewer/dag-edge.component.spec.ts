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

import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TaskDependencyCardinality } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { DagEdgeComponent } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-edge.component';
import { DagPositionedEdge } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';

@Component({
  standalone: true,
  imports: [DagEdgeComponent],
  template: `
    <svg>
      <g
        khi-dag-edge
        [edge]="edge"
        [isHighlighted]="isHighlighted"
        [isDimmed]="isDimmed"
      ></g>
    </svg>
  `,
})
class TestHostComponent {
  edge: DagPositionedEdge = mockPointEdge;
  isHighlighted = false;
  isDimmed = false;
}

const mockPointEdge: DagPositionedEdge = {
  id: 'task-a->task-b',
  sourceId: 'task-a',
  destinationId: 'task-b',
  sourceReferenceId: 'ref-a',
  cardinality: TaskDependencyCardinality.POINT_TO_POINT,
  tag: '',
  priority: 100,
  outputType: 'string',
  pathD: 'M 100 50 C 150 50, 200 150, 250 150',
  startX: 100,
  startY: 50,
  endX: 250,
  endY: 150,
  labelX: 175,
  labelY: 100,
};

const mockFanInEdge: DagPositionedEdge = {
  ...mockPointEdge,
  id: 'task-c->task-d',
  cardinality: TaskDependencyCardinality.FAN_IN,
  tag: 'logs',
};

describe('DagEdgeComponent', () => {
  let hostComponent: TestHostComponent;
  let fixture: ComponentFixture<TestHostComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestHostComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    hostComponent = fixture.componentInstance;
  });

  it('renders point-to-point edge path with correct d attribute', () => {
    hostComponent.edge = mockPointEdge;
    fixture.detectChanges();

    const path = fixture.nativeElement.querySelector('.edge-path');
    expect(path).toBeTruthy();
    expect(path.getAttribute('d')).toBe(mockPointEdge.pathD);
    expect(path.getAttribute('marker-end')).toBe('url(#arrow-marker)');
    expect(fixture.nativeElement.querySelector('.edge-tag-group')).toBeNull();
  });

  it('renders single-line tag badge when tag has no domain prefix', () => {
    hostComponent.edge = mockFanInEdge;
    fixture.detectChanges();

    const group = fixture.nativeElement.querySelector('.dag-edge-group');
    expect(group.classList.contains('fan-in')).toBeTrue();

    const path = fixture.nativeElement.querySelector('.edge-path');
    expect(path.getAttribute('marker-end')).toBe('url(#arrow-marker-fan-in)');

    const tagText = fixture.nativeElement.querySelector('.tag-name-text');
    expect(tagText).toBeTruthy();
    expect(tagText.textContent.trim()).toBe('#logs');
    expect(fixture.nativeElement.querySelector('.tag-domain-text')).toBeNull();

    const tagRect = fixture.nativeElement.querySelector('.tag-badge-bg');
    expect(Number(tagRect.getAttribute('width'))).toBeGreaterThan(0);
    expect(tagRect.getAttribute('height')).toBe('22');
    expect(tagRect.getAttribute('rx')).toBe('11');
  });

  it('renders two-line tag badge when tag contains a domain prefix', () => {
    hostComponent.edge = {
      ...mockFanInEdge,
      tag: 'khi.google.com/timeline-producer',
    };
    fixture.detectChanges();

    const domainText = fixture.nativeElement.querySelector('.tag-domain-text');
    expect(domainText).toBeTruthy();
    expect(domainText.textContent.trim()).toBe('khi.google.com/');

    const nameText = fixture.nativeElement.querySelector('.tag-name-text');
    expect(nameText).toBeTruthy();
    expect(nameText.textContent.trim()).toBe('#timeline-producer');

    const tagRect = fixture.nativeElement.querySelector('.tag-badge-bg');
    expect(Number(tagRect.getAttribute('width'))).toBeGreaterThan(0);
    expect(tagRect.getAttribute('height')).toBe('34');
    expect(tagRect.getAttribute('rx')).toBe('6');
  });

  it('applies highlighted class and highlighted marker URL', () => {
    hostComponent.edge = mockPointEdge;
    hostComponent.isHighlighted = true;
    fixture.detectChanges();

    const group = fixture.nativeElement.querySelector('.dag-edge-group');
    expect(group.classList.contains('highlighted')).toBeTrue();

    const path = fixture.nativeElement.querySelector('.edge-path');
    expect(path.getAttribute('marker-end')).toBe(
      'url(#arrow-marker-highlighted)',
    );
  });

  it('applies dimmed class when isDimmed is true', () => {
    hostComponent.edge = mockPointEdge;
    hostComponent.isDimmed = true;
    fixture.detectChanges();

    const group = fixture.nativeElement.querySelector('.dag-edge-group');
    expect(group.classList.contains('dimmed')).toBeTrue();
  });
});
