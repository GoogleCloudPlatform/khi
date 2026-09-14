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

import {
  DagLayoutResult,
  DagPositionedEdge,
  DagPositionedNode,
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';

/**
 * Options configuring DAG layout spacing and sizing dimensions.
 */
export interface DagLayoutOptions {
  /**
   * Width of each node card in pixels. Defaults to 420.
   */
  readonly nodeWidth?: number;

  /**
   * Height of each node card in pixels. Defaults to 104.
   */
  readonly nodeHeight?: number;

  /**
   * Horizontal spacing between columns (layers) in pixels. Defaults to 80.
   */
  readonly horizontalSpacing?: number;

  /**
   * Horizontal spacing between columns that contain tag dependency edges in pixels. Defaults to 200.
   */
  readonly tagHorizontalSpacing?: number;

  /**
   * Vertical spacing between sibling nodes in the same column in pixels. Defaults to 28.
   */
  readonly verticalSpacing?: number;

  /**
   * Horizontal canvas padding in pixels. Defaults to 48.
   */
  readonly paddingX?: number;

  /**
   * Vertical canvas padding in pixels. Defaults to 48.
   */
  readonly paddingY?: number;
}

const DEFAULT_NODE_WIDTH = 420;
const DEFAULT_NODE_HEIGHT = 104;
const DEFAULT_HORIZONTAL_SPACING = 80;
const DEFAULT_TAG_HORIZONTAL_SPACING = 200;
const DEFAULT_VERTICAL_SPACING = 28;
const DEFAULT_PADDING_X = 48;
const DEFAULT_PADDING_Y = 48;

/**
 * Computes layered DAG coordinates and cubic Bezier edge curves for a set of nodes and directed edges.
 */
export function computeDagLayout(
  nodes: readonly DagViewerNode[],
  edges: readonly DagViewerEdge[],
  options?: DagLayoutOptions,
): DagLayoutResult {
  const nodeWidth = options?.nodeWidth ?? DEFAULT_NODE_WIDTH;
  const nodeHeight = options?.nodeHeight ?? DEFAULT_NODE_HEIGHT;
  const horizontalSpacing =
    options?.horizontalSpacing ?? DEFAULT_HORIZONTAL_SPACING;
  const tagHorizontalSpacing =
    options?.tagHorizontalSpacing ?? DEFAULT_TAG_HORIZONTAL_SPACING;
  const verticalSpacing = options?.verticalSpacing ?? DEFAULT_VERTICAL_SPACING;
  const paddingX = options?.paddingX ?? DEFAULT_PADDING_X;
  const paddingY = options?.paddingY ?? DEFAULT_PADDING_Y;

  if (nodes.length === 0) {
    return {
      nodes: [],
      edges: [],
      width: 0,
      height: 0,
    };
  }

  const nodeMap = new Map<string, DagViewerNode>(
    nodes.map((node) => [node.id, node]),
  );

  // Filter edges to only those whose source and destination nodes exist.
  const validEdges = edges.filter(
    (edge) => nodeMap.has(edge.sourceId) && nodeMap.has(edge.destinationId),
  );

  const inEdges = new Map<string, DagViewerEdge[]>();
  for (const edge of validEdges) {
    const list = inEdges.get(edge.destinationId) ?? [];
    list.push(edge);
    inEdges.set(edge.destinationId, list);
  }

  // Sort nodes by topological order so upstream parents are evaluated before downstream children.
  const sortedNodes = [...nodes].sort(
    (a, b) => a.topologicalOrder - b.topologicalOrder,
  );

  // Compute longest-path topological depth (layer column index) for each node.
  const layerMap = new Map<string, number>();
  for (const node of sortedNodes) {
    let maxIncomingLayer = -1;
    const incoming = inEdges.get(node.id) ?? [];
    for (const edge of incoming) {
      const parentLayer = layerMap.get(edge.sourceId);
      if (parentLayer !== undefined && parentLayer > maxIncomingLayer) {
        maxIncomingLayer = parentLayer;
      }
    }
    layerMap.set(node.id, Math.max(0, maxIncomingLayer + 1));
  }

  // Group nodes by their assigned layer column.
  const layers = new Map<number, DagViewerNode[]>();
  let maxLayer = 0;
  for (const node of sortedNodes) {
    const layer = layerMap.get(node.id) ?? 0;
    if (layer > maxLayer) {
      maxLayer = layer;
    }
    const list = layers.get(layer) ?? [];
    list.push(node);
    layers.set(layer, list);
  }

  // Find the maximum column height to vertically center shorter columns.
  let maxColCount = 0;
  for (let l = 0; l <= maxLayer; l++) {
    const count = (layers.get(l) ?? []).length;
    if (count > maxColCount) {
      maxColCount = count;
    }
  }

  const maxColHeight =
    maxColCount * nodeHeight + Math.max(0, maxColCount - 1) * verticalSpacing;

  // Determine spacing for each layer transition based on whether tag dependency edges cross it.
  const layerSpacing: number[] = [];
  for (let l = 0; l < maxLayer; l++) {
    layerSpacing[l] = horizontalSpacing;
  }
  for (const edge of validEdges) {
    if (!edge.tag) {
      continue;
    }
    const sourceLayer = layerMap.get(edge.sourceId);
    const destLayer = layerMap.get(edge.destinationId);
    if (sourceLayer === undefined || destLayer === undefined) {
      continue;
    }
    if (destLayer > sourceLayer) {
      for (let l = sourceLayer; l < destLayer; l++) {
        layerSpacing[l] = Math.max(layerSpacing[l], tagHorizontalSpacing);
      }
    }
  }

  // Compute X coordinate for each layer column using cumulative spacing.
  const layerX = new Map<number, number>();
  let currentX = paddingX;
  for (let l = 0; l <= maxLayer; l++) {
    layerX.set(l, currentX);
    if (l < maxLayer) {
      currentX += nodeWidth + layerSpacing[l];
    }
  }

  // Position nodes within their respective layer columns.
  const positionedNodes: DagPositionedNode[] = [];
  const positionedNodeMap = new Map<string, DagPositionedNode>();

  for (let l = 0; l <= maxLayer; l++) {
    const colNodes = layers.get(l) ?? [];
    const colHeight =
      colNodes.length * nodeHeight +
      Math.max(0, colNodes.length - 1) * verticalSpacing;
    const colStartY = paddingY + (maxColHeight - colHeight) / 2;
    const colX = layerX.get(l) ?? paddingX;

    colNodes.forEach((node, idx) => {
      const x = colX;
      const y = colStartY + idx * (nodeHeight + verticalSpacing);
      const positioned: DagPositionedNode = {
        ...node,
        x,
        y,
        width: nodeWidth,
        height: nodeHeight,
        layer: l,
      };
      positionedNodes.push(positioned);
      positionedNodeMap.set(node.id, positioned);
    });
  }

  // Position edges and construct smooth cubic Bezier curves.
  const positionedEdges: DagPositionedEdge[] = [];
  for (const edge of validEdges) {
    const source = positionedNodeMap.get(edge.sourceId);
    const dest = positionedNodeMap.get(edge.destinationId);
    if (!source || !dest) {
      continue;
    }

    const startX = source.x + source.width;
    const startY = source.y + source.height / 2;
    const endX = dest.x;
    const endY = dest.y + dest.height / 2;

    let pathD: string;
    let labelX: number;
    let labelY: number;

    if (endX > startX) {
      const dx = endX - startX;
      const cx1 = startX + dx * 0.45;
      const cy1 = startY;
      const cx2 = endX - dx * 0.45;
      const cy2 = endY;
      pathD = `M ${startX} ${startY} C ${cx1} ${cy1}, ${cx2} ${cy2}, ${endX} ${endY}`;
      labelX = (startX + endX) / 2;
      labelY = (startY + endY) / 2;
    } else {
      // Loopback curve for backward edges or same-column connections.
      const loopOffset = 50;
      const cx1 = startX + loopOffset;
      const cy1 = startY - loopOffset;
      const cx2 = endX - loopOffset;
      const cy2 = endY - loopOffset;
      pathD = `M ${startX} ${startY} C ${cx1} ${cy1}, ${cx2} ${cy2}, ${endX} ${endY}`;
      labelX = (startX + endX) / 2;
      labelY = Math.min(startY, endY) - loopOffset;
    }

    positionedEdges.push({
      ...edge,
      pathD,
      startX,
      startY,
      endX,
      endY,
      labelX,
      labelY,
    });
  }

  const lastLayerX = layerX.get(maxLayer) ?? paddingX;
  const totalWidth = lastLayerX + nodeWidth + paddingX;
  const totalHeight = paddingY * 2 + maxColHeight;

  return {
    nodes: positionedNodes,
    edges: positionedEdges,
    width: totalWidth,
    height: totalHeight,
  };
}
