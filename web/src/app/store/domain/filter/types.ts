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

import { TimelineStore } from 'src/app/store/domain/timeline-store';

/**
 * Represents the evaluation state passed through the timeline and log filtering pipeline.
 * Contains the intermediate subsets of timeline and log IDs remaining after each filter step.
 */
export interface LogTimelineFilterContext {
  /**
   * Set of remaining timeline IDs that are currently active in the filtered view.
   */
  timelineIds: Set<number>;
  /**
   * Set of remaining log IDs that are currently active in the filtered view.
   */
  logIds: Set<number>;
}

/**
 * Defines the interface for a pipeline filter step that refines the sets of visible timelines and logs.
 */
export interface LogTimelineFilter {
  /**
   * Returns the priority value determining the execution order of this filter.
   * Lower numerical values execute earlier in the processing pipeline.
   */
  priority(): number;
  /**
   * Processes the input filter context and returns an updated context with filtered timeline and log IDs.
   */
  process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
  ): LogTimelineFilterContext;
}
