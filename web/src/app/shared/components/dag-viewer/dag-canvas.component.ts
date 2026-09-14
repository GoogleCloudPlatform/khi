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
  AfterViewInit,
  Component,
  ElementRef,
  computed,
  input,
  model,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { DagEdgeComponent } from 'src/app/shared/components/dag-viewer/dag-edge.component';
import { computeDagLayout } from 'src/app/shared/components/dag-viewer/dag-layout';
import { DagNodeDetailPanelComponent } from 'src/app/shared/components/dag-viewer/dag-node-detail-panel.component';
import { DagNodeComponent } from 'src/app/shared/components/dag-viewer/dag-node.component';
import {
  DagLayoutResult,
  DagPositionedNode,
  DagViewerEdge,
  DagViewerNode,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

const MIN_ZOOM = 0.15;
const MAX_ZOOM = 2.5;
const ZOOM_STEP = 1.2;

/**
 * Interactive SVG canvas for rendering, exploring, and navigating the resolved task execution DAG.
 */
@Component({
  selector: 'khi-dag-canvas',
  templateUrl: './dag-canvas.component.html',
  styleUrls: ['./dag-canvas.component.scss'],
  imports: [
    MatIconModule,
    MatButtonModule,
    KHIIconRegistrationModule,
    DagEdgeComponent,
    DagNodeComponent,
    DagNodeDetailPanelComponent,
  ],
})
export class DagCanvasComponent implements AfterViewInit {
  /**
   * Reference to the canvas container element for viewport sizing.
   */
  readonly canvasContainer =
    viewChild<ElementRef<HTMLElement>>('canvasContainer');

  /**
   * Task execution nodes to lay out and render.
   */
  readonly nodes = input<readonly DagViewerNode[]>([]);

  /**
   * Directed dependency edges connecting task nodes.
   */
  readonly edges = input<readonly DagViewerEdge[]>([]);

  /**
   * Whether to display the node detail slide-over panel upon node selection.
   */
  readonly showDetailPanel = input<boolean>(true);

  /**
   * Canvas horizontal pan offset in pixels.
   */
  readonly panX = signal<number>(0);

  /**
   * Canvas vertical pan offset in pixels.
   */
  readonly panY = signal<number>(0);

  /**
   * Current canvas zoom scaling factor.
   */
  readonly zoom = signal<number>(1);

  /**
   * Human-readable zoom percentage string.
   */
  readonly zoomPercentage = computed(() => `${Math.round(this.zoom() * 100)}%`);

  /**
   * Currently selected node implementation ID, or null.
   */
  readonly selectedNodeId = model<string | null>(null);

  /**
   * Emitted when a node is clicked.
   */
  readonly nodeClick = output<string>();

  /**
   * Currently hovered node implementation ID, or null.
   */
  readonly hoveredNodeId = signal<string | null>(null);

  /**
   * Whether the user is actively dragging the canvas.
   */
  readonly isDragging = signal<boolean>(false);

  private dragStartX = 0;
  private dragStartY = 0;

  /**
   * Computed 2D layered coordinates and Bezier curves for all nodes and edges.
   */
  readonly layoutResult = computed<DagLayoutResult>(() =>
    computeDagLayout(this.nodes(), this.edges()),
  );

  /**
   * Node map for quick lookup by implementation ID.
   */
  readonly nodeMap = computed<ReadonlyMap<string, DagPositionedNode>>(() => {
    const map = new Map<string, DagPositionedNode>();
    for (const node of this.layoutResult().nodes) {
      map.set(node.id, node);
    }
    return map;
  });

  /**
   * Active focal node ID (selected takes precedence over hovered).
   */
  readonly activeNodeId = computed<string | null>(
    () => this.selectedNodeId() ?? this.hoveredNodeId(),
  );

  /**
   * Set of all ancestor node IDs upstream of the active node.
   */
  readonly upstreamAncestorIds = computed<ReadonlySet<string>>(() => {
    const activeId = this.activeNodeId();
    if (!activeId) {
      return new Set();
    }
    const inEdges = new Map<string, string[]>();
    for (const edge of this.edges()) {
      const list = inEdges.get(edge.destinationId) ?? [];
      list.push(edge.sourceId);
      inEdges.set(edge.destinationId, list);
    }
    const visited = new Set<string>();
    const queue = [...(inEdges.get(activeId) ?? [])];
    while (queue.length > 0) {
      const current = queue.shift()!;
      if (!visited.has(current)) {
        visited.add(current);
        queue.push(...(inEdges.get(current) ?? []));
      }
    }
    return visited;
  });

  /**
   * Set of all descendant node IDs downstream of the active node.
   */
  readonly downstreamDescendantIds = computed<ReadonlySet<string>>(() => {
    const activeId = this.activeNodeId();
    if (!activeId) {
      return new Set();
    }
    const outEdges = new Map<string, string[]>();
    for (const edge of this.edges()) {
      const list = outEdges.get(edge.sourceId) ?? [];
      list.push(edge.destinationId);
      outEdges.set(edge.sourceId, list);
    }
    const visited = new Set<string>();
    const queue = [...(outEdges.get(activeId) ?? [])];
    while (queue.length > 0) {
      const current = queue.shift()!;
      if (!visited.has(current)) {
        visited.add(current);
        queue.push(...(outEdges.get(current) ?? []));
      }
    }
    return visited;
  });

  /**
   * Set of all highlighted node IDs (focal + ancestors + descendants).
   */
  readonly highlightedNodeIds = computed<ReadonlySet<string>>(() => {
    const activeId = this.activeNodeId();
    if (!activeId) {
      return new Set();
    }
    const set = new Set<string>([activeId]);
    for (const id of this.upstreamAncestorIds()) {
      set.add(id);
    }
    for (const id of this.downstreamDescendantIds()) {
      set.add(id);
    }
    return set;
  });

  /**
   * Set of all highlighted edge IDs directly connecting highlighted nodes in the active path.
   */
  readonly highlightedEdgeIds = computed<ReadonlySet<string>>(() => {
    const activeId = this.activeNodeId();
    if (!activeId) {
      return new Set();
    }
    const highlightedNodes = this.highlightedNodeIds();
    const set = new Set<string>();
    for (const edge of this.edges()) {
      if (
        highlightedNodes.has(edge.sourceId) &&
        highlightedNodes.has(edge.destinationId)
      ) {
        set.add(edge.id);
      }
    }
    return set;
  });

  /**
   * Currently selected node object, or null.
   */
  readonly selectedNode = computed<DagViewerNode | null>(() => {
    const id = this.selectedNodeId();
    if (!id) {
      return null;
    }
    return this.nodeMap().get(id) ?? null;
  });

  /**
   * Direct upstream predecessor tasks for the selected node.
   */
  readonly selectedUpstreamNodes = computed<readonly DagViewerNode[]>(() => {
    const selId = this.selectedNodeId();
    if (!selId) {
      return [];
    }
    const sourceIds = this.edges()
      .filter((e) => e.destinationId === selId)
      .map((e) => e.sourceId);
    const nodes: DagViewerNode[] = [];
    for (const id of sourceIds) {
      const n = this.nodeMap().get(id);
      if (n) {
        nodes.push(n);
      }
    }
    return nodes;
  });

  /**
   * Direct downstream consumer tasks for the selected node.
   */
  readonly selectedDownstreamNodes = computed<readonly DagViewerNode[]>(() => {
    const selId = this.selectedNodeId();
    if (!selId) {
      return [];
    }
    const destIds = this.edges()
      .filter((e) => e.sourceId === selId)
      .map((e) => e.destinationId);
    const nodes: DagViewerNode[] = [];
    for (const id of destIds) {
      const n = this.nodeMap().get(id);
      if (n) {
        nodes.push(n);
      }
    }
    return nodes;
  });

  /**
   * SVG transform string applied to the root graph `<g>`.
   */
  readonly transformString = computed(
    () => `translate(${this.panX()}, ${this.panY()}) scale(${this.zoom()})`,
  );

  /**
   * Fits the graph to the canvas container upon initial view.
   */
  ngAfterViewInit(): void {
    this.fitToView();
  }

  /**
   * Determines if a specific node is highlighted.
   */
  isNodeHighlighted(id: string): boolean {
    return this.highlightedNodeIds().has(id);
  }

  /**
   * Determines if a specific node is dimmed.
   */
  isNodeDimmed(id: string): boolean {
    return this.activeNodeId() !== null && !this.highlightedNodeIds().has(id);
  }

  /**
   * Determines if a specific edge is highlighted.
   */
  isEdgeHighlighted(id: string): boolean {
    return this.highlightedEdgeIds().has(id);
  }

  /**
   * Determines if a specific edge is dimmed.
   */
  isEdgeDimmed(id: string): boolean {
    return this.activeNodeId() !== null && !this.highlightedEdgeIds().has(id);
  }

  /**
   * Handles task card selection.
   */
  onNodeSelected(id: string): void {
    if (this.selectedNodeId() === id) {
      this.selectedNodeId.set(null);
    } else {
      this.selectedNodeId.set(id);
    }
    this.nodeClick.emit(id);
  }

  /**
   * Handles task card hover state.
   */
  onNodeHovered(id: string | null): void {
    this.hoveredNodeId.set(id);
  }

  /**
   * Handles mouse down on the SVG canvas to initiate panning.
   */
  onMouseDown(event: MouseEvent): void {
    // Only initiate dragging on primary click
    if (event.button !== 0) {
      return;
    }
    this.isDragging.set(true);
    this.dragStartX = event.clientX - this.panX();
    this.dragStartY = event.clientY - this.panY();
  }

  /**
   * Handles mouse movement to update canvas pan offset.
   */
  onMouseMove(event: MouseEvent): void {
    if (!this.isDragging()) {
      return;
    }
    this.panX.set(event.clientX - this.dragStartX);
    this.panY.set(event.clientY - this.dragStartY);
  }

  /**
   * Terminates canvas drag interaction.
   */
  onMouseUp(): void {
    this.isDragging.set(false);
  }

  /**
   * Handles mouse wheel for zooming anchored to cursor position.
   */
  onWheel(event: WheelEvent): void {
    event.preventDefault();
    const container = this.canvasContainer()?.nativeElement;
    if (!container) {
      return;
    }

    const rect = container.getBoundingClientRect();
    const cursorX = event.clientX - rect.left;
    const cursorY = event.clientY - rect.top;

    const zoomFactor = event.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP;
    const currentZoom = this.zoom();
    const newZoom = Math.min(
      MAX_ZOOM,
      Math.max(MIN_ZOOM, currentZoom * zoomFactor),
    );

    if (newZoom === currentZoom) {
      return;
    }

    // Zoom anchored to the cursor position:
    // (cursorX - panX) / currentZoom = (cursorX - newPanX) / newZoom
    const newPanX = cursorX - (cursorX - this.panX()) * (newZoom / currentZoom);
    const newPanY = cursorY - (cursorY - this.panY()) * (newZoom / currentZoom);

    this.zoom.set(newZoom);
    this.panX.set(newPanX);
    this.panY.set(newPanY);
  }

  /**
   * Zooms in by one step.
   */
  zoomIn(): void {
    this.zoom.update((z) => Math.min(MAX_ZOOM, z * ZOOM_STEP));
  }

  /**
   * Zooms out by one step.
   */
  zoomOut(): void {
    this.zoom.update((z) => Math.max(MIN_ZOOM, z / ZOOM_STEP));
  }

  /**
   * Resets zoom to 1.0 and pan to origin.
   */
  resetZoom(): void {
    this.zoom.set(1);
    this.panX.set(0);
    this.panY.set(0);
  }

  /**
   * Automatically calculates zoom and pan to fit the complete graph in view.
   */
  fitToView(): void {
    const container = this.canvasContainer()?.nativeElement;
    if (!container) {
      return;
    }

    const containerWidth = container.clientWidth;
    const containerHeight = container.clientHeight;
    const layout = this.layoutResult();

    if (
      containerWidth === 0 ||
      containerHeight === 0 ||
      layout.width === 0 ||
      layout.height === 0
    ) {
      return;
    }

    const scaleX = (containerWidth - 60) / layout.width;
    const scaleY = (containerHeight - 60) / layout.height;
    const fitScale = Math.min(
      1.0,
      Math.max(MIN_ZOOM, Math.min(scaleX, scaleY)),
    );

    const centeredX = (containerWidth - layout.width * fitScale) / 2;
    const centeredY = (containerHeight - layout.height * fitScale) / 2;

    this.zoom.set(fitScale);
    this.panX.set(centeredX);
    this.panY.set(centeredY);
  }

  /**
   * Deselects the currently selected node.
   */
  closeDetailPanel(): void {
    this.selectedNodeId.set(null);
  }
}
