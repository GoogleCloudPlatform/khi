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
import { GraphDataConverterService } from '../../graph-converter.service';
import { SelectionManagerService } from '../../selection-manager.service';
import { WindowConnectorService } from '../window-connector.service';
import { withLatestFrom } from 'rxjs';
import { Inject, Injectable } from '@angular/core';
import { UpdateGraphMessage } from './graph-page-datasource.service';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from '../../timeline-filter.service';

@Injectable()
export class GraphPageDataSourceServer {
  constructor(
    private graphConverter: GraphDataConverterService,
    private connector: WindowConnectorService,
    private selectionManager: SelectionManagerService,
    @Inject(DEFAULT_TIMELINE_FILTER) private filter: TimelineFilter,
  ) {}

  public activate() {
    this.connector
      .receiver(GRAPH_PAGE_OPEN)
      .pipe(
        withLatestFrom(
          this.selectionManager.selectedLog,
          this.filter.filteredTimeline,
        ),
      )
      .subscribe(([message, log, timeline]) => {
        if (log && timeline) {
          const graphData = this.graphConverter.getGraphDataAt(
            timeline,
            log.time,
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
