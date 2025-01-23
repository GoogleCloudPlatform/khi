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

import { InjectionToken } from '@angular/core';
import { InspectionDataStore } from './inspection-data-store.service';
import {
  connectable,
  map,
  merge,
  ReplaySubject,
  shareReplay,
  Subject,
} from 'rxjs';
import { ParentRelationship } from '../generated';
import {
  FilterChain,
  PropertyMatchRegexFilterChainElement,
  PropertyMatchSetFilterChainElement,
} from './filter/chain';
import { TimelineLayer } from '../store/timeline';
import {
  FilterNamepaceOrKindWithoutResource,
  FilterSubresourceTimelinesOnlyWithFilteredLogs,
  FilterSubresourceWithoutParent,
  FilterTimelinesOnlyWithFilteredLogs,
} from './filter/timeline-filter-chain';
import { ViewStateService } from './view-state.service';

/**
 * Injection token for the default TimelineFilter.
 */
export const DEFAULT_TIMELINE_FILTER = new InjectionToken(
  'DEFAULT_TIMELINE_FILTER',
);

/**
 * TimelineFilter provides Observable fields for timelines.
 * It listen changes on inspection data store and filters timelines with given conditions.
 */
export class TimelineFilter {
  constructor(
    public readonly dataStore: InspectionDataStore,
    public readonly viewStateStore: ViewStateService,
  ) {
    this.kindTimelineFilter.connect();
    this.namespaceTimelineFilter.connect();
    this.subresourceParentRelationshipFilter.connect();
    this.resourceNameTimelineRegexFilter.connect();

    this.timelineFilterChain.addFilterElement(
      new PropertyMatchSetFilterChainElement(
        (t) => t.getNameOfLayer(TimelineLayer.Kind),
        this.kindTimelineFilter,
      ),
    );
    this.timelineFilterChain.addFilterElement(
      new PropertyMatchSetFilterChainElement(
        (t) => t.getNameOfLayer(TimelineLayer.Namespace),
        this.namespaceTimelineFilter,
        (t) => t.layer >= TimelineLayer.Namespace,
      ),
    );
    this.timelineFilterChain.addFilterElement(
      new PropertyMatchSetFilterChainElement(
        (t) => t.parentRelationship,
        this.subresourceParentRelationshipFilter,
        (t) => t.layer >= TimelineLayer.Subresource,
      ),
    );
    this.timelineFilterChain.addFilterElement(
      new PropertyMatchRegexFilterChainElement(
        (t) => t.getNameOfLayer(TimelineLayer.Name),
        this.resourceNameTimelineRegexFilterInRegExpList,
        (t) => t.layer >= TimelineLayer.Name,
      ),
    );
    this.timelineFilterChain.addFilterElement(
      new FilterTimelinesOnlyWithFilteredLogs(
        dataStore.filteredOutLogIndicesSet,
        viewStateStore.hideResourcesWithoutMatchingLogs,
      ),
    );
    this.timelineFilterChain.addFilterElement(
      new FilterSubresourceTimelinesOnlyWithFilteredLogs(
        dataStore.filteredOutLogIndicesSet,
        viewStateStore.hideSubresourcesWithoutMatchingLogs,
      ),
    );

    this.timelineFilterChainPostprocess.addFilterElement(
      new FilterNamepaceOrKindWithoutResource(),
    );
    this.timelineFilterChainPostprocess.addFilterElement(
      new FilterSubresourceWithoutParent(),
    );
  }

  /**
   * Observable for currently selected kind name set.
   */
  private readonly kindTimelineFilterSubject = new Subject<Set<string>>();

  /**
   * Observable for currently selected kind name set.
   * This observable also emits the all kind names on the list of available kind names are changed.
   */
  public readonly kindTimelineFilter = connectable(
    merge(this.kindTimelineFilterSubject, this.dataStore.availableKinds),
    {
      connector: () => new ReplaySubject(1),
      resetOnDisconnect: false,
    },
  );

