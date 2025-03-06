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
  ParentRelationshipMetadata,
  ParentRelationshipMetadataType,
} from '../generated';
import {
  ResourceTimeline,
  TimelineLayer,
  timelineLayerToName,
} from '../store/timeline';

/**
 * The view model of TimelineComponent.
 */
export interface TimelineComponentViewModel {
  /**
   * A timeline list shown on the main area of TimelineComponent.
   */
  scrollableTimelines: TimelineViewModel[];

  /**
   * A timeline list shown stick at the top of the scrollable area of TimelineComponent.
   */
  stickyTimelines: TimelineViewModel[];

  /**
   * A timeline currently directly highlighted by the user interaction.
   */
  highlightedTimeline: TimelineViewModel | null;

  /*
   * The resource path of selected Timeline.
   */
  selectedTimelineResourcePath: string;

  /**
   * The resource path of highlighted(hovered) timeline.
   */
  highlightedTimelineResourcePath: string;

  /**
   * The set of resource paths of timelines highlighted by its selected ancestor.
   */
  highlightedChildrenOfSelectedTimelineResourcePath: Set<string>;
}

/**
 * A timeline contained in TimelineComponentViewModel.
 */
export interface TimelineViewModel {
  /**
   * The source of this vm.
   */
  data: ResourceTimeline;
  /**
   * The resource path representing the FQDN of timeline name.
   */
  resourcePath: string;

  /**
   * The main visible name of a timeline.
   */
  label: string;

  /**
   * The sub visible name of a timeline.
   */
  subLabel: string;

  /**
   * The name of layer where this timeline is placed.
   * (e.g: namespace, kind, name..etc)
   */
  layerName: string;

  /**
   * The metadata annotating the meaning of relationship between this timeline and its parent.
   */
  relationshipMetadata: ParentRelationshipMetadataType;
}

export function convertTimlineEntryToTimelineComponentViewModel(
  timeline: ResourceTimeline,
): TimelineViewModel {
  return {
    data: timeline,
    resourcePath: timeline.resourcePath,
    label: timeline.name,
    subLabel:
      timeline.layer === TimelineLayer.Kind
        ? timeline.resourcePath.split('#')[0]
        : '',
    layerName: timelineLayerToName(timeline.layer),
    relationshipMetadata:
      ParentRelationshipMetadata[timeline.parentRelationship],
  };
}

export function emptyTimelineComponentViewModel(): TimelineComponentViewModel {
  return {
    scrollableTimelines: [] as TimelineViewModel[],
    stickyTimelines: [] as TimelineViewModel[],
    highlightedTimeline: null,
    selectedTimelineResourcePath: '',
    highlightedTimelineResourcePath: '',
    highlightedChildrenOfSelectedTimelineResourcePath: new Set<string>(),
  };
}
