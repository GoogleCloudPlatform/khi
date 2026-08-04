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

import { Subject } from 'rxjs';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { Timeline } from 'src/app/store/domain/timeline';
import {
  LogTimelineFilter,
  LogTimelineFilterContext,
} from 'src/app/store/domain/filter/types';

/**
 * Filter step that excludes descendant timelines of collapsed parent timelines from the visible view.
 */
export class CollapseTimelineFilter implements LogTimelineFilter {
  public readonly displayName = 'CollapseTimelineFilter';
  public readonly priority = Number.MAX_SAFE_INTEGER - 50000;

  private readonly _onChanged = new Subject<void>();
  public readonly onChanged = this._onChanged.asObservable();

  private _collapsedTimelineIds = new Set<number>();

  /**
   * Gets the set of currently collapsed timeline IDs.
   */
  get collapsedTimelineIds(): ReadonlySet<number> {
    return this._collapsedTimelineIds;
  }

  /**
   * Sets the collapsed timeline IDs and triggers filter pipeline re-evaluation.
   * @param ids - The new set of collapsed timeline IDs.
   */
  public setCollapsedTimelineIds(ids: ReadonlySet<number>): void {
    this._collapsedTimelineIds = new Set(ids);
    this._onChanged.next();
  }

  /**
   * Clears all collapsed timeline IDs to expand all timelines.
   */
  public expandAll(): void {
    this.setCollapsedTimelineIds(new Set());
  }

  /**
   * Expands direct children timelines of a parent timeline.
   * @param parentTimeline - The parent timeline whose direct children will be expanded.
   */
  public expandChildren(parentTimeline: Timeline): void {
    const next = new Set(this._collapsedTimelineIds);
    for (const child of parentTimeline.children()) {
      next.delete(child.id);
    }
    this.setCollapsedTimelineIds(next);
  }

  /**
   * Collapses direct children timelines of a parent timeline.
   * @param parentTimeline - The parent timeline whose direct children will be collapsed.
   */
  public collapseChildren(parentTimeline: Timeline): void {
    const next = new Set(this._collapsedTimelineIds);
    for (const child of parentTimeline.children()) {
      if (child.childrenCount > 0) {
        next.add(child.id);
      }
    }
    this.setCollapsedTimelineIds(next);
  }

  /**
   * Expands a parent timeline and all of its descendants recursively.
   * @param parentTimeline - The parent timeline to expand recursively.
   */
  public expandDescendants(parentTimeline: Timeline): void {
    const next = new Set(this._collapsedTimelineIds);
    next.delete(parentTimeline.id);
    for (const desc of parentTimeline.descendants()) {
      next.delete(desc.id);
    }
    this.setCollapsedTimelineIds(next);
  }

  /**
   * Collapses a parent timeline and all of its descendants recursively.
   * @param parentTimeline - The parent timeline to collapse recursively.
   */
  public collapseDescendants(parentTimeline: Timeline): void {
    const next = new Set(this._collapsedTimelineIds);
    next.add(parentTimeline.id);
    for (const desc of parentTimeline.descendants()) {
      if (desc.childrenCount > 0) {
        next.add(desc.id);
      }
    }
    this.setCollapsedTimelineIds(next);
  }

  /**
   * Collapses a specific timeline.
   * @param timelineId - The ID of the timeline to collapse.
   */
  public collapseTimeline(timelineId: number): void {
    const next = new Set(this._collapsedTimelineIds);
    next.add(timelineId);
    this.setCollapsedTimelineIds(next);
  }

  /**
   * Toggles the collapse state of a specific timeline.
   * @param timelineId - The ID of the timeline to toggle.
   */
  public toggleTimelineCollapse(timelineId: number): void {
    const next = new Set(this._collapsedTimelineIds);
    if (next.has(timelineId)) {
      next.delete(timelineId);
    } else {
      next.add(timelineId);
    }
    this.setCollapsedTimelineIds(next);
  }

  /**
   * Processes the filter context by removing all descendant timeline IDs of collapsed timelines.
   */
  public async process(
    context: LogTimelineFilterContext,
    timelineStore: TimelineStore,
  ): Promise<LogTimelineFilterContext> {
    if (this._collapsedTimelineIds.size === 0) {
      return context;
    }

    const nextTimelineIds = new Set(context.timelineIds);

    for (const collapsedId of this._collapsedTimelineIds) {
      if (!nextTimelineIds.has(collapsedId)) {
        continue;
      }
      try {
        const parentTimeline = timelineStore.getTimeline(collapsedId);
        for (const descendant of parentTimeline.descendants()) {
          nextTimelineIds.delete(descendant.id);
        }
      } catch {
        // Ignore if timeline is not found
      }
    }

    return {
      timelineIds: nextTimelineIds,
      logIds: context.logIds,
    };
  }
}
