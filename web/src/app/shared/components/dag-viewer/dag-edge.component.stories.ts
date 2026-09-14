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

import { Meta, StoryObj, componentWrapperDecorator } from '@storybook/angular';
import { TaskDependencyCardinality } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { DagEdgeComponent } from 'src/app/shared/components/dag-viewer/dag-edge.component';
import { DagPositionedEdge } from 'src/app/shared/components/dag-viewer/dag-viewer.model';

const mockEdge: DagPositionedEdge = {
  id: 'task-a->task-b',
  sourceId: 'task-a',
  destinationId: 'task-b',
  sourceReferenceId: 'ref-a',
  cardinality: TaskDependencyCardinality.POINT_TO_POINT,
  tag: '',
  priority: 100,
  outputType: '*parser.ParsedPod',
  pathD: 'M 40 100 C 140 100, 260 100, 360 100',
  startX: 40,
  startY: 100,
  endX: 360,
  endY: 100,
  labelX: 200,
  labelY: 100,
};

const meta: Meta<DagEdgeComponent> = {
  title: 'Shared/DagViewer/DagEdge',
  component: DagEdgeComponent,
  tags: ['autodocs'],
  decorators: [
    componentWrapperDecorator(
      (story) => `
        <svg width="400" height="200" style="background: #fafafa; border: 1px solid #e0e0e0; border-radius: 4px;">
          <defs>
            <marker id="arrow-marker" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 8 5 L 0 8.5 z" fill="#79747e" />
            </marker>
            <marker id="arrow-marker-highlighted" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 8 5 L 0 8.5 z" fill="#6750a4" />
            </marker>
            <marker id="arrow-marker-fan-in" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 8 5 L 0 8.5 z" fill="#7d5260" />
            </marker>
            <marker id="arrow-marker-satisfied" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 8 5 L 0 8.5 z" fill="#388e3c" />
            </marker>
          </defs>
          ${story}
        </svg>
      `,
    ),
  ],
  render: (args) => ({
    props: args,
    template: `
      <g
        khi-dag-edge
        [edge]="edge"
        [isHighlighted]="isHighlighted"
        [isDimmed]="isDimmed"
        [isSatisfied]="isSatisfied"
      ></g>
    `,
  }),
  args: {
    edge: mockEdge,
    isHighlighted: false,
    isDimmed: false,
    isSatisfied: false,
  },
};

export default meta;
type Story = StoryObj<DagEdgeComponent>;

export const PointToPoint: Story = {
  args: {},
};

export const Satisfied: Story = {
  args: {
    isSatisfied: true,
  },
};

export const Highlighted: Story = {
  args: {
    isHighlighted: true,
  },
};

export const Dimmed: Story = {
  args: {
    isDimmed: true,
  },
};

export const FanInWithTag: Story = {
  args: {
    edge: {
      ...mockEdge,
      cardinality: TaskDependencyCardinality.FAN_IN,
      tag: 'audit-logs',
    },
  },
};

export const FanInWithDomainTag: Story = {
  args: {
    edge: {
      ...mockEdge,
      cardinality: TaskDependencyCardinality.FAN_IN,
      tag: 'khi.google.com/inspection/timeline-producer',
    },
  },
};
