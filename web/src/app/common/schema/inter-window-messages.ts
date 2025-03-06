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

import { ResourceTimeline } from 'src/app/store/timeline';

export const DIFF_PAGE_OPEN = 'DIFF_PAGE_OPEN';
export const UPDATE_SELECTED_RESOURCE_MESSAGE_KEY = 'UPDATE_SELECTED_RESOURCE';
export const UPDATE_GRAPH_DATA = 'UPDATE_GRAPH_DATA';
export const GRAPH_PAGE_OPEN = 'GRAPH_PAGE_OPEN';
/**
 * Main window broadcast this message when another resource was selected.
 */
export interface UpdateSelectedResourceMessage {
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
