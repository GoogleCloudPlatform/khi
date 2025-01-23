/**
 * Copyright 2025 Google LLC
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

import { TimelineEntry, TimelineLayer } from 'src/app/store/timeline';
import { FilterChainElement } from './chain';
import { combineLatestWith, map, Observable } from 'rxjs';

/**
 * FilterNamepaceOrKindWithoutResource removes TimelineEntry without any name layer timeline in its children.
 */
export class FilterNamepaceOrKindWithoutResource
  implements FilterChainElement<TimelineEntry>
{
  chain(items: Observable<TimelineEntry[]>): Observable<TimelineEntry[]> {
    return items.pipe(
      map((timelines) => {
        let lastProcessedKind: TimelineEntry | null = null;
        let lastProcessedNamespace: TimelineEntry | null = null;
        const filteredTimelines: TimelineEntry[] = [];
        for (const timeline of timelines) {
          if (timeline.layer === TimelineLayer.Kind) {
            lastProcessedKind = timeline;
            continue;
          } else if (timeline.layer === TimelineLayer.Namespace) {
            lastProcessedNamespace = timeline;
            continue;
          } else if (timeline.layer === TimelineLayer.Name) {
            if (lastProcessedNamespace) {
              if (lastProcessedKind) {
                filteredTimelines.push(lastProcessedKind);
                lastProcessedKind = null;
              }
              filteredTimelines.push(lastProcessedNamespace);
              lastProcessedNamespace = null;
            }
          }
          filteredTimelines.push(timeline);
        }
        return filteredTimelines;
      }),
    );
  }
}

/**
 * FilterSubresourceWithoutParent removes subresource timelines when its parent resource is filtered out already.
 */
export class FilterSubresourceWithoutParent
  implements FilterChainElement<TimelineEntry>
{
  chain(items: Observable<TimelineEntry[]>): Observable<TimelineEntry[]> {
    return items.pipe(
      map((timelines) => {
        let lastProcessedResourceName: string = '';
        const filteredTimelines: TimelineEntry[] = [];
        for (const timeline of timelines) {
          if (timeline.layer === TimelineLayer.Name) {
            lastProcessedResourceName = timeline.getNameOfLayer(
              TimelineLayer.Name,
            );
          } else if (timeline.layer === TimelineLayer.Subresource) {
            if (
              timeline.getNameOfLayer(TimelineLayer.Name) !==
              lastProcessedResourceName
            ) {
              continue;
            }
          }
          filteredTimelines.push(timeline);
        }
        return filteredTimelines;
      }),
    );
  }
}

/**
 * FilterTimelinesOnlyWithFilteredLogs removes timelines that don't have any logs that are not filtered out.
 */
export class FilterTimelinesOnlyWithFilteredLogs
  implements FilterChainElement<TimelineEntry>
{
  constructor(
    private readonly filteredOut: Observable<Set<number>>,
    private readonly hideFilteredOut: Observable<boolean>,
  ) {}

  chain(items: Observable<TimelineEntry[]>): Observable<TimelineEntry[]> {
    return items.pipe(
      combineLatestWith(this.filteredOut, this.hideFilteredOut),
      map(([timelines, logIndices, hideFilteredOut]) =>
        timelines.filter(
          (t) =>
            !hideFilteredOut ||
            t.layer !== TimelineLayer.Name ||
            logIndices.size === 0 ||
            t.hasNonFilteredOutIndicesRecursive(logIndices),
        ),
      ),
    );
  }
}

/**
 * FilterTimelinesOnlyWithFilteredLogs removes subresource layer timelines that don't have any logs that are not filtered out.
 */
export class FilterSubresourceTimelinesOnlyWithFilteredLogs
  implements FilterChainElement<TimelineEntry>
{
  constructor(
    private readonly filteredOut: Observable<Set<number>>,
    private readonly hideFilteredOut: Observable<boolean>,
  ) {}

  chain(items: Observable<TimelineEntry[]>): Observable<TimelineEntry[]> {
    return items.pipe(
      combineLatestWith(this.filteredOut, this.hideFilteredOut),
      map(([timelines, logIndices, hideFilteredOut]) =>
        timelines.filter(
          (t) =>
            !hideFilteredOut ||
            t.layer !== TimelineLayer.Subresource ||
            logIndices.size === 0 ||
            t.hasNonFilteredOutIndicesRecursive(logIndices),
        ),
      ),
    );
  }
}
