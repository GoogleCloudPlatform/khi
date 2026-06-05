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
import { toSignal } from '@angular/core/rxjs-interop';
import {
  TimelineFrameComponent,
  TimelineHoverOverlayRequest,
} from 'src/app/timeline/components/timeline-frame.component';
import { ViewStateService } from 'src/app/services/view-state.service';
import { InspectionDataStoreV2 } from 'src/app/services/inspection-data-store-v2.service';
import { SelectionManagerV2 } from 'src/app/services/selection-manager-v2.service';
import {
  TimelineChartItemHighlight,
  TimelineChartItemHighlightType,
  TimelineHighlight,
  TimelineHighlightType,
} from 'src/app/timeline/components/interaction-model';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { InspectionDataV2 } from 'src/app/store/domain/inspection-data';
import { StyleStore } from 'src/app/store/domain/style-store';
import {
  TimelineChartStyle,
  TimelineRulerStyle,
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model-v2';
import { TimelineChartMouseEvent } from 'src/app/timeline/components/timeline-chart.component';

/**
 * Smart component for the timeline view.
 *
 * It connects the presentational components (TimelineFrame, TimelineCornerIndicator, etc.)
 * with the V2 application state (InspectionDataStoreV2, SelectionManagerV2, ViewStateService).
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

  private readonly viewStateService = inject(ViewStateService);

  private readonly inspectionDataStore = inject(InspectionDataStoreV2);

  private readonly selectionManager = inject(SelectionManagerV2);

  private readonly inspectionData = computed(() => {
    return this.inspectionDataStore.inspectionData();
  });

  /**
   * List of timelines to be displayed, filtered by the current filter settings.
   */
  protected readonly filteredTimelines = computed(() => {
    return this.inspectionDataStore.timelineView()?.filteredTimelines() ?? [];
  });

  protected readonly pixelsPerMs = toSignal(
    this.viewStateService.pixelPerTime,
    { initialValue: 0.01 },
  );

  private readonly inspectionDataUniqueIDs = new WeakMap<
    InspectionDataV2,
    string
  >();

  /**
   * The unique ID of the inspection data.
   * This is used to detect when the inspection data has changed to refresh timeline renderer cache.
   */
  protected readonly inspectionDataUniqueID = computed(() => {
    const data = this.inspectionData();
    if (!data) {
      return '';
    }
    let id = this.inspectionDataUniqueIDs.get(data);
    if (!id) {
      id = Math.random().toString(36).substring(2, 15);
      this.inspectionDataUniqueIDs.set(data, id);
    }
    return id;
  });

  /**
   * The StyleStore containing all color and layout styling definitions.
   */
  protected readonly styleStore = computed(() => {
    return this.inspectionData()?.styleStore ?? new StyleStore();
  });

  /**
   * Style configuration for the timeline chart area.
   */
  protected readonly chartStyle = computed<TimelineChartStyle>(() => {
    return generateDefaultChartStyle();
  });

  /**
   * Style configuration for the timeline ruler.
   */
  protected readonly rulerStyle = computed<TimelineRulerStyle>(() => {
    return generateDefaultRulerStyle(this.styleStore());
  });

  /**
   * List of all logs in the inspection data.
   */
  protected readonly allLogs = computed(() => {
    const logStore = this.inspectionData()?.logStore;
    if (!logStore) {
      return [];
    }
    return Array.from(logStore.logs());
  });

  /**
   * List of logs matching the current filter criteria.
   * Used for the histogram and log distribution views.
   */
  protected readonly filteredLogs = computed(() => {
    return this.inspectionDataStore.timelineView()?.filteredLogs() ?? [];
  });

  /**
   * The start time of the inspection data range.
   * Used to determine the minimum scrollable/viewable time.
   */
  protected readonly minQueryLogTimeMS = computed(() => {
    const header = this.inspectionData()?.metadata?.header;
    if (header) {
      return header.startTimeUnixSeconds * 1000;
    }
    const logs = this.filteredLogs();
    if (logs.length === 0) {
      return Date.now() - 60 * 60 * 1000;
    }
    return logs[0].legacyTimestampMs;
  });

  /**
   * The end time of the inspection data range.
   * Used to determine the maximum scrollable/viewable time.
   */
  protected readonly maxQueryLogTimeMS = computed(() => {
    const header = this.inspectionData()?.metadata?.header;
    if (header) {
      return header.endTimeUnixSeconds * 1000;
    }
    const logs = this.filteredLogs();
    if (logs.length === 0) {
      return Date.now();
    }
    return logs[logs.length - 1].legacyTimestampMs;
  });

  protected readonly viewportLeftTimeMs = toSignal(
    this.viewStateService.timeOffset,
    { initialValue: 0 },
  );

  protected readonly timezoneShiftHours = toSignal(
    this.viewStateService.timezoneShift,
    { initialValue: 0 },
  );

  private readonly highlightedTimeline = computed(() => {
    return this.selectionManager.highlightedTimeline();
  });

  private readonly selectedTimeline = computed(() => {
    return this.selectionManager.selectedTimeline();
  });

  private readonly highlightedRevisionsOnCurrentTimeline = computed(() => {
    return this.selectionManager.highlightedChildrenOfSelectedTimeline();
  });

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
          (result[timeline.id] = TimelineHighlightType.ChildrenOfSelected),
      );
    }
    const highlighted = this.highlightedTimeline();
    if (highlighted) {
      result[highlighted.id] = TimelineHighlightType.Hovered;
    }
    const timeline = this.selectedTimeline();
    if (timeline) {
      result[timeline.id] = TimelineHighlightType.Selected;
    }
    return result;
  });

  private readonly selectedLogIndex = computed(() => {
    return this.selectionManager.selectedLogIndex();
  });

  private readonly highlightedLogIndices = computed(() => {
    return this.selectionManager.highlightLogIndices();
  });

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

  private readonly selectedLog = computed(() => {
    return this.selectionManager.selectedLog();
  });

  /**
   * The time of the currently selected log.
   * Used to display a vertical cursor line on the timeline.
   */
  protected readonly cursorTimeMs = computed(() => {
    const log = this.selectedLog();
    if (!log) {
      return 0;
    }
    return log.legacyTimestampMs;
  });

  private readonly lastClickedTimeMs = signal(0);

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

      const beginTimeMs = lastClickedTimeMs - optimalT;
      const endTimeMs = lastClickedTimeMs + optimalT;

      const beginTimeNs = BigInt(Math.floor(beginTimeMs)) * 1000000n;
      const endTimeNs = BigInt(Math.floor(endTimeMs)) * 1000000n;

      const events = timeline.lookupEventsInRangeNs(beginTimeNs, endTimeNs);
      const revisions = timeline.lookupRevisionsInRangeNs(
        beginTimeNs,
        endTimeNs,
      );
      let findRevisionStartTimeNs = beginTimeNs;
      if (revisions.length > 0) {
        findRevisionStartTimeNs = revisions[0].changedTime;
      }
      const initialRevision = timeline.lookupRevisionAtNs(
        findRevisionStartTimeNs,
        true,
      );

      return {
        timelineId: timeline.id,
        timeMs: lastClickedTimeMs,
        overlay: {
          timeline: timeline,
          revisions: revisions,
          events: events,
          initialRevision: initialRevision,
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
  protected hoverOnTimeline(
    event: ReadonlyDomainElement<Timeline> | null,
  ): void {
    this.selectionManager.onHighlightTimeline(event);
  }

  /**
   * Handles clicking on a timeline ruler item.
   * Updates the selection manager to select the timeline.
   */
  protected clickOnTimeline(event: ReadonlyDomainElement<Timeline>): void {
    this.selectionManager.onSelectTimeline(event);
  }

  /**
   * Handles hovering over an item (event or revision) on the timeline chart.
   * Updates highlights for the timeline and the specific log.
   */
  protected hoverOnTimelineItem(event: TimelineChartMouseEvent): void {
    this.selectionManager.onHighlightTimeline(event.timeline);
    if (event.timeline === null) {
      this.selectionManager.onHighlightLog();
    } else {
      if (event.revisionIndex !== undefined) {
        this.selectionManager.onHighlightLog(
          event.timeline.revisions[event.revisionIndex].log,
        );
        this.lastClickedTimeMs.set(event.timeMS);
      } else if (event.eventIndex !== undefined) {
        this.selectionManager.onHighlightLog(
          event.timeline.events[event.eventIndex].log,
        );
        this.lastClickedTimeMs.set(event.timeMS);
      } else {
        this.selectionManager.onHighlightLog();
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
        this.selectionManager.onSelectRevision(
          event.timeline.revisions[event.revisionIndex],
        );
      } else if (event.eventIndex !== undefined) {
        this.selectionManager.onSelectEvent(
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
    timeline: ReadonlyDomainElement<Timeline>,
    centerTimeMs: number,
    maxT: number,
    maxC: number,
  ): number {
    let low = 0;
    let high = maxT;
    let optimalT = 0;

    while (low <= high) {
      const mid = Math.floor((low + high) / 2);

      const beginTimeNs = BigInt(Math.floor(centerTimeMs - mid)) * 1000000n;
      const endTimeNs = BigInt(Math.floor(centerTimeMs + mid)) * 1000000n;

      const events = timeline.lookupEventsInRangeNs(beginTimeNs, endTimeNs);
      const revisions = timeline.lookupRevisionsInRangeNs(
        beginTimeNs,
        endTimeNs,
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
