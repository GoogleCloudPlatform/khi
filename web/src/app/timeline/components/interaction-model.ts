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

/**
 * Defines the highlighting state of a timeline layer.
 */
export enum TimelineHighlightType {
  /**
   * No highlighting.
   */
  None = 0,
  /**
   * The timeline is selected by the user.
   */
  Selected = 1,
  /**
   * The timeline is currently being hovered over.
   */
  Hovered = 2,
  /**
   * The timeline is a child of the currently selected timeline.
   */
  ChildrenOfSelected = 3,
}

/**
 * A map associating timeline IDs with their current highlighting state.
 * TimelineHighlightType.None is used for undefined values.
 */
export type TimelineHighlight = { [timelineId: string]: TimelineHighlightType };

/**
 * Defines the highlighting state of an individual item (event or revision) on the chart.
 */
export enum TimelineChartItemHighlightType {
  /**
   * No highlighting.
   */
  None = 0,
  /**
   * The item is selected.
   */
  Selected = 2,
  /**
   * The item is being hovered over.
   */
  Hovered = 1,
}

/**
 * A map associating log indices with their highlighting state on the chart.
 * TimelineChartItemHighlightType.None is used for undefined values.
 */
export type TimelineChartItemHighlight = {
  [logIndex: number]: TimelineChartItemHighlightType;
};
