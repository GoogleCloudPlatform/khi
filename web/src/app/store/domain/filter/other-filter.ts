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

import { Injectable, signal } from '@angular/core';
import { Subject } from 'rxjs';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  CancellationError,
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';

/**
 * Recursively includes all descendant timelines for any matching ancestor and extracts their base log stream.
 */
export class IncludeDescendantsFilter implements LogTimelineFilter {
  readonly displayName = 'Include descendants';
  readonly priority = 20;

  /**
   * Processes the context by expanding matching timeline IDs to include their descendants and extracts their related log IDs.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
  ): Promise<LogTimelineFilterContext> {
    const includedSet = new Set<number>();

    const markDescendants = (t: ReadonlyDomainElement<Timeline>) => {
      if (includedSet.has(t.id)) {
        return;
      }
      includedSet.add(t.id);
      for (const child of t.children()) {
        markDescendants(child);
      }
    };

    for (const id of context.timelineIds) {
      markDescendants(timelineStore.getTimeline(id));
    }

    const passedLogIds = new Set(context.logIds);

    for (const id of includedSet) {
      const t = timelineStore.getTimeline(id);
      for (const rev of t.revisions) {
        passedLogIds.add(rev.log.id);
      }
      for (const evt of t.events) {
        passedLogIds.add(evt.log.id);
      }
    }

    return {
      timelineIds: includedSet,
      logIds: passedLogIds,
    };
  }
}

/**
 * Recursively includes all ancestor timelines for any matching descendant to ensure context paths remain visible.
 */
export class IncludeAncestorsFilter implements LogTimelineFilter {
  readonly displayName = 'Include ancestors';
  readonly priority = Number.MAX_SAFE_INTEGER - 100000;

  /**
   * Processes the context by expanding matching timeline IDs to include their ancestors and extracts their related log IDs.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
  ): Promise<LogTimelineFilterContext> {
    const includedSet = new Set<number>();

    const markAncestors = (t: ReadonlyDomainElement<Timeline>) => {
      if (includedSet.has(t.id)) {
        return;
      }
      includedSet.add(t.id);
      const p = t.parent;
      if (p) {
        markAncestors(p);
      }
    };

    for (const id of context.timelineIds) {
      markAncestors(timelineStore.getTimeline(id));
    }

    return {
      timelineIds: includedSet,
      logIds: context.logIds,
    };
  }
}

/**
 * Removes timelines that contain no matching logs to reduce visual clutter.
 */
@Injectable({ providedIn: 'root' })
export class ExcludeNoLogsFilter implements LogTimelineFilter {
  readonly displayName = 'Exclude no logs';
  readonly priority = 40;

  private readonly _onChanged = new Subject<void>();

  /**
   * Emits whenever the filter's internal configurations or state changes.
   */
  public readonly onChanged = this._onChanged.asObservable();

  private readonly _enabled = signal(false);

  /**
   * Holds whether this filter is actively enabled as a read-only signal.
   */
  public readonly enabled = this._enabled.asReadonly();

  /**
   * Updates the enabled state of this filter and notifies any active subscribers.
   */
  public setEnabled(value: boolean): void {
    this._enabled.set(value);
    this._onChanged.next();
  }

  /**
   * Processes the context by retaining only timeline IDs that possess at least one log present in the current context.
   * If disabled, returns the context as-is.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
    signal?: AbortSignal,
  ): Promise<LogTimelineFilterContext> {
    if (!this.enabled()) {
      return context;
    }
    const passedTimelineIds = new Set<number>();

    for (const id of context.timelineIds) {
      if (signal?.aborted) {
        throw new CancellationError();
      }
      const t = timelineStore.getTimeline(id);
      let hasLogs = false;
      for (const rev of t.revisions) {
        if (context.logIds.has(rev.log.id)) {
          hasLogs = true;
          break;
        }
      }
      if (!hasLogs) {
        for (const evt of t.events) {
          if (context.logIds.has(evt.log.id)) {
            hasLogs = true;
            break;
          }
        }
      }
      if (hasLogs) {
        passedTimelineIds.add(id);
      }
    }

    return {
      timelineIds: passedTimelineIds,
      logIds: context.logIds,
    };
  }
}
