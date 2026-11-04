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

import {
  ApplicationRef,
  Component,
  DestroyRef,
  OnInit,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import {
  KhiTimelineRendererBridge,
  TimelineRenderMeasureResult,
  TimelineRenderParams,
} from 'src/app/common/schema/timeline-render-types';
import {
  MIN_CHART_VIEWPORT_WIDTH_PX,
  MIN_INDEX_PANE_WIDTH_PX,
  TIMELINE_GUTTER_WIDTH_PX,
  TIMELINE_HEADER_HEIGHT_PX,
  TIMELINE_RENDER_HORIZONTAL_OVERDRAW_PX,
  TIMELINE_TEXT_WIDTH_MEASURER,
  calculateTimelineStageDimensions,
} from 'src/app/pages/timeline-render/types/timeline-render-sizing';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import {
  PROGRESS_DIALOG_STATUS_UPDATOR,
  ProgressDialogStatusUpdator,
} from 'src/app/services/progress/progress-interface';
import { StyleOverrideService } from 'src/app/services/style-override.service';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { InspectionData } from 'src/app/store/domain/inspection-data';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineView } from 'src/app/store/domain/timeline-view';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  BASE_ROW_HEIGHT,
  TimelineChartStyle,
  TimelineRulerStyle,
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model';
import { TimelineFrameComponent } from 'src/app/timeline/components/timeline-frame.component';
import { BigIntTimeUtil } from 'src/app/utils/bigint-time-util';

const NOOP_PROGRESS_UPDATOR: ProgressDialogStatusUpdator = {
  show: () => undefined,
  dismiss: () => undefined,
  updateProgress: () => undefined,
};

/**
 * Visible time range in milliseconds applied to the timeline viewport.
 */
interface ViewportTimeRange {
  readonly viewStartTimeMs: number;
  readonly viewEndTimeMs: number;
}

/**
 * Smart component dedicated to headless MCP timeline screenshot rendering.
 */
@Component({
  selector: 'khi-mcp-timeline-render-smart',
  imports: [TimelineFrameComponent],
  templateUrl: './mcp-timeline-render-smart.component.html',
  styleUrls: ['./mcp-timeline-render-smart.component.scss'],
  providers: [
    {
      provide: PROGRESS_DIALOG_STATUS_UPDATOR,
      useValue: NOOP_PROGRESS_UPDATOR,
    },
    InspectionDataLoaderService,
  ],
})
export class McpTimelineRenderSmartComponent implements OnInit {
  private static readonly DEFAULT_MAX_NODES = 50;

  private readonly appRef = inject(ApplicationRef);
  private readonly destroyRef = inject(DestroyRef);
  private readonly dataLoader = inject(InspectionDataLoaderService);
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly styleOverrideService = inject(StyleOverrideService);
  private readonly measureText = inject(TIMELINE_TEXT_WIDTH_MEASURER);

  private readonly timelineFrame = viewChild.required<TimelineFrameComponent>(
    TimelineFrameComponent,
  );

  private loadedInspectionId: string | null = null;
  private pendingPixelsPerNs = 1e-8;
  private pendingViewportLeftTimeNs = 0n;

  private readonly inspectionDataUniqueIDs = new WeakMap<
    InspectionData,
    string
  >();

  protected readonly collapsedTimelineIds: ReadonlySet<number> = new Set();

  protected readonly horizontalOverdrawPx =
    TIMELINE_RENDER_HORIZONTAL_OVERDRAW_PX;

  protected readonly renderedTimelines = signal<
    readonly ReadonlyDomainElement<Timeline>[]
  >([]);

  protected readonly timezoneShiftHours = signal<number>(0);

  protected readonly pixelsPerNs = signal<number>(1e-8);

  protected readonly viewportLeftTimeNs = signal<bigint>(0n);

  protected readonly indexPaneWidthPx = signal<number>(MIN_INDEX_PANE_WIDTH_PX);

  protected readonly stageWidthPx = signal<number>(
    MIN_INDEX_PANE_WIDTH_PX +
      TIMELINE_GUTTER_WIDTH_PX +
      MIN_CHART_VIEWPORT_WIDTH_PX,
  );

  protected readonly stageHeightPx = signal<number>(
    TIMELINE_HEADER_HEIGHT_PX + BASE_ROW_HEIGHT,
  );

  protected readonly styleStore = computed<StyleStoreLike>(() => {
    return this.styleOverrideService;
  });

  protected readonly chartStyle = computed<TimelineChartStyle>(() => {
    return generateDefaultChartStyle();
  });

  protected readonly rulerStyle = computed<TimelineRulerStyle>(() => {
    return generateDefaultRulerStyle(this.styleStore());
  });

