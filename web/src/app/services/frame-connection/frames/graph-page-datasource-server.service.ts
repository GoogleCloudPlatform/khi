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

import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';

import { SelectionManagerV2 } from 'src/app/services/selection-manager-v2.service';

@Injectable()
export class GraphPageDataSourceServer {
  private readonly graphConverter = inject(GraphDataConverterService);
  private readonly connector = inject(WindowConnectorService);
  private readonly store = inject(InspectionDataStoreV2);
  private readonly selectionManager = inject(SelectionManagerV2);

  public activate() {
    this.connector.receiver(GRAPH_PAGE_OPEN).subscribe((message) => {
      const log = this.selectionManager.selectedLog();
      const timelineView = this.store.timelineView();
      if (log && timelineView) {
        const graphData = this.graphConverter.getGraphDataAt(
          timelineView.filteredTimelines(),
          log.timestamp,
        );
        this.connector.unicast<UpdateGraphMessage>(
          UPDATE_GRAPH_DATA,
          {
            graphData,
          },
          message.sourceFrameId!,
        );
      }
    });
  }
}
