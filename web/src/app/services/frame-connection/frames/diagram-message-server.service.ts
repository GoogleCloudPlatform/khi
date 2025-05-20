/**
 * Copyright 2025 Google LLC
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

import { inject, Injectable } from '@angular/core';
import { WindowConnectorService } from '../window-connector.service';
import { map } from 'rxjs';
import {
  QUERY_CURRENT_INSPECTION_METADATA,
  QUERY_DIAGRAM_DATA,
} from 'src/app/common/schema/inter-window-messages';
import { InspectionDataStoreService } from '../../inspection-data-store.service';
import { InspectionData } from 'src/app/store/inspection-data';
import { DiagramFrame } from '../../diagram/diagram-model-types';
import { SAMPLE_DIAGRAM } from '../../diagram/sample-diagram-models';

/**
 * Service that handles RPC communication for diagram data
 *
 * Runs on the main application page and responds to requests from diagram frames.
 * Provides inspection metadata and diagram model data to visualization components.
 */
@Injectable({ providedIn: 'root' })
export class DiagramMessageServer {
  /**
   * Communication service for handling inter-frame RPC
   */
  private readonly connector = inject(WindowConnectorService);

  /**
   * Data store containing the current inspection information
   */
  private readonly dataStore = inject(InspectionDataStoreService);

  /**
   * Registers RPC handlers for diagram-related queries
   *
   * Sets up handlers to respond to:
   * - Inspection metadata requests (time range)
   * - Diagram data requests for specific timestamps
   */
  public start() {
    this.connector.serveRPC(QUERY_CURRENT_INSPECTION_METADATA, () => {
      return this.dataStore.inspectionData.pipe(
        map((i) => {
          return i === null
            ? null
            : {
                // when no inspection data opened, return null as the response.
                startTime: i.range.begin,
                endTime: i.range.end,
              };
        }),
      );
    });

    this.connector.serveRPC(QUERY_DIAGRAM_DATA, (req) => {
      return this.dataStore.inspectionData.pipe(
        map((i) => {
          return i === null
            ? null
            : {
                // when no inspection data opened, return null as the response.
                model: getDiagramFrame(i, req.ts),
              };
        }),
      );
    });
  }
}

/**
 * Generates a diagram frame for the specified timestamp
 *
 * @param inspectionData - The current inspection data
 * @param time - Target timestamp for the diagram frame
 * @returns DiagramFrame representing the state at the specified time
 */
function getDiagramFrame(
  inspectionData: InspectionData,
  time: number,
): DiagramFrame {
  const clone = structuredClone(SAMPLE_DIAGRAM);
  clone.ts = time;
  clone.nodes[0].name = Math.random() + ''; // TODO:
  return clone;
}
