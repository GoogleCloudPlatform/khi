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

import { ReferenceResolverStore } from '../common/loader/reference-resolver';
import { InspectionMetadataHeader } from '../common/schema/metadata-types';
import { ParentRelationship } from '../generated';
import { LogEntry } from './log';
import { ResourceTimeline, TimelineLayer } from './timeline';

export class TimeRange {
  constructor(
    public begin: number,
    public end: number,
  ) {}

  get duration(): number {
    return this.end - this.begin;
  }
}

/**
 * The store of inspection data.
 *
 */
export class InspectionData {
  /**
   * Set of namespace names included in this inspection data.
   */
  public readonly namespaces: Set<string>;

  /**
   * Set of kinds included in this inspection data.
   */
  public readonly kinds: Set<string>;

  /**
   * Set of ParentRelationships used in this inspection data.
   */
  public readonly relationships: Set<ParentRelationship>;

  /**
   * Map to bind a string resource path to a TimelineEntry.
   */
  private readonly resourcePathToTimelineMap: Map<string, ResourceTimeline> =
    new Map();

  constructor(
    public readonly header: InspectionMetadataHeader,
    public readonly range: TimeRange,
    public readonly referenceResolver: ReferenceResolverStore,
    public readonly timelines: ResourceTimeline[],
    public readonly logs: LogEntry[],
  ) {
    this.namespaces = this.getUniqueValuesOfTimelinesOnLayer(
      TimelineLayer.Namespace,
      (t) => t.name,
    );
    this.kinds = this.getUniqueValuesOfTimelinesOnLayer(
      TimelineLayer.Kind,
      (t) => t.name,
    );
    this.relationships = this.getUniqueValuesOfTimelinesOnLayer(
      TimelineLayer.Subresource,
      (t) => t.parentRelationship,
    );
    for (const timeline of this.timelines) {
      this.resourcePathToTimelineMap.set(timeline.resourcePath, timeline);
    }
  }

  /**
   * Generate a Set from a field of timelines at a specific layer.
   */
  private getUniqueValuesOfTimelinesOnLayer<T>(
    layer: TimelineLayer,
    getter: (t: ResourceTimeline) => T,
  ): Set<T> {
    const result = new Set<T>();
    for (const t of this.getTimelinesOfLayer(layer)) {
      result.add(getter(t));
    }
    return result;
  }

  /**
   * Get all the timelines on the specific layer.
   */
  public getTimelinesOfLayer(layer: TimelineLayer): ResourceTimeline[] {
    return this.timelines.filter((t) => t.layer === layer);
  }

  /**
   * Find a TimelineEntry with resource path string. Return null when specified path is not existing in this inspection data.
   */
  public getTimelineByResourcePath(
    resourcePath: string,
  ): ResourceTimeline | null {
    return this.resourcePathToTimelineMap.get(resourcePath) ?? null;
  }

  /**
   * Find list of aliased timelines of the timeline specified with resource path.
   * @param resourcePath The resource path of ResourceTimeline to find aliased ResoruceTimeline
   * @returns The list of aliased timelines of the timeline at the given resource path.
   */
  public getAliasedTimelines(resourcePath: string): ResourceTimeline[] {
    const timeline = this.getTimelineByResourcePath(resourcePath);
    if (!timeline) return [];
    return this.timelines.filter(
      (t) => timeline !== t && t.timelineId === timeline.timelineId,
    );
  }
}
