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

import { LogStore } from 'src/app/store/domain/log-store';
import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { StyleStore } from 'src/app/store/domain/style-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';

/**
 * Represents the complete domain model for v6 file format or later.
 *
 * This model is accessed by views after being converted from Protobuf types.
 * It provides efficient memory layouts and high-speed in-memory searches on the frontend,
 * while absorbing differences in the Protobuf side.
 *
 * TODO: Remove "V2" suffix once this completely replaces the existing InspectionData.
 */
export interface InspectionDataV2 {
  /**
   * Interned storage provider providing efficient memory layout.
   */
  readonly internPool: InternPoolStore;

  /**
   * Provides visual style definitions used in the inspection data.
   */
  readonly styleStore: StyleStore;

  /**
   * Log store provides efficient access for log data.
   */
  readonly logStore: LogStore;

  /**
   * Timeline store provides efficient access for timeline data.
   */
  readonly timelineStore: TimelineStore;
}
