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

import { GraphData } from 'src/app/common/schema/graph-schema';
import { InterframeDatasource } from '../inter-frame-datasource.service';
import { Injectable } from '@angular/core';
import { WindowConnectorService } from '../window-connector.service';
import {
  GRAPH_PAGE_OPEN,
  UPDATE_GRAPH_DATA,
} from 'src/app/common/schema/inter-window-messages';

export interface UpdateGraphMessage {
  graphData: GraphData;
}

@Injectable()
export class GraphPageDataSource extends InterframeDatasource<GraphData> {
  private enabled = false;

  constructor(private connector: WindowConnectorService) {
    super();
  }
  override enable(): void {
    if (this.enabled) {
      return;
    }
    this.connector
      .receiver<UpdateGraphMessage>(UPDATE_GRAPH_DATA)
      .subscribe((graphData) => {
        this.data$.next(graphData.data.graphData);
      });
    this.connector.broadcast(GRAPH_PAGE_OPEN, {});
  }
  override disable(): void {}
}
