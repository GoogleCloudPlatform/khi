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

import { ParentRelationship } from '../generated';
import { ResourceEvent } from './event';
import { ResourceRevision } from './revision';

/**
 * ResourceTimeline is a model representing a timeline associated with a specific resorce path.
 *
 * Note: KHI can have multiple resources bound to a timeline. For example, the timeline to show owner reference can have the aliased timeline.
 */
export class ResourceTimeline {
  private readonly resourcePathFragments: string[] = [];

  private _privateParent: ResourceTimeline | null = null;
  /**
   * Get the parent of this timeline.
   * Returns null when this timeline is at the root layer.
   */
  public get parent(): ResourceTimeline | null {
    return this._privateParent;
  }

  public readonly children: ResourceTimeline[] = [];

  /**
   * @param timelineId The ID of timeline. There can be several ResourceTimelines shareing same timelineId but with different resource path.
   * @param resourcePath The resource path representing the location of the timeline. Example: core.v1/pod#kube-system#nginx
   * @param revisions The list of ResourceRevisions contained in this timeline.
   * @param events The list of ResourceEvents contained in this timeline.
   * @param parentRelationship A type representing the relationship between this timeline and this parent.
   */
  constructor(
    public readonly timelineId: string,
    public readonly resourcePath: string,
    public readonly revisions: ResourceRevision[],
    public readonly events: ResourceEvent[],
    public readonly parentRelationship: ParentRelationship,
  ) {
    this.resourcePathFragments = this.resourcePath.split('#');
  }

  /**
   * Add a timeline as a children.
   */
  public addChildTimeline(timeline: ResourceTimeline) {
    timeline._privateParent = this;
    this.children.push(timeline);
  }

  /**
   * Return the layer of this Timeline.
   * See the TimelineLayer enum more detail about the layer.
   */
  public get layer(): TimelineLayer {
    return this.resourcePathFragments.length - 1;
  }

  /**
   * Returns the name of current layer.
   * This is equivalent to timeline.getNameOfLayer(timeline.layer)
   */
  public get name(): string {
    return this.getNameOfLayer(this.layer);
  }

  /**
   * Return the name at the specified layer.
   * For example, this will return namespace name by calling getNameOfLayer with TimelineLayer.Namespace on the timeline for a Pod.
   * @param layer The layer of timeline to get the name.
   * @returns the name of speicfied layer. Returns an empty string when the layer is not defined for this timeline.
   */
  public getNameOfLayer(layer: TimelineLayer) {
    return this.resourcePathFragments[layer] ?? '';
  }

  /**
   * Get the list of events contained in the specified range.
   * The range is regarded as [beginTime,endTime). The log at the endTime won't be included in the result.
   */
  public queryEventsInRange(
    beginTime: number,
    endTime: number,
  ): ResourceEvent[] {
    const result = [] as ResourceEvent[];
    for (const event of this.events) {
      if (event.ts >= beginTime && event.ts < endTime) {
        result.push(event);
      }
    }
    return result;
  }

  /**
   * Try to get an event from the range and returns an event that is the nearest to the center of the range.
   */
  public pickEventNearCenterOfRange(
    beginTime: number,
    endTime: number,
  ): ResourceEvent | null {
    const inRange = this.queryEventsInRange(beginTime, endTime);
    if (!inRange.length) {
      return null;
    }
    const center = (beginTime + endTime) / 2;
    let nearest = inRange[0];
    let nearestDistance = Number.MAX_VALUE;
    for (const event of inRange) {
      const timeDistance = Math.abs(event.ts - center);
      if (timeDistance < nearestDistance) {
        nearest = event;
        nearestDistance = timeDistance;
      }
    }
    return nearest;
  }

  /**
   * Get the list of revision contained in the specified range.
   * The range is regarded as [beginTime,endTime). The log at the endTime won't be included in the result.
   */
  public queryRevisionsInRange(
    beginTime: number,
    endTime: number,
  ): ResourceRevision[] {
    const result = [] as ResourceRevision[];
    for (const revision of this.revisions) {
      const startEdgeIncludedInRange =
        revision.startAt >= beginTime && revision.startAt < endTime;
      const endEdgeIncludedInRange =
        revision.endAt > beginTime && revision.endAt < endTime;
      const rangeIncludedInRevision =
        revision.startAt < beginTime && revision.endAt > endTime;
      if (
        startEdgeIncludedInRange ||
        endEdgeIncludedInRange ||
        rangeIncludedInRevision
      ) {
        result.push(revision);
      }
    }
    return result;
  }