  protected readonly inspectionDataUniqueID = computed(() => {
    const data = this.inspectionDataStore.inspectionData();
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

  protected readonly allLogs = computed(() => {
    const logStore = this.inspectionDataStore.inspectionData()?.logStore;
    if (!logStore) {
      return [];
    }
    return Array.from(logStore.logs());
  });

  protected readonly filteredLogIds = computed(() => {
    return (
      this.inspectionDataStore.timelineView()?.filteredLogIds() ??
      IdBitset.createEmpty()
    );
  });

  protected readonly minQueryLogTimeNs = computed<bigint>(() => {
    const header = this.inspectionDataStore.inspectionData()?.metadata?.header;
    if (header) {
      return BigInt(header.startTimeUnixSeconds) * 1_000_000_000n;
    }
    const logs = this.allLogs();
    if (logs.length === 0) {
      return BigInt(Date.now() - 60 * 60 * 1000) * 1_000_000n;
    }
    return logs[0].timestamp;
  });

  protected readonly maxQueryLogTimeNs = computed<bigint>(() => {
    const header = this.inspectionDataStore.inspectionData()?.metadata?.header;
    if (header) {
      return BigInt(header.endTimeUnixSeconds) * 1_000_000_000n;
    }
    const logs = this.allLogs();
    if (logs.length === 0) {
      return BigInt(Date.now()) * 1_000_000n;
    }
    return logs[logs.length - 1].timestamp;
  });

  /**
   * Registers the window.__khiCDPBridge.timelineRenderer automation bridge for MCP headless clients.
   */
  public ngOnInit(): void {
    const bridge: KhiTimelineRendererBridge = {
      applyAndMeasure: async (params) => {
        await document.fonts.ready;
        return this.applyAndMeasure(params);
      },
      waitForStableFrame: async () => {
        await document.fonts.ready;
        return this.waitForStableFrame();
      },
    };
    window.__khiCDPBridge = {
      ...window.__khiCDPBridge,
      timelineRenderer: bridge,
    };
    this.destroyRef.onDestroy(() => {
      if (window.__khiCDPBridge?.timelineRenderer === bridge) {
        delete window.__khiCDPBridge.timelineRenderer;
      }
    });
  }

  /**
   * Loads inspection data if needed, applies filters, selects target timelines, and computes stage dimensions.
   */
  public async applyAndMeasure(
    params: TimelineRenderParams,
  ): Promise<TimelineRenderMeasureResult> {
    await this.ensureInspectionLoaded(params.inspectionId);
    const timelineView = this.inspectionDataStore.timelineView();
    if (!timelineView) {
      throw new Error('TimelineView is not initialized.');
    }

    await this.applyBackendFilter(timelineView, params);
    const matched = this.selectMatchedTimelines(
      timelineView.filteredTimelines(),
      params,
    );

    const maxNodes =
      params.maxNodes && params.maxNodes > 0
        ? params.maxNodes
        : McpTimelineRenderSmartComponent.DEFAULT_MAX_NODES;
    const rendered = matched.slice(0, maxNodes);
    this.renderedTimelines.set(rendered);
    this.timezoneShiftHours.set(params.timezoneShiftHours ?? 0);

    const dimensions = calculateTimelineStageDimensions(
      rendered,
      this.styleStore(),
      this.measureText,
      params.overlay,
    );
    this.indexPaneWidthPx.set(dimensions.indexPaneWidthPx);
    this.stageWidthPx.set(dimensions.stageWidthPx);
    this.stageHeightPx.set(dimensions.timelineFrameHeightPx);

    const { viewStartTimeMs, viewEndTimeMs } = this.applyHorizontalViewport(
      dimensions.chartViewportWidthPx,
      params.viewStartTimeMs,
      params.viewEndTimeMs,
    );

    this.appRef.tick();

    return {
      width: dimensions.stageWidthPx,
      height: dimensions.timelineFrameHeightPx,
      renderedCount: rendered.length,
      totalMatched: matched.length,
      hasMore: matched.length > rendered.length,
      viewStartTimeMs,
      viewEndTimeMs,
    };
  }

  /**
   * Waits for frame rendering to settle and syncs the canvas viewport state.
   */
  public async waitForStableFrame(): Promise<void> {
    const frame = this.timelineFrame();
    const maxAttempts = 200;
    for (let i = 0; i < maxAttempts && !frame.readyToRender(); i++) {
      await this.nextAnimationFrame();
    }

    await this.nextAnimationFrame();
    this.appRef.tick();
    await this.nextAnimationFrame();

    this.pixelsPerNs.set(this.pendingPixelsPerNs);
    this.viewportLeftTimeNs.set(this.pendingViewportLeftTimeNs);
    this.appRef.tick();
    frame.syncViewportAndInvalidate(
      this.pendingViewportLeftTimeNs,
      this.pendingPixelsPerNs,
    );

    await this.nextAnimationFrame();
    this.appRef.tick();
    frame.syncViewportAndInvalidate(
      this.pendingViewportLeftTimeNs,
      this.pendingPixelsPerNs,
    );
    await this.nextAnimationFrame();
  }

  private async ensureInspectionLoaded(inspectionId: string): Promise<void> {
    if (
      this.loadedInspectionId === inspectionId &&
      this.inspectionDataStore.inspectionData() !== null
    ) {
      return;
    }
    const previousData = this.inspectionDataStore.inspectionData();
    this.loadedInspectionId = null;
    await this.dataLoader.loadInspectionDataFromBackend(inspectionId);
    const currentData = this.inspectionDataStore.inspectionData();
    if (!currentData || currentData === previousData) {
      throw new Error(
        `Failed to load inspection data for inspectionId=${inspectionId}`,
      );
    }
    this.loadedInspectionId = inspectionId;
  }

  private async applyBackendFilter(
    timelineView: TimelineView,
    params: TimelineRenderParams,
  ): Promise<void> {
    timelineView.expandAllTimelines();
    const filterStartTime = params.filterStartTimeNs
      ? BigInt(params.filterStartTimeNs)
      : null;
    const filterEndTime = params.filterEndTimeNs
      ? BigInt(params.filterEndTimeNs)
      : null;

    timelineView.backendFilter.updateFilterParams({
      timelineQuery: params.timelineQuery ?? '',
      timelineExclusionQuery: params.timelineExclusionQuery ?? '',
      logQuery: params.logQuery ?? '',
      excludeNoLogs: params.excludeNoLogs ?? false,
      filterStartTime,
      filterEndTime,
    });

    await Promise.resolve();
    while (timelineView.isFiltering()) {
      await new Promise((resolve) => setTimeout(resolve, 10));
    }
  }

  private selectMatchedTimelines(
    candidates: readonly ReadonlyDomainElement<Timeline>[],
    params: TimelineRenderParams,
  ): ReadonlyDomainElement<Timeline>[] {
    let matched = [...candidates];
    if (params.timelineIds && params.timelineIds.length > 0) {
      const allowedIds = new Set<number>();
      const timelineById = new Map<number, ReadonlyDomainElement<Timeline>>();
      const storeTimelines =
        this.inspectionDataStore.inspectionData()?.timelineStore.timelines ??
        candidates;
      for (const timeline of storeTimelines) {
        timelineById.set(timeline.id, timeline);
      }
      for (const targetId of params.timelineIds) {
        allowedIds.add(targetId);
        const target = timelineById.get(targetId);
        if (target) {
          for (const node of target.path) {
            allowedIds.add(node.id);
          }
        }
      }
      matched = matched.filter((timeline) => allowedIds.has(timeline.id));
    }

    if (params.maxDepth !== undefined && params.maxDepth > 0) {
      matched = matched.filter(
        (timeline) => timeline.layer <= params.maxDepth!,
      );
    }

    return matched;
  }

  private applyHorizontalViewport(
    chartViewportWidthPx: number,
    requestedStartTimeMs?: number,
    requestedEndTimeMs?: number,
  ): ViewportTimeRange {
    const minLogNs = this.minQueryLogTimeNs();
    const maxLogNs = this.maxQueryLogTimeNs();
    const viewStartTimeNs =
      requestedStartTimeMs !== undefined
        ? BigInt(Math.round(requestedStartTimeMs)) * 1_000_000n
        : minLogNs;
    const viewEndTimeNs =
      requestedEndTimeMs !== undefined
        ? BigInt(Math.round(requestedEndTimeMs)) * 1_000_000n
        : maxLogNs;
    const durationNs = Math.max(
      1_000_000,
      Number(viewEndTimeNs - viewStartTimeNs),
    );

    const safeChartWidthPx = Math.max(100, chartViewportWidthPx);
    const pixelsPerNs = safeChartWidthPx / durationNs;

    this.pendingPixelsPerNs = pixelsPerNs;
    this.pendingViewportLeftTimeNs = viewStartTimeNs;
    this.pixelsPerNs.set(pixelsPerNs);
    this.viewportLeftTimeNs.set(viewStartTimeNs);

    return {
      viewStartTimeMs: BigIntTimeUtil.NsToNumberMs(viewStartTimeNs),
      viewEndTimeMs: BigIntTimeUtil.NsToNumberMs(viewEndTimeNs),
    };
  }

  private nextAnimationFrame(): Promise<void> {
    return new Promise((resolve) => requestAnimationFrame(() => resolve()));
  }
}
