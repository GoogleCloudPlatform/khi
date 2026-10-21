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
import { ActivatedRoute } from '@angular/router';
import {
  KhiTimelineRendererBridge,
  TimelineRenderMeasureResult,
  TimelineRenderParams,
} from 'src/app/common/schema/timeline-render-types';
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

const NOOP_PROGRESS_UPDATOR: ProgressDialogStatusUpdator = {
  show: () => undefined,
  dismiss: () => undefined,
  updateProgress: () => undefined,
};

/**
 * Smart component that renders only the timeline frame for headless screenshot capture.
 *
 * It exposes a browser automation bridge on window.__khiCDPBridge.timelineRenderer so the backend MCP
 * renderer can dynamically switch filters, timeline subsets, and time ranges without reloading.
 */
@Component({
  selector: 'khi-timeline-render-smart',
  imports: [TimelineFrameComponent],
  templateUrl: './timeline-render-smart.component.html',
  styleUrls: ['./timeline-render-smart.component.scss'],
  providers: [
    {
      provide: PROGRESS_DIALOG_STATUS_UPDATOR,
      useValue: NOOP_PROGRESS_UPDATOR,
    },
    InspectionDataLoaderService,
  ],
})
export class TimelineRenderSmartComponent implements OnInit {
  private static readonly HEADER_HEIGHT = 60;
  private static readonly INDEX_PANE_WIDTH = 300;
  private static readonly GUTTER_WIDTH = 8;
  private static readonly DEFAULT_WIDTH = 1600;
  private static readonly MIN_WIDTH = 800;
  private static readonly MAX_WIDTH = 3840;
  private static readonly DEFAULT_MAX_NODES = 50;

  private readonly route = inject(ActivatedRoute);
  private readonly appRef = inject(ApplicationRef);
  private readonly destroyRef = inject(DestroyRef);
  private readonly dataLoader = inject(InspectionDataLoaderService);
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly styleOverrideService = inject(StyleOverrideService);

  private readonly timelineFrame = viewChild.required<TimelineFrameComponent>(
    TimelineFrameComponent,
  );

  private loadedInspectionId: string | null = null;
  private pendingPixelsPerMs = 0.01;
  private pendingViewportLeftTimeMs = 0;

  private readonly inspectionDataUniqueIDs = new WeakMap<
    InspectionData,
    string
  >();

  protected readonly collapsedTimelineIds: ReadonlySet<number> = new Set();

  protected readonly renderedTimelines = signal<
    readonly ReadonlyDomainElement<Timeline>[]
  >([]);

  protected readonly timezoneShiftHours = signal<number>(0);

  protected readonly pixelsPerMs = signal<number>(0.01);

  protected readonly viewportLeftTimeMs = signal<number>(0);

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

  protected readonly minQueryLogTimeMS = computed(() => {
    const header = this.inspectionDataStore.inspectionData()?.metadata?.header;
    if (header) {
      return header.startTimeUnixSeconds * 1000;
    }
    const logs = this.allLogs();
    if (logs.length === 0) {
      return Date.now() - 60 * 60 * 1000;
    }
    return logs[0].legacyTimestampMs;
  });

  protected readonly maxQueryLogTimeMS = computed(() => {
    const header = this.inspectionDataStore.inspectionData()?.metadata?.header;
    if (header) {
      return header.endTimeUnixSeconds * 1000;
    }
    const logs = this.allLogs();
    if (logs.length === 0) {
      return Date.now();
    }
    return logs[logs.length - 1].legacyTimestampMs;
  });

