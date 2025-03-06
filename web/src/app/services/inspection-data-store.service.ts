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

import { Injectable } from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  ReplaySubject,
  combineLatest,
  debounceTime,
  filter,
  map,
  shareReplay,
  startWith,
  switchMap,
} from 'rxjs';
import { InspectionData, TimeRange } from '../store/inspection-data';
import { FilterWorkerService } from './filter-worker.service';
import { ParentRelationship } from '../generated';
import { ResourceTimeline } from '../store/timeline';

/**
 * InspectionDataStore provides observable to the inspection data loaded.
 * This store won't provide any filterings performed in response to user's interactions.
 * Implementation of this class must compute values and emit them only when another inspection data was loaded.
 */
export interface InspectionDataStore {
  /**
   * allTimelines emits the array of timeline entry without any filter.
   */
  allTimelines: Observable<ResourceTimeline[]>;

  /**
   * availableKinds emits the set of all kind names found in the inspection data.
   */
  availableKinds: Observable<Set<string>>;

  /**
   * availableNamespaces emits the set of all namespaces found in the inspection data.
   */
  availableNamespaces: Observable<Set<string>>;

  /**
   * availableSubresourceParentRelationships emits the set of all parent relationships of subresources in the inspection data.
   */
  availableSubresourceParentRelationships: Observable<Set<ParentRelationship>>;

  filteredOutLogIndicesSet: Observable<Set<number>>;
}

/**
 * Manage the inspection data(Timelines, Logs,...) after receiving it from somewhere
 * Provides filter feature on this layer
 */
@Injectable({ providedIn: 'root' })
export class InspectionDataStoreService implements InspectionDataStore {
  /**
   * Source of the inspection data
   */
  public inspectionData = new BehaviorSubject<InspectionData | null>(null);

  /**
   * Inspectiondata with null check.
   */
  public currentValidInspectionData = this.inspectionData.pipe(
    filter((d) => !!d),
    shareReplay({
      bufferSize: 1,
      // This reference must be kept when all subscriber removed its subscription because it takes relative longer time to load this data from the source again.
      refCount: false,
    }),
  ) as unknown as Observable<InspectionData>;

  public referenceResolver = this.currentValidInspectionData.pipe(
    map((data) => data.referenceResolver),
  );

  public allTimelines = this.currentValidInspectionData.pipe(
    map((data) => data.timelines),
    startWith([] as ResourceTimeline[]),
  );

  public $timeRange = this.currentValidInspectionData.pipe(
    map((t) => t.range),
    startWith(new TimeRange(0, 0)),
  );
  public availableKinds = this.currentValidInspectionData.pipe(
    map((t) => t.kinds),
    startWith(new Set<string>()),
  );
  public availableNamespaces = this.currentValidInspectionData.pipe(
    map((t) => t.namespaces),
    startWith(new Set<string>()),
  );

  public availableSubresourceParentRelationships =
    this.currentValidInspectionData.pipe(
      map((t) => t.relationships),
      startWith(new Set<ParentRelationship>()),
    );

  public allLogs = this.currentValidInspectionData.pipe(
    map((t) => t.logs),
    startWith([]),
  );
  private logFilter = new ReplaySubject<string>(1);

  /**
   * An observable emits the Set of log indices to be filtered out.
   */
  public filteredOutLogIndicesSet = combineLatest([
    this.allLogs,
    this.logFilter.pipe(startWith('')),
  ]).pipe(
    debounceTime(0),
    switchMap(([allLogs, filter]) =>
      this.filterWorker.filterLogs(allLogs, filter),
    ),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
    startWith(new Set<number>()),
  );

  /**
   * An observable emits list of logs filtered.
   */
  public filteredLogs = combineLatest([
    this.allLogs,
    this.filteredOutLogIndicesSet,
  ]).pipe(
    debounceTime(0),
    map(([allLogs, filteredOutLogs]) =>
      allLogs.filter((_, index) => !filteredOutLogs.has(index)),
    ),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  private filterWorker: FilterWorkerService = new FilterWorkerService(this);

  public setNewInspectionData(data: InspectionData) {
    this.inspectionData.next(data);
  }

  public async setLogRegexFilter(filter: string) {
    this.logFilter.next(filter);
  }
}
