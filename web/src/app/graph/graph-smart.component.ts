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

import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { GraphLayoutComponent } from 'src/app/graph/components/graph-layout.component';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import { SelectionManagerV2 } from 'src/app/services/selection-manager-v2.service';
import { GraphDataConverterService } from 'src/app/services/graph-converter.service';
import { GraphData, emptyGraphData } from 'src/app/common/schema/graph-schema';

/**
 * Acts as a smart container for the graph view, delegating presentation to the layout component.
 */
@Component({
  selector: 'khi-graph-smart',
  templateUrl: './graph-smart.component.html',
  styleUrls: ['./graph-smart.component.scss'],
  imports: [CommonModule, GraphLayoutComponent],
})
export class GraphSmartComponent {
  private readonly inspectionDataStore = inject(InspectionDataStoreV2);
  private readonly selectionManager = inject(SelectionManagerV2);
  private readonly graphConverter = inject(GraphDataConverterService);

  /**
   * Computed signal holding the graph data derived from the currently selected log.
   */
  readonly graphData = computed<GraphData>(() => {
    const log = this.selectionManager.selectedLog();
    const timelineView = this.inspectionDataStore.timelineView();
    if (!log || !timelineView) {
      return emptyGraphData();
    }
    return this.graphConverter.getGraphDataAt(
      timelineView.filteredTimelines(),
      log.timestamp,
    );
  });
}
