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

import { Component, computed, inject, resource, signal } from '@angular/core';
import { GraphLayoutComponent } from 'src/app/graph/components/graph-layout.component';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { GraphConverterService } from 'src/app/services/graph-converter.service';
import {
  DEFAULT_DELETION_THRESHOLD_SECONDS,
  GraphData,
  emptyGraphData,
} from 'src/app/common/schema/graph-schema';

/**
 * Acts as a smart container for the graph view, delegating presentation to the layout component.
 */
@Component({
  selector: 'khi-graph-smart',
  templateUrl: './graph-smart.component.html',
  styleUrls: ['./graph-smart.component.scss'],
  imports: [GraphLayoutComponent],
})
export class GraphSmartComponent {
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly selectionManager = inject(SelectionManager);
  private readonly graphConverter = inject(GraphConverterService);

  /**
   * Holds the deletion retention threshold in seconds.
   */
  readonly deletionThresholdSeconds = signal<number>(
    DEFAULT_DELETION_THRESHOLD_SECONDS,
  );

  private readonly graphResource = resource({
    params: () => ({
      log: this.selectionManager.selectedLog(),
      timelineBitset: this.inspectionDataStore
        .timelineView()
        ?.filteredTimelineBitset(),
      deletionThresholdSeconds: this.deletionThresholdSeconds(),
    }),
    loader: async ({
      params: { log, timelineBitset, deletionThresholdSeconds },
      abortSignal,
    }) => {
      if (!log) {
        return emptyGraphData();
      }
      return this.graphConverter.getGraphDataAt(
        log.timestamp,
        timelineBitset,
        deletionThresholdSeconds,
        abortSignal,
      );
    },
  });

  /**
   * Computes the graph data derived from the currently selected log.
   */
  readonly graphData = computed<GraphData>(() => {
    if (this.graphResource.error()) {
      return emptyGraphData();
    }
    return this.graphResource.value() ?? emptyGraphData();
  });

  /**
   * Indicates whether the graph resource is currently loading.
   */
  readonly isLoading = this.graphResource.isLoading;
}