  /**
   * Observable for currently selected namespace set.
   */
  private readonly namespaceTimelineFilterSubject = new Subject<Set<string>>();

  /**
   * Observable for currently selected namespace set.
   * This observable also emits the all namespaces on the list of available namespaces are changed.
   */
  public readonly namespaceTimelineFilter = connectable(
    merge(
      this.namespaceTimelineFilterSubject,
      this.dataStore.availableNamespaces,
    ),
    {
      connector: () => new ReplaySubject(1),
      resetOnDisconnect: false,
    },
  );

  /**
   * Observable for currently selected parent relationship set for subresources.
   */
  private readonly subresourceParentRelationshipFilterSubject = new Subject<
    Set<ParentRelationship>
  >();

  /**
   * Observable for currently selected parent relationships of subresources.
   * This observable also emits the all parent relationships of subresource when the available parent relationships are changed.
   */
  public readonly subresourceParentRelationshipFilter = connectable(
    merge(
      this.subresourceParentRelationshipFilterSubject,
      this.dataStore.availableSubresourceParentRelationships,
    ),
    {
      connector: () => new ReplaySubject(1),
      resetOnDisconnect: false,
    },
  );

  /**
   * Observable for current active regex name filter for resource names.
   */
  private readonly resourceNameTimelineRegexFilterSubject =
    new Subject<string>();

  /**
   * Observable for currently used regex filter for resource name.
   * This observable resets the filter when a new data loaded on the data store.
   */
  public readonly resourceNameTimelineRegexFilter = connectable(
    merge(
      this.resourceNameTimelineRegexFilterSubject,
      this.dataStore.allTimelines.pipe(map(() => '')),
    ),
    {
      connector: () => new ReplaySubject(1),
      resetOnDisconnect: false,
    },
  );

  /**
   * Observable emitting RegExp list parsed from resourceNameTimelineRegexFilter. This allows user to use multiple filter with splitting regex filter with white space.
   */
  readonly resourceNameTimelineRegexFilterInRegExpList =
    this.resourceNameTimelineRegexFilter.pipe(
      map((regex) =>
        regex
          .split(' ')
          .filter((f) => f != '')
          .map((filter) => new RegExp(filter)),
      ), // convert to the list of RegExp[]
      map((regexps) => (regexps.length == 0 ? [/.*/] : regexps)), // return a regex matching anything when no filter provided.
    );

  private readonly timelineFilterChain = new FilterChain(
    this.dataStore.allTimelines,
  );

  private readonly timelineFilterChainPostprocess = new FilterChain(
    this.timelineFilterChain.filtered,
  );

  /**
   * The list of timelines filtered by this TimelineFilter.
   */
  public readonly filteredTimeline =
    this.timelineFilterChainPostprocess.filtered.pipe(
      shareReplay({ bufferSize: 1, refCount: false }),
    );

  /**
   * setKindFilter limits the filteredTimeline by kind names.
   */
  public setKindFilter(kinds: Set<string>) {
    this.kindTimelineFilterSubject.next(kinds);
  }

  /**
   * setNamespaceFilter limits the filteredTimeline by namespace names.
   */
  public setNamespaceFilter(namespaces: Set<string>) {
    this.namespaceTimelineFilterSubject.next(namespaces);
  }

  /**
   * setSubresourceParentRelationshipFilter limits the filteredTimeline of subresources by its relationship.
   */
  public setSubresourceParentRelationshipFilter(
    relationships: Set<ParentRelationship>,
  ) {
    this.subresourceParentRelationshipFilterSubject.next(relationships);
  }

  /**
   * setResourceNameRegexFilter limits the filteredTimeline of resources by its name.
   */
  public setResourceNameRegexFilter(regexFilter: string) {
    this.resourceNameTimelineRegexFilterSubject.next(regexFilter);
  }
}
