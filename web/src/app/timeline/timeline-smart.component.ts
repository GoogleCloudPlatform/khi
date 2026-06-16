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
import { Timeline, Event, Revision } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { InspectionDataV2 } from 'src/app/store/domain/inspection-data';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { StyleOverrideService } from 'src/app/services/style-override.service';
import {
  TimelineChartStyle,
  TimelineRulerStyle,
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model-v2';
import { TimelineChartMouseEvent } from 'src/app/timeline/components/timeline-chart.component';
import { bisectLeft } from 'src/app/common/misc-util';
import { BigIntTimeUtil } from 'src/app/utils/bigint-time-util';

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

  private readonly styleOverrideService = inject(StyleOverrideService);

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
  protected readonly styleStore = computed<StyleStoreLike>(() => {
    return this.styleOverrideService;
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

  private readonly lastClickedTime = signal(0n);

  private readonly isMouseOnTimeline = signal(false);

  /**
   * Handles the mouse entering the chart area.
   */
  protected onMouseEnterChart(): void {
    this.isMouseOnTimeline.set(true);
  }

  /**
   * Handles the mouse leaving the chart area.
   */
  protected onMouseLeaveChart(): void {
    this.isMouseOnTimeline.set(false);
  }

  private readonly isMouseOnStickyHeader = signal(false);

  /**
   * Handles the mouse entering the sticky header area.
   */
  protected onMouseEnterStickyHeader(): void {
    this.isMouseOnStickyHeader.set(true);
  }

  /**
   * Handles the mouse leaving the sticky header area.
   */
  protected onMouseLeaveStickyHeader(): void {
    this.isMouseOnStickyHeader.set(false);
  }

  /**
   * Resolves the target timeline and target timestamp for hover overlay rendering.
   * Prioritizes directly hovered timeline items over remote selection highlights.
   */
  private resolveTargetTimelineAndTime(): {
    timeline: ReadonlyDomainElement<Timeline>;
    targetTimeNs: bigint;
  } | null {
    if (this.isMouseOnTimeline()) {
      const timeline = this.highlightedTimeline();
      if (!timeline) {
        return null;
      }
      return { timeline, targetTimeNs: this.lastClickedTime() };
    }

    const highlightedLogs = this.selectionManager.highlightedLogs();
    if (highlightedLogs.length === 0) {
      return null;
    }
    const targetLog = highlightedLogs[0];
    const targetTimeNs = targetLog.timestamp;

    const currentSelected = this.selectedTimeline();
    if (currentSelected && currentSelected.hasLog(targetLog)) {
      return { timeline: currentSelected, targetTimeNs };
    }

    if (currentSelected) {
      const descendants = new Set(currentSelected.descendants());
      const allTimelines = this.filteredTimelines();
      const descendantMatch = allTimelines.find(
        (t) => descendants.has(t) && t.hasLog(targetLog),
      );
      if (descendantMatch) {
        return { timeline: descendantMatch, targetTimeNs };
      }
    }

    const allTimelines = this.filteredTimelines();
    const globalMatch = allTimelines.find((t) => t.hasLog(targetLog));
    return globalMatch ? { timeline: globalMatch, targetTimeNs } : null;
  }

  /**
   * Retrieves up to two log items (events or revisions) preceding the target time.
   * Uses binary search to quickly locate historical items on sorted timeline collections.
   */
  private lookupPrecedingLogs(
    timeline: ReadonlyDomainElement<Timeline>,
    targetTimeNs: bigint,
  ): (ReadonlyDomainElement<Event> | ReadonlyDomainElement<Revision>)[] {
    const events = timeline.events;
    const eEndIdx = bisectLeft(events, targetTimeNs, (item, target) =>
      item.timestamp < target ? -1 : 1,
    );
    const eStartIdx = Math.max(0, eEndIdx - 2);
    const prevEvents = events.slice(eStartIdx, eEndIdx);

    const revisions = timeline.revisions;
    const rEndIdx = bisectLeft(revisions, targetTimeNs, (item, target) =>
      item.changedTime < target ? -1 : 1,
    );
    const rStartIdx = Math.max(0, rEndIdx - 2);
    const prevRevisions = revisions.slice(rStartIdx, rEndIdx);

    const combinedPrev = [
      ...prevEvents.map((e) => ({ item: e, time: e.timestamp })),
      ...prevRevisions.map((r) => ({ item: r, time: r.changedTime })),
    ];
    combinedPrev.sort((a, b) =>
      a.time < b.time ? -1 : a.time > b.time ? 1 : 0,
    );
    return combinedPrev.slice(-2).map((x) => x.item);
  }

  /**
   * Retrieves succeeding log items within the selectable pixel range limit.
   * Truncates the result to fit within the remaining display budget.
   */
  private lookupSucceedingLogs(
    timeline: ReadonlyDomainElement<Timeline>,
    targetTimeNs: bigint,
    remainingCount: number,
  ): (ReadonlyDomainElement<Event> | ReadonlyDomainElement<Revision>)[] {
    if (remainingCount <= 0) {
      return [];
    }
    const rangeNs =
      BigInt(
        Math.floor(this.HOVER_VIEW_SELECTABLE_RANGE_IN_PX / this.pixelsPerMs()),
      ) * 1000000n;
    const maxTimeNs = targetTimeNs + rangeNs;

    const nextEvents = timeline.lookupEventsInRangeNs(targetTimeNs, maxTimeNs);
    const nextRevisionsRaw = timeline.lookupRevisionsInRangeNs(
      targetTimeNs,
      maxTimeNs,
    );
    const nextRevisions = nextRevisionsRaw.filter(
      (r) => r.changedTime >= targetTimeNs,
    );

    const combinedNext = [
      ...nextEvents.map((e) => ({ item: e, time: e.timestamp })),
      ...nextRevisions.map((r) => ({ item: r, time: r.changedTime })),
    ];
    combinedNext.sort((a, b) =>
      a.time < b.time ? -1 : a.time > b.time ? 1 : 0,
    );

    return combinedNext.slice(0, remainingCount).map((x) => x.item);
  }

  /**
   * Data required to render the hover overlay (tooltip) when hovering over the timeline.
   * Calculates specific events or revisions near the hovered time.
   */
  protected readonly timelineHoverOverlayRequest =
    computed<TimelineHoverOverlayRequest | null>(() => {
      const resolved = this.resolveTargetTimelineAndTime();
      if (!resolved) {
        return null;
      }
      const { timeline, targetTimeNs } = resolved;
      const targetTimeMs = BigIntTimeUtil.NsToNumberMs(targetTimeNs);

      const prevLogs = this.lookupPrecedingLogs(timeline, targetTimeNs);
      const remainingCount = this.MAX_HOVER_VIEW_LOG_COUNT - prevLogs.length;
      const nextLogs = this.lookupSucceedingLogs(
        timeline,
        targetTimeNs,
        remainingCount,
      );

      const finalEvents: ReadonlyDomainElement<Event>[] = [];
      const finalRevisions: ReadonlyDomainElement<Revision>[] = [];

      for (const item of [...prevLogs, ...nextLogs]) {
        if ('changedTime' in item) {
          finalRevisions.push(item as ReadonlyDomainElement<Revision>);
        } else {
          finalEvents.push(item as ReadonlyDomainElement<Event>);
        }
      }

      let findRevisionStartTimeNs = targetTimeNs;
      if (prevLogs.length > 0) {
        const firstPrev = prevLogs[0];
        findRevisionStartTimeNs =
          'changedTime' in firstPrev
            ? (firstPrev as { changedTime: bigint }).changedTime
            : (firstPrev as { timestamp: bigint }).timestamp;
      }
      const initialRevision = timeline.lookupRevisionAtNs(
        findRevisionStartTimeNs,
        true,
      );

      const hasHighlightedLog =
        this.selectionManager.highlightedLogs().length > 0;
      return {
        timelineId: timeline.id,
        timeMs: targetTimeMs,
        isMouseOnTimeline: this.isMouseOnTimeline(),
        isStickyHeaderHover: this.isMouseOnStickyHeader(),
        overlay: {
          timeline: timeline,
          revisions: finalRevisions,
          events: finalEvents,
          initialRevision: initialRevision,
          cursorTime: hasHighlightedLog ? null : targetTimeNs,
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
        const log = event.timeline.revisions[event.revisionIndex].log;
        this.selectionManager.onHighlightLog(log);
        this.lastClickedTime.set(
          log ? log.timestamp : BigInt(Math.floor(event.timeMS)) * 1000000n,
        );
      } else if (event.eventIndex !== undefined) {
        const log = event.timeline.events[event.eventIndex].log;
        this.selectionManager.onHighlightLog(log);
        this.lastClickedTime.set(
          log ? log.timestamp : BigInt(Math.floor(event.timeMS)) * 1000000n,
        );
      } else {
        this.selectionManager.onHighlightLog();
        this.lastClickedTime.set(BigInt(Math.floor(event.timeMS)) * 1000000n);
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
}
