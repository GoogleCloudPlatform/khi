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

import { ReferenceResolverStore } from 'src/app/common/loader/reference-resolver';
import { InspectionData } from 'src/app/store/inspection-data';

/**
 * LifecycleExtension is an interface with set of event handlers.
 * This provides extensible points for plugins to be triggerd on various frontendevents in KHI.
 */
export interface LifecycleHookExtension {
  /**
   * onPageLoaded called when a page is loaded.
   */
  onPageLoaded?: (page: PageType) => void;

  /**
   * onInspectionDataOpen called when any data load is completed.
   */
  onInspectionDataOpen?: (
    inspectionData: InspectionData,
    textBufferSource: ReferenceResolverStore,
    rawData: ArrayBuffer,
  ) => void;

  /**
   * onInspectionStart called when user triggered a new inspection.
   */
  onInspectionStart?: () => void;
}

export enum PageType {
  Main = 'MAIN',
  GraphView = 'GRAPH_VIEW',
  DiffView = 'DIFF_VIEW',
}
