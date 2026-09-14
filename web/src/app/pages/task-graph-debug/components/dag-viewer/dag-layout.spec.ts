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

import { TaskDependencyCardinality } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { computeDagLayout } from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-layout';
import {
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';

describe('computeDagLayout', () => {
  const createMockNode = (
    id: string,
    refId: string,
    topologicalOrder: number,
  ): DagViewerNode => ({
    id,
    referenceId: refId,
    isFeature: false,
    isFormTask: false,
    topologicalOrder,
    priority: 100,
    labels: {},
    outputType: 'string',
    providedTags: [],
  });

  const createMockEdge = (
    sourceId: string,
    destinationId: string,
    cardinality: TaskDependencyCardinality = TaskDependencyCardinality.POINT_TO_POINT,
    tag: string = '',
  ): DagViewerEdge => ({
    id: `${sourceId}->${destinationId}`,
    sourceId,
    destinationId,
    sourceReferenceId: sourceId,
    cardinality,
    tag,
    priority: 100,
    outputType: 'string',
  });

  it('returns empty result when nodes array is empty', () => {
    const result = computeDagLayout([], []);
    expect(result.nodes.length).toBe(0);
    expect(result.edges.length).toBe(0);
    expect(result.width).toBe(0);
    expect(result.height).toBe(0);
  });

  it('positions a single node at layer 0 with padding', () => {
    const node = createMockNode('task-a', 'ref-a', 0);
    const result = computeDagLayout([node], []);

    expect(result.nodes.length).toBe(1);
    expect(result.nodes[0].layer).toBe(0);
    expect(result.nodes[0].x).toBe(48); // default paddingX
    expect(result.nodes[0].y).toBe(48); // default paddingY
    expect(result.width).toBeGreaterThan(0);
    expect(result.height).toBeGreaterThan(0);
  });

  it('layers linear dependency chain in sequential columns', () => {
    const nodeA = createMockNode('task-a', 'ref-a', 0);
    const nodeB = createMockNode('task-b', 'ref-b', 1);
    const nodeC = createMockNode('task-c', 'ref-c', 2);

    const edgeAB = createMockEdge('task-a', 'task-b');
    const edgeBC = createMockEdge('task-b', 'task-c');

    const result = computeDagLayout([nodeA, nodeB, nodeC], [edgeAB, edgeBC]);

    expect(result.nodes.length).toBe(3);
    const posA = result.nodes.find((n) => n.id === 'task-a')!;
    const posB = result.nodes.find((n) => n.id === 'task-b')!;
    const posC = result.nodes.find((n) => n.id === 'task-c')!;

    expect(posA.layer).toBe(0);
    expect(posB.layer).toBe(1);
    expect(posC.layer).toBe(2);

    expect(posA.x).toBeLessThan(posB.x);
    expect(posB.x).toBeLessThan(posC.x);

    expect(result.edges.length).toBe(2);
    expect(result.edges[0].pathD).toContain('M ');
    expect(result.edges[0].pathD).toContain(' C ');
  });

  it('layers diamond dependency graph correctly', () => {
    // A -> B, A -> C, B -> D, C -> D
    const nodeA = createMockNode('task-a', 'ref-a', 0);
    const nodeB = createMockNode('task-b', 'ref-b', 1);
    const nodeC = createMockNode('task-c', 'ref-c', 2);
    const nodeD = createMockNode('task-d', 'ref-d', 3);

    const edges = [
      createMockEdge('task-a', 'task-b'),
      createMockEdge('task-a', 'task-c'),
      createMockEdge('task-b', 'task-d'),
      createMockEdge('task-c', 'task-d'),
    ];

    const result = computeDagLayout([nodeA, nodeB, nodeC, nodeD], edges);

    const posA = result.nodes.find((n) => n.id === 'task-a')!;
    const posB = result.nodes.find((n) => n.id === 'task-b')!;
    const posC = result.nodes.find((n) => n.id === 'task-c')!;
    const posD = result.nodes.find((n) => n.id === 'task-d')!;

    expect(posA.layer).toBe(0);
    expect(posB.layer).toBe(1);
    expect(posC.layer).toBe(1);
    expect(posD.layer).toBe(2);

    // Nodes B and C should have the same X but different Y coordinates.
    expect(posB.x).toBe(posC.x);
    expect(posB.y).not.toBe(posC.y);

    expect(result.edges.length).toBe(4);
  });

  it('ignores dangling edges referencing non-existent nodes', () => {
    const nodeA = createMockNode('task-a', 'ref-a', 0);
    const danglingEdge = createMockEdge('task-a', 'task-missing');

    const result = computeDagLayout([nodeA], [danglingEdge]);

    expect(result.nodes.length).toBe(1);
    expect(result.edges.length).toBe(0);
  });

  it('handles backward or same-layer edges with loopback path', () => {
    const nodeA = createMockNode('task-a', 'ref-a', 0);
    const nodeB = createMockNode('task-b', 'ref-b', 1);
    // Backward edge B -> A
    const backwardEdge = createMockEdge('task-b', 'task-a');

    const result = computeDagLayout([nodeA, nodeB], [backwardEdge]);

    expect(result.edges.length).toBe(1);
    expect(result.edges[0].pathD).toContain('M ');
    expect(result.edges[0].pathD).toContain(' C ');
  });

  it('widens horizontal spacing between layers when edges have tags', () => {
    const nodeA = createMockNode('task-a', 'ref-a', 0);
    const nodeB = createMockNode('task-b', 'ref-b', 1);

    // Standard edge without tag: spacing should be default 80
    const edgeNoTag = createMockEdge('task-a', 'task-b');
    const resultNoTag = computeDagLayout([nodeA, nodeB], [edgeNoTag]);
    const posANoTag = resultNoTag.nodes.find((n) => n.id === 'task-a')!;
    const posBNoTag = resultNoTag.nodes.find((n) => n.id === 'task-b')!;
    const spacingNoTag = posBNoTag.x - (posANoTag.x + posANoTag.width);
    expect(spacingNoTag).toBe(80);

    // Edge with tag: spacing should expand to default tagHorizontalSpacing 200
    const edgeWithTag = createMockEdge(
      'task-a',
      'task-b',
      TaskDependencyCardinality.FAN_IN,
      'khi.google.com/inspection/timeline-producer',
    );
    const resultWithTag = computeDagLayout([nodeA, nodeB], [edgeWithTag]);
    const posAWithTag = resultWithTag.nodes.find((n) => n.id === 'task-a')!;
    const posBWithTag = resultWithTag.nodes.find((n) => n.id === 'task-b')!;
    const spacingWithTag = posBWithTag.x - (posAWithTag.x + posAWithTag.width);
    expect(spacingWithTag).toBe(200);
  });
});
