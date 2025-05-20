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

import { DiagramFrame } from 'src/app/services/diagram/diagram-model-types';
import { KHIWindowRPCType } from 'src/app/services/frame-connection/window-connector.service';
import { ResourceTimeline } from 'src/app/store/timeline';

export const UPDATE_SELECTED_RESOURCE_MESSAGE_KEY = 'UPDATE_SELECTED_RESOURCE';
export const UPDATE_GRAPH_DATA = 'UPDATE_GRAPH_DATA';
export const GRAPH_PAGE_OPEN = 'GRAPH_PAGE_OPEN';

/**
 * A RPC to notify a diff page opened and obtain the resource information.
 */
export const DIFF_PAGE_OPEN = new KHIWindowRPCType<
  object,
  DiffPageOpenResponse
>('DIFF_PAGE_OPEN');

/**
 * Main window broadcast this message when another resource was selected.
 */
export interface DiffPageOpenResponse {
  timeline: ResourceTimeline;
  logIndex: number;
}
/**
 * A viewmodel for entire diff page.
 */
export interface DiffPageViewModel {
  timeline: ResourceTimeline;
  logIndex: number;
}

/**
 * RPC type for getting metadata used in the diagram page.
 * When no inspection data opened at the time, main page returns null.
 */
export const QUERY_CURRENT_INSPECTION_METADATA = new KHIWindowRPCType<
  object,
  QueryCurrentInspectionMetadataResponse | null
>('QUERY_CURRENT_INSPECTION_DATA');

export interface QueryCurrentInspectionMetadataResponse {
  startTime: number;
  endTime: number;
}

/**
 * RPC type for getting the diagram model for specified time.
 * Main page returns null when no inspection data opened at the time.
 */
export const QUERY_DIAGRAM_DATA = new KHIWindowRPCType<
  QueryDiagramDataRequest,
  QueryDiagramDataResponse | null
>('QUERY_DIAGRAM_DATA');

/**
 * Request body type for QUERY_DIAGRAM_DATA message.
 */
export interface QueryDiagramDataRequest {
  ts: number;
}

/**
 * Response body type for QUERY_DIAGRAM_DATA message.
 */
export interface QueryDiagramDataResponse {
  model: DiagramFrame;
}
