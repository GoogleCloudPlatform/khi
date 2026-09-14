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

import { Meta, StoryObj } from '@storybook/angular';
import { InspectionRunTaskGraphLayoutComponent } from 'src/app/dialogs/inspection-run-task-graph/components/inspection-run-task-graph-layout.component';
import { InspectionRunTaskGraphViewModel } from 'src/app/dialogs/inspection-run-task-graph/types/inspection-run-task-graph.viewmodel';
import { TaskDependencyCardinality } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import {
  DagNodeRunPhase,
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

const runPhaseByNodeId: Record<string, DagNodeRunPhase> = {
  'input-cluster-name#default': DagNodeRunPhase.DONE,
  'query-k8s-audit#default': DagNodeRunPhase.DONE,
  'parse-k8s-audit#default': DagNodeRunPhase.RUNNING,
  'build-timeline#default': DagNodeRunPhase.WAITING,
  'serialize-result#default': DagNodeRunPhase.WAITING,
};

const runDurationMsByNodeId: Record<string, number> = {
  'input-cluster-name#default': 120,
  'query-k8s-audit#default': 18400,
  'parse-k8s-audit#default': 4300,
  'build-timeline#default': 0,
  'serialize-result#default': 0,
};

const sampleNodes: DagViewerNode[] = Object.keys(runPhaseByNodeId).map(
  (id, index) => ({
    id,
    referenceId: id.split('#')[0],
    isFeature: index === 2,
    isFormTask: index === 0,
    topologicalOrder: index,
    priority: 0,
    labels: {},
    outputType: 'string',
    providedTags: [],
    runPhase: runPhaseByNodeId[id],
    runDurationMs: runDurationMsByNodeId[id],
  }),
);

const sampleEdges: DagViewerEdge[] = sampleNodes
  .slice(1)
  .map((node, index) => ({
    id: `edge-${index}`,
    sourceId: sampleNodes[index].id,
    destinationId: node.id,
    sourceReferenceId: sampleNodes[index].referenceId,
    cardinality: TaskDependencyCardinality.POINT_TO_POINT,
    tag: '',
    priority: 0,
    outputType: 'string',
  }));

const runningViewModel: InspectionRunTaskGraphViewModel = {
  inspectionName: 'my-cluster / 2026-09-13',
  nodes: sampleNodes,
  edges: sampleEdges,
  finishedTaskCount: 2,
  totalTaskCount: 5,
  elapsedMs: 22800,
  isRunFinished: false,
  watchErrorMessage: '',
};

const meta: Meta<InspectionRunTaskGraphLayoutComponent> = {
  title: 'Dialogs/InspectionRunTaskGraph/InspectionRunTaskGraphLayout',
  component: InspectionRunTaskGraphLayoutComponent,
  tags: ['autodocs'],
  args: {
    viewModel: runningViewModel,
  },
};

export default meta;
type Story = StoryObj<InspectionRunTaskGraphLayoutComponent>;

export const Running: Story = {};

export const Finished: Story = {
  args: {
    viewModel: {
      ...runningViewModel,
      nodes: sampleNodes.map((node) => ({
        ...node,
        runPhase: DagNodeRunPhase.DONE,
        runDurationMs: node.runDurationMs || 900,
      })),
      finishedTaskCount: 5,
      isRunFinished: true,
    },
  },
};

export const WithError: Story = {
  args: {
    viewModel: {
      ...runningViewModel,
      nodes: sampleNodes.map((node) =>
        node.runPhase === DagNodeRunPhase.RUNNING
          ? { ...node, runPhase: DagNodeRunPhase.ERROR }
          : node,
      ),
      isRunFinished: true,
      watchErrorMessage: 'Lost connection while observing the inspection run.',
    },
  },
};

export const Empty: Story = {
  args: {
    viewModel: {
      inspectionName: 'connecting...',
      nodes: [],
      edges: [],
      finishedTaskCount: 0,
      totalTaskCount: 0,
      elapsedMs: 0,
      isRunFinished: false,
      watchErrorMessage: '',
    },
  },
};