  /**
   * Return the latest revision of a given time.
   * @param time
   * @returns
   */
  public getLatestRevisionOfTime(time: number): ResourceRevision | null {
    let result: ResourceRevision | null = null;
    for (const revision of this.revisions) {
      if (revision.startAt <= time) {
        result = revision;
      }
      if (revision.endAt > time) {
        break;
      }
    }
    return result;
  }

  /**
   * Get a pair of revisions from given log index.
   * @param logIndex
   * @returns
   */
  public getRevisionPairByLogId(
    logIndex: number,
  ): ResourceRevisionChangePair | null {
    if (this.revisions.length === 0) return null;
    // Find the revision from back, because there could be inferred revision element associating to the same logIndex.
    for (let i = this.revisions.length - 1; i >= 0; i--) {
      if (this.revisions[i].logIndex === logIndex) {
        if (i !== 0)
          return new ResourceRevisionChangePair(
            this.revisions[i - 1],
            this.revisions[i],
          );
        return new ResourceRevisionChangePair(null, this.revisions[i]);
      }
    }
    console.warn(
      `Attempted to find the logIndex ${logIndex} from timeline ${this.resourcePath}, but not found.`,
    );
    return null;
  }

  /**
   * Get the list of TimelineEntry having this TimelineEntry as an ancestor.
   */
  public getAllChildrenRecursive(): ResourceTimeline[] {
    return [
      ...this.children,
      ...this.children.reduce((p, t) => {
        p.push(...t.getAllChildrenRecursive());
        return p;
      }, [] as ResourceTimeline[]),
    ];
  }

  /**
   * Returns if this TimelineEntry is related to any of logs in the given set.
   */
  public hasNonFilteredOutIndices(filteredOut: Set<number>): boolean {
    for (const revision of this.revisions) {
      if (revision.logIndex === -1) continue;
      if (!filteredOut.has(revision.logIndex)) return true;
    }
    for (const event of this.events) {
      if (event.logIndex === -1) continue;
      if (!filteredOut.has(event.logIndex)) return true;
    }
    return false;
  }

  /**
   * Returns if this TimelineEntry or any of its children is related to any of logs in the given set.
   */
  public hasNonFilteredOutIndicesRecursive(filtereOut: Set<number>): boolean {
    if (this.hasNonFilteredOutIndices(filtereOut)) return true;
    for (const child of this.children) {
      if (child.hasNonFilteredOutIndicesRecursive(filtereOut)) return true;
    }
    return false;
  }

  public static clone(timeline: ResourceTimeline): ResourceTimeline {
    const p = new ResourceTimeline(
      timeline.timelineId,
      timeline.resourcePath,
      timeline.revisions.map((r) => ResourceRevision.clone(r)),
      timeline.events.map((e) => ResourceEvent.clone(e)),
      timeline.parentRelationship,
    );
    timeline.children
      .map((t) => ResourceTimeline.clone(t))
      .forEach((child) => p.addChildTimeline(child));
    return p;
  }
}

/**
 * Enum for layer of a timeline.
 * Timelines are hierarchical structure, it has k8s specific names on each depths.
 */
export enum TimelineLayer {
  APIVersion = 0,
  Kind = 1,
  Namespace = 2,
  Name = 3,
  Subresource = 4,
}

/**
 * Convert the TimelineLayer to the layer specific name.
 * @param layer the TimelineLayer to convert into string.
 * @returns String representation of the TimelineLayer.
 */
export function timelineLayerToName(layer: TimelineLayer): string {
  switch (layer) {
    case TimelineLayer.APIVersion:
      return 'apiVersion';
    case TimelineLayer.Kind:
      return 'kind';
    case TimelineLayer.Namespace:
      return 'namespace';
    case TimelineLayer.Name:
      return 'name';
    case TimelineLayer.Subresource:
      return 'subresource';
  }
}

/**
 * Represents a pair of adjacent ResourceRevisions.
 */
export class ResourceRevisionChangePair {
  constructor(
    public readonly previous: ResourceRevision | null,
    public readonly current: ResourceRevision,
  ) {}
}
