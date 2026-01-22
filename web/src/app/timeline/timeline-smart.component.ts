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

import { Component, computed, inject, signal } from '@angular/core';
import {
  TimelineFrameComponent,
  TimelineHoverOverlayRequest,
} from './components/timeline-frame.component';
import { DEFAULT_TIMELINE_FILTER } from '../services/timeline-filter.service';
import { toSignal } from '@angular/core/rxjs-interop';
import { ViewStateService } from '../services/view-state.service';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { ResourceTimeline } from '../store/timeline';
import { TimelineChartMouseEvent } from './components/timeline-chart.component';
import { SelectionManagerService } from '../services/selection-manager.service';
import {
  TimelineChartItemHighlight,
  TimelineChartItemHighlightType,
  TimelineHighlight,
  TimelineHighlightType,
} from './components/interaction-model';

/**
 * Smart component for the timeline view.
 *
 * It connects the presentational components (TimelineFrame, TimelineCornerIndicator, etc.)
 * with the application state (InspectionDataStoreService, SelectionManagerService, ViewStateService).
 *
 * It is responsible for:
 * - Providing data to the timeline frame (logs, timelines, highlights).
 * - Handling user interaction events raised from presentational components.
 */
@Component({
  selector: 'khi-timeline-smart',
  standalone: true,
  imports: [TimelineFrameComponent],
  templateUrl: './timeline-smart.component.html',
})
export class TimelineSmartComponent {
  private readonly HOVER_VIEW_SELECTABLE_RANGE_IN_PX = 300;
  private readonly MAX_HOVER_VIEW_LOG_COUNT = 20;
  private readonly timelineFilter = inject(DEFAULT_TIMELINE_FILTER);

  private readonly viewStateService = inject(ViewStateService);

  private readonly inspectionDataService = inject(InspectionDataStoreService);

  private readonly selectionManager = inject(SelectionManagerService);

  private readonly inspectionData = toSignal(
    this.inspectionDataService.inspectionData,
    { initialValue: null },
  );

  /**
   * List of timelines to be displayed, filtered by the current filter settings.
   */
  protected readonly filteredTimelines = toSignal(
    this.timelineFilter.filteredTimeline,
    { initialValue: [] },
  );

  /**
   * Current horizontal zoom level (pixels per millisecond).
   */
  protected readonly pixelsPerMs = toSignal(
    this.viewStateService.pixelPerTime,
    { initialValue: 0.0001 },
  );

  /**
   * The start time of the inspection data range.
   * Used to determine the minimum scrollable/viewable time.
   */
  protected readonly minQueryLogTimeMS = computed(() => {
    const store = this.inspectionData();
    if (!store) {
      return Date.now() - 60 * 60 * 1000;
    }
    return store.range.begin; // Any value is fine but to draw empty timeline when no data is available
  });

  /**
   * The end time of the inspection data range.
   * Used to determine the maximum scrollable/viewable time.
   */
  protected readonly maxQueryLogTimeMS = computed(() => {
    const store = this.inspectionData();
    if (!store) {
      return Date.now(); // Any value is fine but to draw empty timeline when no data is available
    }
    return store.range.end;
  });

  /**
   * The current time at the left edge of the viewport.
   */
  protected readonly viewportLeftTimeMs = toSignal(
    this.viewStateService.timeOffset,
    { initialValue: 0 },
  );

  /**
   * The timezone offset in hours to be applied to the displayed time.
   */
  protected readonly timezoneShiftHours = toSignal(
    this.viewStateService.timezoneShift,
    { initialValue: 0 },
  );

  private readonly highlightedTimeline = toSignal(
    this.selectionManager.highlightedTimeline,
  );

  private readonly selectedTimeline = toSignal(
    this.selectionManager.selectedTimeline,
  );

  private readonly highlightedRevisionsOnCurrentTimeline = toSignal(
    this.selectionManager.highlightedChildrenOfSelectedTimeline,
  );

  /**
   * Map of timeline IDs to their highlight state (Selected, Hovered, ChildrenOfSelected).
   * Used to visually emphasize timelines in the ruler and chart.
   */
  protected readonly timelineHighlights = computed(() => {
    const result: TimelineHighlight = {};
    const childrenOfSelected = this.highlightedRevisionsOnCurrentTimeline();
    if (childrenOfSelected) {
      childrenOfSelected.forEach(
        (timeline) =>
          (result[timeline.timelineId] =
            TimelineHighlightType.ChildrenOfSelected),
      );
    }
    const highlighted = this.highlightedTimeline();
    if (highlighted) {
      result[highlighted.timelineId] = TimelineHighlightType.Hovered;
    }
    const timeline = this.selectedTimeline();
    if (timeline) {
      result[timeline.timelineId] = TimelineHighlightType.Selected;
    }
    return result;
  });

  /**
   * List of all logs in the inspection data.
   */
  protected readonly allLogs = toSignal(this.inspectionDataService.allLogs, {
    initialValue: [],
  });

  /**
   * List of logs matching the current filter criteria.
   * Used for the histogram and log distribution views.
   */
  protected readonly filteredLogs = toSignal(
    this.inspectionDataService.filteredLogs,
    {
      initialValue: [],
    },
  );

  private readonly selectedLogIndex = toSignal(
    this.selectionManager.selectedLogIndex,
  );

  private readonly highlightedLogIndices = toSignal(
    this.selectionManager.highlightLogIndices,
  );

  /**
   * Map of log indices to their highlight state (Selected, Hovered) on the chart.
   */
  protected readonly timelineChartItemHighlights = computed(() => {
    const selectedLogIndex = this.selectedLogIndex();
    const highlightedLogIndices = this.highlightedLogIndices();

    const result: TimelineChartItemHighlight = {};
    if (highlightedLogIndices) {
      highlightedLogIndices.forEach(
        (logIndex) =>
          (result[logIndex] = TimelineChartItemHighlightType.Hovered),
      );
    }
    if (selectedLogIndex !== undefined) {
      result[selectedLogIndex] = TimelineChartItemHighlightType.Selected;
    }

    return result;
  });

