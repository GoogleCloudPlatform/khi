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
import { InspectionRunTaskGraphLayoutComponent } from 'src/app/dialogs/inspection-run-task-graph/components/inspection-run-task-graph-layout.component';
import { InspectionRunTaskGraphViewModel } from 'src/app/dialogs/inspection-run-task-graph/types/inspection-run-task-graph.viewmodel';
import { TaskDependencyCardinality } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagNodeRunPhase,
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

const sampleNodes: DagViewerNode[] = [
  {
    id: 'task-a#default',
    referenceId: 'task-a',
    isFeature: false,
    isFormTask: false,
    topologicalOrder: 0,
    priority: 0,
    labels: {},
    outputType: 'string',
    providedTags: [],
    runPhase: DagNodeRunPhase.DONE,
    runDurationMs: 1200,
  },
  {
    id: 'task-b#default',
    referenceId: 'task-b',
    isFeature: false,
    isFormTask: false,
    topologicalOrder: 1,
    priority: 0,
    labels: {},
    outputType: 'string',
    providedTags: [],
    runPhase: DagNodeRunPhase.RUNNING,
    runDurationMs: 300,
  },
];

const sampleEdges: DagViewerEdge[] = [
  {
    id: 'edge-0',
    sourceId: 'task-a#default',
    destinationId: 'task-b#default',
    sourceReferenceId: 'task-a',
    cardinality: TaskDependencyCardinality.POINT_TO_POINT,
    tag: '',
    priority: 0,
    outputType: 'string',
  },
];

function createViewModel(
  overrides: Partial<InspectionRunTaskGraphViewModel> = {},
): InspectionRunTaskGraphViewModel {
  return {
    inspectionName: 'sample inspection',
    nodes: sampleNodes,
    edges: sampleEdges,
    finishedTaskCount: 1,
    totalTaskCount: 2,
    elapsedMs: 4300,
    isRunFinished: false,
    watchErrorMessage: '',
    ...overrides,
  };
}

describe('InspectionRunTaskGraphLayoutComponent', () => {
  let fixture: ComponentFixture<InspectionRunTaskGraphLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [InspectionRunTaskGraphLayoutComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(InspectionRunTaskGraphLayoutComponent);
  });

  function render(viewModel: InspectionRunTaskGraphViewModel): HTMLElement {
    fixture.componentRef.setInput('viewModel', viewModel);
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  }

  it('renders the inspection name, the finished task ratio and the elapsed time', () => {
    const root = render(createViewModel());

    expect(root.querySelector('.inspection-name')?.textContent).toContain(
      'sample inspection',
    );
    expect(root.querySelector('.progress-label')?.textContent).toContain(
      '1 / 2 tasks',
    );
    expect(root.querySelector('.elapsed-label')?.textContent).toContain('4.3s');
  });

  it('fills the progress bar by the ratio of finished tasks', () => {
    const root = render(
      createViewModel({ finishedTaskCount: 3, totalTaskCount: 4 }),
    );

    const progressValue = root.querySelector<HTMLElement>('.progress-value');
    expect(progressValue?.style.width).toBe('75%');
  });

  it('keeps the progress bar empty when the graph has no task', () => {
    const root = render(
      createViewModel({
        nodes: [],
        edges: [],
        finishedTaskCount: 0,
        totalTaskCount: 0,
      }),
    );

    const progressValue = root.querySelector<HTMLElement>('.progress-value');
    expect(progressValue?.style.width).toBe('0%');
  });

  it('switches the run state label once the run finished', () => {
    expect(
      render(createViewModel()).querySelector('.run-state')?.textContent,
    ).toContain('Running');

    expect(
      render(createViewModel({ isRunFinished: true })).querySelector(
        '.run-state',
      )?.textContent,
    ).toContain('Finished');
  });

  it('shows the error banner only when an error message is present', () => {
    expect(render(createViewModel()).querySelector('.error-banner')).toBeNull();

    const root = render(
      createViewModel({ watchErrorMessage: 'connection lost' }),
    );
    expect(root.querySelector('.error-banner')?.textContent).toContain(
      'connection lost',
    );
  });

  it('renders the run phase legend and the dag canvas', () => {
    const root = render(createViewModel());

    expect(root.querySelector('khi-dag-run-phase-legend')).not.toBeNull();
    expect(root.querySelector('khi-dag-canvas')).not.toBeNull();
  });

  it('omits the dag canvas until at least one node arrives', () => {
    const root = render(createViewModel({ nodes: [], edges: [] }));

    expect(root.querySelector('khi-dag-canvas')).toBeNull();
  });
});
