/**
 * Copyright 2024 Google LLC
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
  GRAPH_PAGE_OPEN,
  UPDATE_GRAPH_DATA,
} from 'src/app/common/schema/inter-window-messages';
import { GraphDataConverterService } from 'src/app/services/graph-converter.service';
import { WindowConnectorService } from 'src/app/services/frame-connection/window-connector.service';
import { Injectable, inject } from '@angular/core';
import { UpdateGraphMessage } from 'src/app/services/frame-connection/frames/graph-page-datasource.service';

import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';

import { SelectionManager } from 'src/app/services/selection-manager.service';

@Injectable()
export class GraphPageDataSourceServer {
  private readonly graphConverter = inject(GraphDataConverterService);
  private readonly connector = inject(WindowConnectorService);
  private readonly store = inject(InspectionDataStore);
  private readonly selectionManager = inject(SelectionManager);

  private abortController?: AbortController;

  /**
   * Activates the server by listening to graph page open requests and responding with generated graph data asynchronously.
   */
  public activate() {
    this.connector.receiver(GRAPH_PAGE_OPEN).subscribe(async (message) => {
      const log = this.selectionManager.selectedLog();
      const timelineView = this.store.timelineView();
      if (log && timelineView) {
        if (this.abortController) {
          this.abortController.abort();
        }
        const controller = new AbortController();
        this.abortController = controller;

        try {
          const graphData = await this.graphConverter.getGraphDataAt(
            timelineView.filteredTimelines(),
            log.timestamp,
            controller.signal,
          );
          if (!controller.signal.aborted) {
            this.connector.unicast<UpdateGraphMessage>(
              UPDATE_GRAPH_DATA,
              {
                graphData,
              },
              message.sourceFrameId!,
            );
          }
        } catch {
          // Ignore cancellation error
        }
      }
    });
  }
}
