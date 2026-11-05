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

import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import {
  ApplicationRef,
  Component,
  DestroyRef,
  InjectionToken,
  OnInit,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { lastValueFrom } from 'rxjs';
import {
  KhiTimelineRendererBridge,
  TimelineRenderMeasureResult,
  TimelineRenderParams,
  TimelineSceneOverlay,
} from 'src/app/common/schema/timeline-render-types';
import { TimelineRenderLayoutComponent } from 'src/app/pages/timeline-render/components/timeline-render-layout.component';
import {
  buildTimelineSceneViewModel,
  extractTimelineRowMetrics,
} from 'src/app/pages/timeline-render/timeline-render-scene-builder';
import {
  TimelineSceneViewModel,
  TimelineSessionJumpTarget,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';
import {
  MIN_INDEX_PANE_WIDTH_PX,
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
  TimelineChartStyle,
  TimelineRulerStyle,
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model';
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

interface SelectedTimelinesResult {
  readonly matched: readonly ReadonlyDomainElement<Timeline>[];
  readonly requiredIds: ReadonlySet<number>;
}

/**
 * Dependency injection token for opening full KHI session links in a new browser tab.
 */
export const TIMELINE_SESSION_URL_OPENER = new InjectionToken<
  (url: string) => void
>('TIMELINE_SESSION_URL_OPENER', {
  providedIn: 'root',
  factory: () => (url: string) => {
    window.open(url, '_blank', 'noopener');
  },
});

/**
 * Smart component dedicated to MCP timeline scene rendering for both headless CDP capture and interactive viewing.
 */
@Component({
  selector: 'khi-mcp-timeline-render-smart',
  imports: [TimelineRenderLayoutComponent],
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

  private readonly route = inject(ActivatedRoute);
  private readonly http = inject(HttpClient);
  private readonly appRef = inject(ApplicationRef);
  private readonly destroyRef = inject(DestroyRef);
  private readonly dataLoader = inject(InspectionDataLoaderService);
  private readonly inspectionDataStore = inject(InspectionDataStore);
  private readonly styleOverrideService = inject(StyleOverrideService);
  private readonly measureText = inject(TIMELINE_TEXT_WIDTH_MEASURER);
  private readonly sessionUrlOpener = inject(TIMELINE_SESSION_URL_OPENER);

  private readonly renderLayout =
    viewChild.required<TimelineRenderLayoutComponent>(
      TimelineRenderLayoutComponent,
    );

  private loadedInspectionId: string | null = null;
  private pendingPixelsPerNs = 1e-8;
  private pendingViewportLeftTimeNs = 0n;
  private lastAppliedParams: TimelineRenderParams | null = null;
  private isDestroyed = false;

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

  protected readonly activeOverlay = signal<TimelineSceneOverlay | null>(null);

  protected readonly interactiveMode = signal<boolean>(false);

  protected readonly activeSceneId = signal<string>('');

  protected readonly activeParentSceneId = signal<string>('');

  protected readonly activeInspectionId = signal<string>('');

  protected readonly errorMessage = signal<string>('');

  protected readonly hoveredBadge = signal<string | null>(null);

  protected readonly styleStore = computed<StyleStoreLike>(() => {
    return this.styleOverrideService;
  });

  protected readonly chartStyle = computed<TimelineChartStyle>(() => {
    return generateDefaultChartStyle();
  });

  protected readonly rulerStyle = computed<TimelineRulerStyle>(() => {
    return generateDefaultRulerStyle(this.styleStore());
  });

  protected readonly sceneViewModel = computed<TimelineSceneViewModel>(() => {
    const rows = extractTimelineRowMetrics(
      this.renderedTimelines(),
      this.styleStore(),
    );
    return buildTimelineSceneViewModel({
      indexPaneWidthPx: this.indexPaneWidthPx(),
      viewportLeftTimeMs: BigIntTimeUtil.NsToNumberMs(
        this.viewportLeftTimeNs(),
      ),
      pixelsPerMs: this.pixelsPerNs() * 1_000_000,
      rows,
      overlay: this.activeOverlay(),
      measureText: this.measureText,
    });
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
   * Registers the CDP automation bridge and loads any sceneId query parameter for interactive viewing.
   */
  public ngOnInit(): void {
    const bridge: KhiTimelineRendererBridge = {
      applyAndMeasure: async (params) => {
        await document.fonts.ready;
        this.interactiveMode.set(false);
        this.errorMessage.set('');
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
      this.isDestroyed = true;
      if (window.__khiCDPBridge?.timelineRenderer === bridge) {
        delete window.__khiCDPBridge.timelineRenderer;
      }
    });

    const sceneId = this.route.snapshot.queryParamMap.get('sceneId');
    if (sceneId) {
      void this.loadInteractiveScene(sceneId);
    }
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

    this.lastAppliedParams = params;
    this.activeInspectionId.set(params.inspectionId);
    this.activeSceneId.set(params.sceneId ?? '');
    this.activeParentSceneId.set(params.parentSceneId ?? '');
    this.hoveredBadge.set(null);

    await this.applyBackendFilter(timelineView, params);
    const selection = this.selectMatchedTimelines(
      timelineView.filteredTimelines(),
      params,
    );

    const maxNodes =
      params.maxNodes && params.maxNodes > 0
        ? params.maxNodes
        : McpTimelineRenderSmartComponent.DEFAULT_MAX_NODES;
    const rendered = this.sliceTimelinesPreservingRequired(
      selection.matched,
      selection.requiredIds,
      maxNodes,
    );
    this.renderedTimelines.set(rendered);
    this.timezoneShiftHours.set(params.timezoneShiftHours ?? 0);
    this.activeOverlay.set(params.overlay ?? null);

    const dimensions = calculateTimelineStageDimensions(
      rendered,
      this.styleStore(),
      this.measureText,
      params.overlay,
    );
    this.indexPaneWidthPx.set(dimensions.indexPaneWidthPx);

    const { viewStartTimeMs, viewEndTimeMs } = this.applyHorizontalViewport(
      dimensions.chartViewportWidthPx,
      params.viewStartTimeMs,
      params.viewEndTimeMs,
    );

    if (this.isDestroyed) {
      return {
        width: dimensions.stageWidthPx,
        height: 0,
        renderedCount: rendered.length,
        totalMatched: selection.matched.length,
        hasMore: selection.matched.length > rendered.length,
        viewStartTimeMs,
        viewEndTimeMs,
      };
    }

    this.appRef.tick();

    const vm = this.sceneViewModel();
    const measuredDeckHeightPx = this.renderLayout().measureDeckHeightPx();
    const deckHeightPx =
      vm.deck.cards.length > 0
        ? Math.max(vm.deck.estimatedHeightPx, measuredDeckHeightPx)
        : 0;
    const height = vm.stageHeightPx + deckHeightPx;

    return {
      width: dimensions.stageWidthPx,
      height,
      renderedCount: rendered.length,
      totalMatched: selection.matched.length,
      hasMore: selection.matched.length > rendered.length,
      viewStartTimeMs,
      viewEndTimeMs,
    };
  }

  /**
   * Waits for frame rendering to settle and syncs the canvas viewport state.
   */
  public async waitForStableFrame(): Promise<void> {
    if (this.isDestroyed) {
      return;
    }
    const layout = this.renderLayout();
    const maxAttempts = 200;
    for (
      let i = 0;
      i < maxAttempts && !this.isDestroyed && !layout.isFrameReadyToRender();
      i++
    ) {
      await this.nextAnimationFrame();
    }
    if (this.isDestroyed) {
      return;
    }

    await this.nextAnimationFrame();
    if (this.isDestroyed) {
      return;
    }
    this.appRef.tick();
    await this.nextAnimationFrame();
    if (this.isDestroyed) {
      return;
    }

    this.pixelsPerNs.set(this.pendingPixelsPerNs);
    this.viewportLeftTimeNs.set(this.pendingViewportLeftTimeNs);
    this.appRef.tick();
    layout.syncViewportAndInvalidate(
      this.pendingViewportLeftTimeNs,
      this.pendingPixelsPerNs,
    );

    await this.nextAnimationFrame();
    if (this.isDestroyed) {
      return;
    }
    this.appRef.tick();
    layout.syncViewportAndInvalidate(
      this.pendingViewportLeftTimeNs,
      this.pendingPixelsPerNs,
    );
    await this.nextAnimationFrame();
  }

  /**
   * Updates synchronized badge hover highlighting across pins and deck cards.
   */
  protected onBadgeHover(badge: string | null): void {
    this.hoveredBadge.set(badge);
  }

  /**
   * Opens a full KHI session focused on the clicked revision or log entry.
   */
  protected onJumpToSession(target: TimelineSessionJumpTarget): void {
    const inspectionId = this.activeInspectionId();
    if (!inspectionId) {
      return;
    }
    const search = new URLSearchParams();
    search.set('inspectionId', inspectionId);
    if (target.timelineId !== undefined) {
      search.set('timelineId', String(target.timelineId));
    }
    if (target.revisionIndex !== undefined) {
      search.set('revisionIndex', String(target.revisionIndex));
    }
    if (target.logId !== undefined) {
      search.set('logId', String(target.logId));
    }
    const sessionMatch =
      typeof window !== 'undefined'
        ? window.location.pathname.match(/\/session\/([^/]+)/)
        : null;
    const sessionId = sessionMatch ? sessionMatch[1] : '0';
    this.sessionUrlOpener(`/session/${sessionId}?${search.toString()}`);
  }

  /**
   * Opens the current scene's inspection and active queries in a full KHI session.
   */
  protected onOpenFullSession(): void {
    const inspectionId = this.activeInspectionId();
    if (!inspectionId) {
      return;
    }
    const search = new URLSearchParams();
    search.set('inspectionId', inspectionId);
    if (this.lastAppliedParams?.timelineQuery) {
      search.set('timelineQuery', this.lastAppliedParams.timelineQuery);
    }
    if (this.lastAppliedParams?.logQuery) {
      search.set('logQuery', this.lastAppliedParams.logQuery);
    }
    const sessionMatch =
      typeof window !== 'undefined'
        ? window.location.pathname.match(/\/session\/([^/]+)/)
        : null;
    const sessionId = sessionMatch ? sessionMatch[1] : '0';
    this.sessionUrlOpener(`/session/${sessionId}?${search.toString()}`);
  }

  private async loadInteractiveScene(sceneId: string): Promise<void> {
    this.interactiveMode.set(true);
    this.activeSceneId.set(sceneId);
    this.errorMessage.set('');
    try {
      const sceneDoc = await lastValueFrom(
        this.http.get<TimelineRenderParams>(
          `/api/v1/mcp-scenes/${encodeURIComponent(sceneId)}`,
        ),
      );
      if (this.isDestroyed) {
        return;
      }
      await this.applyAndMeasure(sceneDoc);
      if (!this.isDestroyed) {
        await this.waitForStableFrame();
      }
    } catch (err) {
      if (this.isDestroyed) {
        return;
      }
      if (err instanceof HttpErrorResponse && err.status === 410) {
        this.errorMessage.set(
          `Scene ${sceneId} has expired. Re-run render_timelines to generate a fresh scene.`,
        );
      } else if (err instanceof HttpErrorResponse && err.status === 404) {
        this.errorMessage.set(`Scene ${sceneId} was not found.`);
      } else {
        const detail = err instanceof Error ? err.message : String(err);
        this.errorMessage.set(`Failed to load scene ${sceneId}: ${detail}`);
      }
    }
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
  ): SelectedTimelinesResult {
    const storeTimelines =
      this.inspectionDataStore.inspectionData()?.timelineStore.timelines ??
      candidates;
    const timelineById = new Map<number, ReadonlyDomainElement<Timeline>>();
    for (const timeline of storeTimelines) {
      timelineById.set(timeline.id, timeline);
    }

    let matched = [...candidates];
    if (params.timelineIds && params.timelineIds.length > 0) {
      const allowedIds = new Set<number>();
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

    const requiredIds = new Set<number>();
    const autoIncludeIds = params.overlay?.autoIncludeTimelineIds ?? [];
    for (const targetId of autoIncludeIds) {
      const target = timelineById.get(targetId);
      if (target) {
        requiredIds.add(target.id);
        for (const node of target.path) {
          requiredIds.add(node.id);
        }
      }
    }

    if (requiredIds.size > 0) {
      const unionIds = new Set<number>(matched.map((t) => t.id));
      for (const reqId of requiredIds) {
        unionIds.add(reqId);
      }
      matched = storeTimelines.filter((timeline) => unionIds.has(timeline.id));
    }

    return { matched, requiredIds };
  }

  private sliceTimelinesPreservingRequired(
    matched: readonly ReadonlyDomainElement<Timeline>[],
    requiredIds: ReadonlySet<number>,
    maxNodes: number,
  ): ReadonlyDomainElement<Timeline>[] {
    if (matched.length <= maxNodes || requiredIds.size === 0) {
      return matched.slice(0, maxNodes);
    }

    let requiredCountInMatched = 0;
    for (const timeline of matched) {
      if (requiredIds.has(timeline.id)) {
        requiredCountInMatched++;
      }
    }

    let optionalBudget = Math.max(0, maxNodes - requiredCountInMatched);
    const result: ReadonlyDomainElement<Timeline>[] = [];
    for (const timeline of matched) {
      if (requiredIds.has(timeline.id)) {
        result.push(timeline);
      } else if (optionalBudget > 0) {
        result.push(timeline);
        optionalBudget--;
      }
    }
    return result;
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
