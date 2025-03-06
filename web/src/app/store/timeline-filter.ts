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

import { ResourceTimeline, TimelineLayer } from './timeline';

export const API_VERSION_CORE = 'core/v1';

export const KIND_NODE = 'node';
export const KIND_POD = 'pod';

export const SUBRESOURCE_BINDING = 'binding';
/**
 * TimelineFilterFacade has static methods to check if the specified timeline is the timeline kind or not.
 */
export class TimelineFilterFacade {
  /**
   * Returns true if the timeline belongs to a node or a descendant of that node.
   */
  public static isNodeOrNodeChildren(timeline: ResourceTimeline): boolean {
    return (
      timeline.getNameOfLayer(TimelineLayer.APIVersion) === API_VERSION_CORE &&
      timeline.getNameOfLayer(TimelineLayer.Kind) === KIND_NODE
    );
  }

  /**
   * Returns true if the timeline belongs to a node or a descendant of that node.
   */
  public static isPodOrPodChildren(timeline: ResourceTimeline): boolean {
    return (
      timeline.getNameOfLayer(TimelineLayer.APIVersion) === API_VERSION_CORE &&
      timeline.getNameOfLayer(TimelineLayer.Kind) === KIND_POD
    );
  }

  /**
   * Returns true if the timeline belongs to a binding subresource of a Pod or a descendant of that binding.
   */
  public static isPodBindingForNode(timeline: ResourceTimeline): boolean {
    return (
      timeline.getNameOfLayer(TimelineLayer.APIVersion) === API_VERSION_CORE &&
      timeline.getNameOfLayer(TimelineLayer.Kind) === KIND_POD &&
      timeline.getNameOfLayer(TimelineLayer.Subresource) === SUBRESOURCE_BINDING
    );
  }
}