  private readonly selectedLog = toSignal(this.selectionManager.selectedLog);
  /**
   * The time of the currently selected log.
   * Used to display a vertical cursor line on the timeline.
   */
  protected readonly cursorTimeMs = computed(() => {
    const log = this.selectedLog();
    if (!log) {
      return 0;
    }
    return log.time;
  });

  private lastClickedTimeMs = signal(0);
  /**
   * Data required to render the hover overlay (tooltip) when hovering over the timeline.
   * Calculates specific events or revisions near the hovered time.
   */
  protected readonly timelineHoverOverlayRequest =
    computed<TimelineHoverOverlayRequest | null>(() => {
      const timeline = this.highlightedTimeline();
      if (!timeline) {
        return null;
      }
      const lastClickedTimeMs = this.lastClickedTimeMs();

      const maxT = this.HOVER_VIEW_SELECTABLE_RANGE_IN_PX / this.pixelsPerMs();
      const maxC = this.MAX_HOVER_VIEW_LOG_COUNT;
      const optimalT = this.calculateOptimalQueryPeriod(
        timeline,
        lastClickedTimeMs,
        maxT,
        maxC,
      );

      const events = timeline.queryEventsInRange(
        lastClickedTimeMs - optimalT,
        lastClickedTimeMs + optimalT,
      );
      const revisions = timeline.queryRevisionsInRange(
        lastClickedTimeMs - optimalT,
        lastClickedTimeMs + optimalT,
      );

      return {
        timelineId: timeline.timelineId,
        timeMs: lastClickedTimeMs,
        overlay: {
          timeline: timeline,
          revisions: revisions,
          events: events,
        },
      } as TimelineHoverOverlayRequest;
    });

  /**
   * Handles changes to the zoom level (pixels per millisecond).
   * Updates the global view state.
   */
  protected onPixelsPerMsChange(pixelsPerMs: number): void {
    this.viewStateService.setPixelPerTime(pixelsPerMs);
  }

  /**
   * Handles changes to the viewport's left time (scrolling).
   * Updates the global view state.
   */
  protected onViewportLeftTimeMsChange(viewportLeftTimeMs: number): void {
    this.viewStateService.setTimeOffset(viewportLeftTimeMs);
  }

  /**
   * Handles hovering over a timeline ruler item.
   * Updates the selection manager to highlight the timeline.
   */
  protected hoverOnTimeline(event: ResourceTimeline): void {
    this.selectionManager.onHighlightTimeline(event);
  }

  /**
   * Handles clicking on a timeline ruler item.
   * Updates the selection manager to select the timeline.
   */
  protected clickOnTimeline(event: ResourceTimeline): void {
    this.selectionManager.onSelectTimeline(event);
  }

  /**
   * Handles hovering over an item (event or revision) on the timeline chart.
   * Updates highlights for the timeline and the specific log.
   */
  protected hoverOnTimelineItem(event: TimelineChartMouseEvent): void {
    this.selectionManager.onHighlightTimeline(event.timeline);
    if (event.timeline === null) {
      this.selectionManager.onHighlightLog([]);
    } else {
      if (event.revisionIndex !== undefined) {
        this.selectionManager.onHighlightLog([
          event.timeline.revisions[event.revisionIndex].logIndex,
        ]);
        this.lastClickedTimeMs.set(event.timeMS);
      } else if (event.eventIndex !== undefined) {
        this.selectionManager.onHighlightLog([
          event.timeline.events[event.eventIndex].logIndex,
        ]);
        this.lastClickedTimeMs.set(event.timeMS);
      } else {
        this.selectionManager.onHighlightLog([]);
      }
    }
  }

  /**
   * Handles clicking on an item (event or revision) on the timeline chart.
   * Updates selection for the timeline and the specific log/revision/event.
   */
  protected clickOnTimelineItem(event: TimelineChartMouseEvent): void {
    this.selectionManager.onSelectTimeline(event.timeline);
    if (event.timeline !== null) {
      if (event.revisionIndex !== undefined) {
        this.selectionManager.changeSelectionByRevision(
          event.timeline,
          event.timeline.revisions[event.revisionIndex],
        );
      } else if (event.eventIndex !== undefined) {
        this.selectionManager.changeSelectionByEvent(
          event.timeline,
          event.timeline.events[event.eventIndex],
        );
      }
    }
  }

  /**
   * Calculates the optimal query period for the hover overlay. It returns the maximum time range that doesn't exceed the maximum number of events.
   * @param timeline The timeline to query.
   * @param centerTimeMs The center time of the query.
   * @param maxT The maximum time range for the query.
   * @param maxC The maximum number of events to query.
   * @returns The optimal query period.
   */
  private calculateOptimalQueryPeriod(
    timeline: ResourceTimeline,
    centerTimeMs: number,
    maxT: number,
    maxC: number,
  ): number {
    let low = 0;
    let high = maxT;
    let optimalT = 0;

    while (low <= high) {
      const mid = Math.floor((low + high) / 2);

      const events = timeline.queryEventsInRange(
        centerTimeMs - mid,
        centerTimeMs + mid,
      );
      const revisions = timeline.queryRevisionsInRange(
        centerTimeMs - mid,
        centerTimeMs + mid,
      );
      const totalCount = events.length + revisions.length;

      if (totalCount <= maxC) {
        optimalT = mid;
        low = mid + 1;
      } else {
        high = mid - 1;
      }
    }

    return optimalT;
  }
}
