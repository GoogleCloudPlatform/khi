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
  effect,
  input,
  model,
  viewChild,
} from '@angular/core';
import {
  DEFAULT_DELETION_THRESHOLD_SECONDS,
  GraphData,
  emptyGraphData,
} from 'src/app/common/schema/graph-schema';
import { GraphRenderer } from 'src/app/pages/graph/architecture-graph/graph/renderer';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { GraphToolbarComponent } from 'src/app/graph/components/graph-toolbar.component';

/**
 * Renders the architecture graph layout based on the provided graph data.
 */
@Component({
  selector: 'khi-graph-layout',
  templateUrl: './graph-layout.component.html',
  styleUrls: ['./graph-layout.component.scss'],
  imports: [MatProgressSpinnerModule, GraphToolbarComponent],
})
export class GraphLayoutComponent implements AfterViewInit {
  /**
   * Holds the graph data to be rendered.
   */
  readonly graphData = input<GraphData>(emptyGraphData());

  /**
   * Indicates whether the graph data is currently loading.
   */
  readonly isLoading = input<boolean>(false);

  /**
   * Holds the deletion retention threshold in seconds for two-way binding.
   */
  readonly deletionThresholdSeconds = model<number>(
    DEFAULT_DELETION_THRESHOLD_SECONDS,
  );

  /**
   * References the container element for the SVG graph.
   */
  private readonly graphContainer =
    viewChild.required<ElementRef<HTMLDivElement>>('graphContainer');

  private graphRenderer?: GraphRenderer;

  constructor() {
    effect(() => {
      const data = this.graphData();
      if (this.graphRenderer && data) {
        this.graphRenderer.updateGraphData(data);
      }
    });
  }

  /**
   * Initializes the graph renderer after the view container is initialized.
   */
  ngAfterViewInit(): void {
    const container = this.graphContainer().nativeElement;
    this.graphRenderer = new GraphRenderer(container);
    this.graphRenderer.updateGraphData(this.graphData());
  }

  /**
   * Fits the graph within the view bounds and centers it in the viewport.
   */
  protected fitToView(): void {
    this.graphRenderer?.fitToView();
  }

  /**
   * Downloads the graph as an SVG image file.
   */
  protected downloadSvg(): void {
    this.graphRenderer?.downloadSvg();
  }

  /**
   * Downloads the graph as a PNG image file.
   */
  protected downloadPng(): void {
    void this.graphRenderer?.downloadPng();
  }
}
