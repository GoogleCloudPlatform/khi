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
  viewChild,
} from '@angular/core';
import { GraphData, emptyGraphData } from 'src/app/common/schema/graph-schema';
import { GraphRenderer } from 'src/app/pages/graph/architecture-graph/graph/renderer';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

/**
 * Renders the architecture graph layout based on the provided graph data.
 */
@Component({
  selector: 'khi-graph-layout',
  templateUrl: './graph-layout.component.html',
  styleUrls: ['./graph-layout.component.scss'],
  imports: [MatProgressSpinnerModule],
})
export class GraphLayoutComponent implements AfterViewInit {
  /**
   * Input signal holding the graph data to be rendered.
   */
  readonly graphData = input<GraphData>(emptyGraphData());

  /**
   * Input signal indicating whether the graph data is currently loading.
   */
  readonly isLoading = input<boolean>(false);

  /**
   * Reference to the container element for the SVG graph.
   */
  readonly graphContainer =
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
}