  /**
   * Registers the window.__khiCDPBridge.timelineRenderer bridge and applies any initial URL query parameters.
   */
  public ngOnInit(): void {
    const bridge: KhiTimelineRendererBridge = {
      applyAndMeasure: (params) => this.applyAndMeasure(params),
      waitForStableFrame: () => this.waitForStableFrame(),
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

    const queryParams = this.route.snapshot.queryParamMap;
    const inspectionId = queryParams.get('inspectionId');
    if (inspectionId) {
      const timelineIdsParam = queryParams.get('timelineIds');
      const timelineIds = timelineIdsParam
        ? timelineIdsParam
            .split(',')
            .map((v) => Number(v.trim()))
            .filter((n) => Number.isInteger(n) && n > 0)
        : undefined;
      void this.applyAndMeasure({
        inspectionId,
        timelineQuery: queryParams.get('timelineQuery') ?? undefined,
        timelineExclusionQuery:
          queryParams.get('timelineExclusionQuery') ?? undefined,
        logQuery: queryParams.get('logQuery') ?? undefined,
        timelineIds,
      });
    }
  }

  /**
   * Loads inspection data if needed, applies filters, selects target timelines, and computes dimensions.
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
        : TimelineRenderSmartComponent.DEFAULT_MAX_NODES;
    const rendered = matched.slice(0, maxNodes);
    this.renderedTimelines.set(rendered);
    this.timezoneShiftHours.set(params.timezoneShiftHours ?? 0);

    const width = Math.max(
      TimelineRenderSmartComponent.MIN_WIDTH,
      Math.min(
        TimelineRenderSmartComponent.MAX_WIDTH,
        params.width ?? TimelineRenderSmartComponent.DEFAULT_WIDTH,
      ),
    );
    const height = this.calculateTotalHeight(rendered);
    const { viewStartTimeMs, viewEndTimeMs } = this.applyHorizontalViewport(
      width,
      params.viewStartTimeMs,
      params.viewEndTimeMs,
    );

    this.appRef.tick();

    return {
      width,
      height,
      renderedCount: rendered.length,
      totalMatched: matched.length,
      hasMore: matched.length > rendered.length,
      viewStartTimeMs,
      viewEndTimeMs,
    };
  }

  /**
   * Waits for ResizeObserver updates and WebGL/Canvas rendering frames to settle.
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

    this.pixelsPerMs.set(this.pendingPixelsPerMs);
    this.viewportLeftTimeMs.set(this.pendingViewportLeftTimeMs);
    this.appRef.tick();
    frame.syncViewportAndInvalidate(
      this.pendingViewportLeftTimeMs,
      this.pendingPixelsPerMs,
    );

    await this.nextAnimationFrame();
    this.appRef.tick();
    frame.syncViewportAndInvalidate(
      this.pendingViewportLeftTimeMs,
      this.pendingPixelsPerMs,
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

  private calculateTotalHeight(
    timelines: readonly ReadonlyDomainElement<Timeline>[],
  ): number {
    const styleStore = this.styleStore();
    let rowsHeight = 0;
    for (const timeline of timelines) {
      let timelineType = timeline.type;
      try {
        timelineType =
          styleStore.getTimelineType(timeline.type.id) ?? timeline.type;
      } catch {
        timelineType = timeline.type;
      }
      rowsHeight += timelineType.height * BASE_ROW_HEIGHT;
    }
    return Math.max(
      TimelineRenderSmartComponent.HEADER_HEIGHT + BASE_ROW_HEIGHT,
      Math.ceil(TimelineRenderSmartComponent.HEADER_HEIGHT + rowsHeight),
    );
  }

  private applyHorizontalViewport(
    width: number,
    requestedStartTimeMs?: number,
    requestedEndTimeMs?: number,
  ): { viewStartTimeMs: number; viewEndTimeMs: number } {
    const minLogMs = this.minQueryLogTimeMS();
    const maxLogMs = this.maxQueryLogTimeMS();
    const viewStartTimeMs = requestedStartTimeMs ?? minLogMs;
    const viewEndTimeMs = requestedEndTimeMs ?? maxLogMs;
    const durationMs = Math.max(1, viewEndTimeMs - viewStartTimeMs);

    const chartViewportWidth = Math.max(
      100,
      width -
        TimelineRenderSmartComponent.INDEX_PANE_WIDTH -
        TimelineRenderSmartComponent.GUTTER_WIDTH,
    );
    const pixelsPerMs = chartViewportWidth / durationMs;

    this.pendingPixelsPerMs = pixelsPerMs;
    this.pendingViewportLeftTimeMs = viewStartTimeMs;
    this.pixelsPerMs.set(pixelsPerMs);
    this.viewportLeftTimeMs.set(viewStartTimeMs);

    return { viewStartTimeMs, viewEndTimeMs };
  }

  private nextAnimationFrame(): Promise<void> {
    return new Promise((resolve) => requestAnimationFrame(() => resolve()));
  }
}
