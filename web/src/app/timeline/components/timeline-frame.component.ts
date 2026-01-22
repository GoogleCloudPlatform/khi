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
  AfterViewInit,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  input,
  model,
  NgZone,
  output,
  OutputEmitterRef,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { AngularSplitModule } from 'angular-split';
import { TimelineIndexComponent } from './timeline-index.component';
import { VerticalScrollCalculator } from './calculator/vertical-scroll-calculator';
import { RenderingLoopManager } from './canvas/rendering-loop-manager';
import { HorizontalScrollCalculator } from './calculator/horizontal-scroll-calculator';
import { TimelineRulerComponent } from './timeline-ruler.component';
import {
  TimelineChartComponent,
  TimelineChartMouseEvent,
} from './timeline-chart.component';
import { CaptureShiftKeyDirective } from 'src/app/common/capture-shiftkey.directive';
import { TimelineCornerIndicatorComponent } from './timeline-corner-indicator.component';
import { ResourceTimeline } from 'src/app/store/timeline';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { CommonModule } from '@angular/common';
import { TimelineLegendComponent } from './timeline-legend.component';
import { LogEntry } from 'src/app/store/log';
import { HistogramCache } from './misc/histogram-cache';
import {
  TimelineHoverOverlay,
  TimelineHoverOverlayComponent,
} from './timeline-hover-overlay.component';
import {
  RulerViewModelBuilder,
  TimelineRulerViewModel,
} from './timeline-ruler.viewmodel';
import { TimelineChartViewModel } from './timeline-chart.viewmodel';
import {
  TimelineHighlight,
  TimelineChartItemHighlight,
  TimelineHighlightType,
  TimelineChartItemHighlightType,
} from './interaction-model';
import {
  TimelineChartStyle,
  generateDefaultChartStyle,
  TimelineRulerStyle,
  generateDefaultRulerStyle,
} from './style-model';
import {
  getMinTimeSpanForHistogram,
  getTickTimeMS,
} from './calculator/human-friendly-tick';

export interface TimeScaleEvent {
  event: WheelEvent;
  centerTimeMs: number;
}

export interface TimelineHoverOverlayRequest {
  timeMs: number;
  timelineId: string;
  overlay: TimelineHoverOverlay;
}

@Component({
  selector: 'khi-timeline-frame',
  templateUrl: './timeline-frame.component.html',
  styleUrls: ['./timeline-frame.component.scss'],
  imports: [
    CommonModule,
    TimelineLegendComponent,
    AngularSplitModule,
    TimelineIndexComponent,
    TimelineRulerComponent,
    TimelineChartComponent,
    TimelineCornerIndicatorComponent,
    CaptureShiftKeyDirective,
    MatIconModule,
    KHIIconRegistrationModule,
    TimelineHoverOverlayComponent,
  ],
  providers: [RenderingLoopManager],
})
/**
 * TimelineFrameComponent is the main component for the timeline view.
 * It manages the layout of the timeline, including the index area, ruler, chart area, and sticky headers.
 * It also handles user interactions such as scrolling, zooming, and selection.
 *
 * The layout is managed by CSS Grid, but the scroll synchronization and virtual scrolling are handled by TypeScript logic
 * to support virtual bi-directional scrolling and efficient rendering of large datasets.
 */
export class TimelineFrameComponent implements AfterViewInit {
  protected readonly HEADER_HEIGHT = 60;
  protected readonly GUTTER_WIDTH = 8;
  protected readonly MIN_TICK_WIDTH_PX = 10;
  protected readonly MAX_HISTOGRAM_SIZE = 10000;
  protected readonly BASE_SCALE_SENSITIVITY = 0.001;
  /**
   * The detection area margin in pixels to scroll the viewport when the cursor is outside the viewport.
   * If the cursor is very close to the edge of the viewport, the viewport will scroll to keep the cursor in the viewport.
   */
  protected readonly CURSOR_SCROLL_MARGIN_IN_PX = 100;
  /**
   * The margin in pixels to scroll the viewport when the selected timeline is outside the viewport.
   * If the selected timeline is very close to the edge of the viewport, the viewport will scroll to keep the selected timeline in the viewport.
   */
  protected readonly TIMELINE_SELECTION_MARGIN_IN_PX = 100;

  private readonly ngZone = inject(NgZone);
  private readonly destroyRef = inject(DestroyRef);
  private readonly renderingLoopManager = inject(RenderingLoopManager);

  private readonly container = viewChild<ElementRef<HTMLElement>>('container');
  private readonly indexSplitArea =
    viewChild<ElementRef<HTMLElement>>('indexSplitArea');

  /**
   * Style configuration for the timeline chart area.
   */
  readonly chartStyle = input<TimelineChartStyle>(generateDefaultChartStyle());
  /**
   * Style configuration for the timeline ruler.
   */
  readonly rulerStyle = input<TimelineRulerStyle>(generateDefaultRulerStyle());
  /**
   * The number of pixels to overdraw horizontally outside the viewport.
   * Increasing this value reduces blank areas during fast scrolling but increases rendering cost.
   */
  readonly horizontalOverdrawInPx = input<number>(300);
  /**
   * The number of timelines to overdraw vertically outside the viewport.
   * Increasing this value reduces blank areas during fast scrolling but increases rendering cost.
   */
  readonly verticalOverdrawTimelineCount = input<number>(10);

  /**
   * The list of timelines to display.
   */
  readonly timelines = input<ResourceTimeline[]>([]);
  /**
   * The minimum time in milliseconds for the query range.
   * This is used as the start time for the timeline view.
   */
  readonly minQueryLogTimeMS = input<number>(0);
  /**
   * The maximum time in milliseconds for the query range.
   * This is used as the end time for the timeline view.
   */
  readonly maxQueryLogTimeMS = input<number>(0);
  /**
   * The list of all logs without filtering.
   * Used for calculating the background histogram.
   */
  readonly allLogsWithoutFilter = input<LogEntry[]>([]);
  /**
   * The list of filtered logs.
   * Used for displaying logs on the timeline.
   */
  readonly filteredLogs = input<LogEntry[]>([]);

  /**
   * The set of indices of non-filtered active logs.
   * Used for showing filtering state on the timeline.
   */
  readonly activeLogsIndices = computed(() => {
    const filteredLogs = this.filteredLogs();
    const set = new Set<number>();
    for (const log of filteredLogs) {
      set.add(log.logIndex);
    }
    return set;
  });

  /**
   * The minimum time span for a single histogram bucket.
   * Calculated based on the total time range and the maximum number of buckets.
   */
  protected readonly minTimeSpanForHistogram = computed(() => {
    const minQueryTime = this.minQueryLogTimeMS();
    const maxQueryTime = this.maxQueryLogTimeMS();
    return getMinTimeSpanForHistogram(
      this.MAX_HISTOGRAM_SIZE,
      minQueryTime,
      maxQueryTime,
    );
  });

  /**
   * Cache for the histogram of all logs.
   */
  protected readonly allLogsWithoutFilterHistogramCache = computed(() => {
    const minTimeSpanForHistogram = this.minTimeSpanForHistogram();
    const allLogsWithoutFilter = this.allLogsWithoutFilter();
    return new HistogramCache(allLogsWithoutFilter, minTimeSpanForHistogram);
  });

  /**
   * Cache for the histogram of filtered logs.
   * It shares the same time range and bucket size as the allLogsWithoutFilterHistogramCache.
   */
  protected readonly filteredLogsHistogramCache = computed(() => {
    const minTimeSpanForHistogram = this.minTimeSpanForHistogram();
    const allLogsHistogramCache = this.allLogsWithoutFilterHistogramCache();
    const filteredLogs = this.filteredLogs();
    return new HistogramCache(
      filteredLogs,
      minTimeSpanForHistogram,
      allLogsHistogramCache.logMinTimeMS,
      allLogsHistogramCache.logMaxTimeMS,
    );
  });

  /**
   * The time at the left edge of the viewport in milliseconds.
   * This is a two-way bound signal (model).
   */
  readonly viewportLeftTimeMS = model<number>(0);

  /**
   * The scale of the timeline in pixels per millisecond.
   * This is a two-way bound signal (model).
   */
  readonly pixelsPerMs = model<number>(1.0);
  /**
   * Highlights for timelines (rows).
   * Key is the timeline ID, value is the highlight type (Hovered, Selected, etc.).
   */
  readonly timelineHighlights = input<TimelineHighlight>({});

  /**
   * Highlights for items within the timeline chart (events, revisions).
   * Key is the log index, value is the highlight type.
   */
  readonly timelineChartItemHighlights = input<TimelineChartItemHighlight>({});

  /**
   * The index of the log that is currently selected.
   */
  readonly selectedLogIndex = computed(() => {
    const highlights = this.timelineChartItemHighlights();
    const findResult = Object.entries(highlights).find(([, value]) => {
      return value === TimelineChartItemHighlightType.Selected;
    });
    if (findResult === undefined) {
      return null;
    }
    const [highlightedLogIndex] = findResult;
    return highlightedLogIndex;
  });

  /**
   * Request to show a hover overlay.
   */
  readonly timelineHoverOverlayRequest =
    input<TimelineHoverOverlayRequest | null>(null);
  protected readonly timelineHoverOverlay = computed(() => {
    const request = this.timelineHoverOverlayRequest();
    return request?.overlay ?? null;
  });
  protected readonly timelineHoverOverlayOffsetX = computed(() => {
    const request = this.timelineHoverOverlayRequest();
    if (request === null) {
      return 0;
    }
    const horizontalScrollCalculator = this.horizontalScrollCalculator();
    const pixelsPerMs = this.pixelsPerMs();
    const timeMSToOffsetLeft = horizontalScrollCalculator.timeMSToOffsetLeft(
      request.timeMs,
      pixelsPerMs,
    );
    return timeMSToOffsetLeft;
  });
  protected readonly timelineHoverOverlayOffsetY = computed(() => {
    const request = this.timelineHoverOverlayRequest();
    if (request === null) {
      return 0;
    }
    const verticalScrollCalculator = this.verticalScrollCalculator();
    const timeMSToOffsetLeft =
      verticalScrollCalculator.timelineIDToTimelineBottomOffset(
        request.timelineId,
      );
    return timeMSToOffsetLeft;
  });

  /**
   * Returns the currently selected timeline based on the timelineHighlights input.
   */
  protected readonly selectedTimeline = computed(() => {
    const highlights = this.timelineHighlights();
    const timelines = this.timelines();
    const selectedHighlight = Object.entries(highlights).find(
      ([, type]) => type === TimelineHighlightType.Selected,
    );
    const selectedTimelineID = selectedHighlight ? selectedHighlight[0] : null;
    return selectedHighlight
      ? (timelines.find(
          (timeline) => timeline.timelineId === selectedTimelineID,
        ) ?? null)
      : null;
  });

  /**
   * current cursor position time in milliseconds.
   */
  readonly cursorTimeMS = input<number>(0);

  /**
   * current cursor position offset from the left of the viewport in pixels.
   */
  protected readonly cursorOffsetLeft = computed<number>(() => {
    const horizontalScrollCalculator = this.horizontalScrollCalculator();
    return horizontalScrollCalculator.timeMSToOffsetLeft(
      this.cursorTimeMS(),
      this.pixelsPerMs(),
    );
  });

  /**
   * Formatted string of the current cursor time.
   */
  protected readonly cursorTimeString = computed(() => {
    const cursorTimeMS = this.cursorTimeMS();
    const timezoneShiftHours = this.timezoneShiftHours();
    const cursorTimeDate = new Date(
      cursorTimeMS + timezoneShiftHours * 60 * 60 * 1000,
    );
    const timeString = cursorTimeDate.toISOString();
    return timeString.slice(0, timeString.length - 1);
  });

  /**
   * Emitted when the user hovers over a timeline (row).
   */
  readonly hoverOnTimeline = output<ResourceTimeline>();
  /**
   * Emitted when the user clicks on a timeline (row).
   */
  readonly clickOnTimeline = output<ResourceTimeline>();

  /**
   * Emitted when the user hovers over an item (event or revision) in the chart.
   */
  readonly hoverOnTimelineItem = output<TimelineChartMouseEvent>();
  /**
   * Emitted when the user clicks on an item (event or revision) in the chart.
   */
  readonly clickOnTimelineItem = output<TimelineChartMouseEvent>();

  /**
   * The timezone shift in hours from UTC.
   */
  readonly timezoneShiftHours = input<number>(0);
  protected readonly timezoneShiftLabel = computed(() => {
    if (this.timezoneShiftHours() >= 0) {
      return `UTC +${this.timezoneShiftHours()}`;
    } else {
      return `UTC -${this.timezoneShiftHours()}`;
    }
  });

  /**
   * Sensitivity factor for mouse wheel zooming.
   */
  readonly scrollSensitivity = input<number>(20);
  /**
   * Sensitivity factor for trackpad pinch zooming (or Ctrl + Wheel).
   */
  readonly spreadGestureSensitivity = input<number>(5);

  /**
   * ViewModel for the timeline chart area.
   * Contains only the timelines and logs that are currently visible (or within the overdraw margin).
   */
  protected readonly chartViewModel = computed<TimelineChartViewModel>(() => {
    return {
      timelinesInDrawArea: this.visibleTimelines(),
      logBeginTime: this.minQueryLogTimeMS(),
      logEndTime: this.maxQueryLogTimeMS(),
    };
  });

  /**
   * ViewModel for the sticky headers in the chart area.
   */
  protected readonly stickyChartViewModel = computed<TimelineChartViewModel>(
    () => {
      return {
        timelinesInDrawArea: this.stickyTimelines(),
        logBeginTime: this.minQueryLogTimeMS(),
        logEndTime: this.maxQueryLogTimeMS(),
      };
    },
  );

  /**
   * Interval of ticks in milliseconds for the ruler.
   */
  protected readonly tickTimeMS = computed(() => {
    return getTickTimeMS(this.pixelsPerMs(), this.MIN_TICK_WIDTH_PX);
  });

  /**
   * ViewModel for the ruler.
   */
  protected readonly rulerViewModel = computed<TimelineRulerViewModel>(() => {
    return this.rulerViewModelCalculator().generateRulerViewModel(
      this.contentLeftTime(),
      this.pixelsPerMs(),
      this.viewportWidth(),
      this.timezoneShiftHours(),
      this.allLogsWithoutFilterHistogramCache(),
      this.filteredLogsHistogramCache(),
    );
  });

  /**
   * Calculator for vertical scrolling and layout of rows.
   */
  protected readonly verticalScrollCalculator = computed(() => {
    return new VerticalScrollCalculator(
      this.timelines(),
      this.chartStyle(),
      this.verticalOverdrawTimelineCount(),
    );
  });

  /**
   * Calculator for horizontal scrolling and time-to-pixel conversion.
   */
  protected readonly horizontalScrollCalculator = computed(() => {
    return new HorizontalScrollCalculator(
      this.minQueryLogTimeMS(),
      this.maxQueryLogTimeMS(),
      this.horizontalOverdrawInPx(),
    );
  });

  private readonly rulerViewModelCalculator = computed(() => {
    return new RulerViewModelBuilder(this.horizontalOverdrawInPx());
  });

  /**
   * The width of the index area in pixels. Updated by ResizeObserver.
   */
  protected readonly indexAreaWidthPixels = signal<number>(0);
  /**
   * The total width of the container in pixels. Updated by ResizeObserver.
   */
  protected readonly containerWidth = signal(0);
  /**
   * The width of the viewport (chart area) in pixels.
   */
  protected readonly viewportWidth = computed(() => {
    return (
      this.containerWidth() - this.indexAreaWidthPixels() - this.GUTTER_WIDTH
    );
  });
  /**
   * The height of the viewport in pixels. Updated by ResizeObserver.
   */
  protected readonly viewportHeight = signal(0);
  /**
   * The current vertical scroll position of the viewport. Updated by scroll event listener.
   */
  protected readonly viewportScrollTop = signal(0);
  /**
   * Indicates whether the Shift key is currently pressed.
   * Used for switching between scrolling and zooming (when Shift is held).
   */
  protected readonly shiftStatus = signal(false);

  /**
   * Indicates whether the scale mode is enabled from the ruler.
   */
  protected readonly scaleModeFromRuler = signal(false);

  /**
   * The current scale mode.
   */
  protected readonly scaleMode = computed(() => {
    return this.scaleModeFromRuler() || this.shiftStatus();
  });

  /**
   * The list of timelines that are currently visible in the vertically scrollable viewport.
   */
  protected readonly visibleTimelines = computed(() => {
    const scrollY = this.viewportScrollTop();
    const visibleHeight = this.viewportHeight();
    return this.verticalScrollCalculator().timelinesInDrawArea(
      scrollY,
      visibleHeight,
    );
  });

  /**
   * The list of timelines that should be sticky at the top of the viewport.
   */
  protected readonly stickyTimelines = computed(() => {
    return this.verticalScrollCalculator().stickyTimelines(
      this.viewportScrollTop(),
    );
  });

  /**
   * The total height of all timelines content in pixels.
   */
  protected readonly totalContentHeight = computed(() => {
    return this.verticalScrollCalculator().totalHeight;
  });

  /**
   * The vertical offset of the visible content from the top of the container.
   * This is used to implement virtual scrolling by translating the content container.
   */
  protected readonly contentVerticalOffset = computed(() => {
    return this.verticalScrollCalculator().topDrawAreaOffset(
      this.viewportScrollTop(),
    );
  });

  /**
   * The total height of the rendered content (visible subset) in pixels.
   */
  protected readonly totalContentRenderHeight = computed(() => {
    return this.verticalScrollCalculator().totalRenderHeight(
      this.viewportHeight(),
    );
  });

  /**
   * The total width of the content in pixels if purely based on time range and scale.
   */
  protected readonly totalContentWidth = computed(() => {
    return this.horizontalScrollCalculator().totalWidth(this.pixelsPerMs());
  });
  /**
   * The width of the rendered content (horizontal subset) in pixels.
   * Currently, this is often set to match viewport width + overdraw, or handled dynamically.
   */
  protected readonly totalRenderContentWidth = computed(() => {
    return this.horizontalScrollCalculator().totalRenderWidth(
      this.viewportWidth(),
    );
  });

  /**
   * The horizontal offset of the visible content from the left of the container.
   * Used for virtual scrolling translation.
   */
  protected readonly contentHorizontalOffset = computed(() => {
    return this.horizontalScrollCalculator().leftDrawAreaOffset(
      this.viewportLeftTimeMS(),
      this.tickTimeMS(),
      this.pixelsPerMs(),
    );
  });

  /**
   * The time corresponding to the left edge of the rendered content area.
   */
  protected readonly contentLeftTime = computed(() => {
    return this.horizontalScrollCalculator().leftDrawAreaTimeMS(
      this.viewportLeftTimeMS(),
      this.tickTimeMS(),
      this.pixelsPerMs(),
    );
  });

  /**
   * Whether the user is currently grabbing the chart or not.
   */
  private readonly isGrabbing = signal(false);

  /**
   * Whether the user is currently grabbing and moving the chart or not.
   * This is needed in addition to isGrabbing not to prevent click event by applying pointer-events: none to the chart area just by mouse down event.
   */
  protected readonly isGrabbingAndMoving = signal(false);

  /**
   * The position of the last mouse down event.
   */
  private readonly lastMouseDownPosition: { x: number; y: number } = {
    x: 0,
    y: 0,
  };

  /**
   * The current action that is being performed.
   * This is defined not to move and scale at the same frame.
   */
  private currrentAction: 'moving' | 'scaling' | 'none' = 'none';

  /**
   * The source of truth for the horizontal scroll position.
   * "scroll" means the viewportLeftTimeMS property is updated by the scroll event.
   * "property" means the scroll position is updated by the viewportLeftTimeMS property.
   *
   * This property is usually kept as "property" but changed to "scroll" only when users triggers scrolling animation.
   */
  private horizontalScrollSourceOfTruth: 'scroll' | 'property' = 'property';

  constructor() {
    // Updates the scrollLeft property of the container element when the viewportLeftTimeMS changes.
    effect(() => {
      const calculator = this.horizontalScrollCalculator();
      const vpLT = this.viewportLeftTimeMS();
      const pixelsPerMs = this.pixelsPerMs();

      const container = this.container();
      if (!container) {
        console.warn(
          'container is not ready. Ignoring updating the secrollLeft property from viewportLeftTime change.',
        );
        return;
      }
      if (this.horizontalScrollSourceOfTruth === 'scroll') {
        return;
      }
      container.nativeElement.scrollLeft = calculator.timeMSToOffsetLeft(
        vpLT,
        pixelsPerMs,
      );
    });
    // Updates the viewportLeftTimeMS property and pxielsPerMs when the loaded inspection data is updated.
    effect(() => {
      const minTime = this.minQueryLogTimeMS();
      const maxTime = this.maxQueryLogTimeMS();
      const viewportWidth = untracked(this.viewportWidth);
      const overdrawX = untracked(this.horizontalOverdrawInPx);
      const drawMargin = overdrawX * 0.1; // Scroll and scale to match viewport to show 10% of margin area.
      const pixelsPerMs =
        Math.max(1, viewportWidth + 2 * drawMargin) /
        Math.max(1, maxTime - minTime);
      const viewportLeftTimeMS = minTime - drawMargin / pixelsPerMs;

      this.pixelsPerMs.set(pixelsPerMs);
      this.viewportLeftTimeMS.set(viewportLeftTimeMS);
    });

    // Updates the viewportLeftTimeMs property when the curosrTime is updated if that is outside of the viewport.
    effect(() => {
      const cursorTime = this.cursorTimeMS();
      const viewportLeftTimeMS = untracked(this.viewportLeftTimeMS);
      const viewportWidth = untracked(this.viewportWidth);
      const pixelsPerMs = untracked(this.pixelsPerMs);
      const minCursorTime =
        viewportLeftTimeMS + this.CURSOR_SCROLL_MARGIN_IN_PX / pixelsPerMs;
      const maxCursorTime =
        viewportLeftTimeMS +
        (viewportWidth - this.CURSOR_SCROLL_MARGIN_IN_PX) / pixelsPerMs;
      const logMinTime = untracked(this.minQueryLogTimeMS);
      const horizontalOverdrawInPx = untracked(this.horizontalOverdrawInPx);

      if (cursorTime < minCursorTime || cursorTime > maxCursorTime) {
        const newVPLT =
          cursorTime - this.CURSOR_SCROLL_MARGIN_IN_PX / pixelsPerMs;
        const newScrollLeft =
          (newVPLT - logMinTime) * pixelsPerMs + horizontalOverdrawInPx;
        this.horizontalScrollSourceOfTruth = 'scroll';
        this.container()?.nativeElement.scrollTo({
          left: newScrollLeft,
          behavior: 'smooth',
        });
      }
    });
    effect(() => {
      this.selectedLogIndex(); // Just for triggering the effect when the selected log index is changed.
      const selectedTimeline = this.selectedTimeline();
      if (!selectedTimeline) {
        return;
      }
      const verticalCalculator = untracked(this.verticalScrollCalculator);
      const timelineTopOffset =
        verticalCalculator.timelineIDToTimelineTopOffset(
          selectedTimeline.timelineId,
        );
      const viewportHeight = untracked(this.viewportHeight);
      const minScrollTop =
        timelineTopOffset -
        viewportHeight +
        this.TIMELINE_SELECTION_MARGIN_IN_PX;
      const maxScrollTop =
        minScrollTop +
        viewportHeight -
        2 * this.TIMELINE_SELECTION_MARGIN_IN_PX;
      const container = this.container();
      if (!container) {
        return;
      }
      const currentScrollTop = container.nativeElement.scrollTop;
      if (currentScrollTop < minScrollTop || currentScrollTop > maxScrollTop) {
        container.nativeElement.scrollTo({
          top: maxScrollTop,
          behavior: 'smooth',
        });
      }
    });
  }

  handleTimelineEventForIndex(
    e: ResourceTimeline,
    outputRef: OutputEmitterRef<ResourceTimeline>,
  ) {
    outputRef.emit(e);
  }

  handleTimelineChartItemEvent(
    e: TimelineChartMouseEvent,
    outputRef: OutputEmitterRef<TimelineChartMouseEvent>,
  ) {
    outputRef.emit(e);
  }

  handleMouseDown(e: MouseEvent) {
    const indexArea = this.indexSplitArea()?.nativeElement;
    if (!indexArea) {
      return;
    }
    const indexAreaRect = indexArea.getBoundingClientRect();
    const isChartArea = e.clientX > indexAreaRect.right + this.GUTTER_WIDTH;
    if (isChartArea) {
      this.isGrabbing.set(true);
      this.lastMouseDownPosition.x = e.clientX;
      this.lastMouseDownPosition.y = e.clientY;
    }
  }

  handleMouseUp() {
    this.isGrabbing.set(false);
    this.isGrabbingAndMoving.set(false);
  }

  handleMouseLeave() {
    this.isGrabbing.set(false);
    this.isGrabbingAndMoving.set(false);
  }

  ngAfterViewInit(): void {
    // Run outside of Angular zone to avoid unnecessary change detection by size changing or scrolls..
    // Frequent scroll events or resize events can trigger Angular's change detection if processed within the zone, leading to performance issues.
    this.ngZone.runOutsideAngular(() => {
      // Monitor resizing event of the index area.
      const container = this.container();
      if (!container) {
        throw new Error('failed to lookup container');
      }
      const indexSplitArea = this.indexSplitArea();
      if (!indexSplitArea) {
        throw new Error('failed to lookup index split area');
      }
      const resizeObserver = new ResizeObserver((entries) => {
        for (const entry of entries) {
          // Update signals inside rendering loop to ensure it runs before the next frame
          this.renderingLoopManager.registerOnceBeforeRenderHandler(() => {
            this.indexAreaWidthPixels.set(entry.contentRect.width);
          });
        }
      });
      resizeObserver.observe(indexSplitArea.nativeElement);

      // Monitor resizing event of the container and calculate the viewport height.
      const containerResizeObserver = new ResizeObserver((entries) => {
        for (const entry of entries) {
          this.renderingLoopManager.registerOnceBeforeRenderHandler(() => {
            this.viewportHeight.set(
              entry.contentRect.height - this.HEADER_HEIGHT,
            );
            this.containerWidth.set(entry.contentRect.width);
          });
        }
      });
      containerResizeObserver.observe(container.nativeElement);

      // Handle wheel and scroll events from container.
      // Wheel events assigned to sticky element may not emit wheel event(?), so we handle it here.
      const onContainerWheel = (event: WheelEvent) => {
        const containerBox = container.nativeElement.getBoundingClientRect();
        const indexAreaBox =
          indexSplitArea.nativeElement.getBoundingClientRect();
        const x = event.clientX - containerBox.left;
        // Ignore events on the index area (left side)
        if (x < indexAreaBox.width + this.GUTTER_WIDTH) {
          return;
        }
        // Handle zooming if Shift key is pressed or Ctrl key is pressed (pinch gesture)
        if (this.shiftStatus() || event.ctrlKey) {
          event.preventDefault();
          this.onWheelForScaling(event);
        }
      };
      container.nativeElement.addEventListener('wheel', onContainerWheel, {
        passive: false,
      });

      const onContainerScroll = () => {
        if (this.shiftStatus() || this.currrentAction !== 'none') {
          return;
        }
        this.onScrollForMove();
      };
      container.nativeElement.addEventListener('scroll', onContainerScroll, {
        passive: true,
      });

      const onScrollEnd = () => {
        this.horizontalScrollSourceOfTruth = 'property';
      };
      container.nativeElement.addEventListener('scrollend', onScrollEnd);

      const onMouseMove = (e: MouseEvent) => {
        if (!this.isGrabbing()) {
          return;
        }
        const dx = e.clientX - this.lastMouseDownPosition.x;
        const dy = e.clientY - this.lastMouseDownPosition.y;
        this.lastMouseDownPosition.x = e.clientX;
        this.lastMouseDownPosition.y = e.clientY;
        this.isGrabbingAndMoving.set(true);
        this.renderingLoopManager.registerOnceBeforeRenderHandler(() => {
          container.nativeElement.scrollBy({
            left: -dx,
            top: -dy,
          });
        });
      };
      window.addEventListener('mousemove', onMouseMove, { passive: true });

      this.destroyRef.onDestroy(() => {
        resizeObserver.disconnect();
        containerResizeObserver.disconnect();
        container.nativeElement.removeEventListener('wheel', onContainerWheel);
        container.nativeElement.removeEventListener(
          'scroll',
          onContainerScroll,
        );
        container.nativeElement.removeEventListener('scrollend', onScrollEnd);
        window.removeEventListener('mousemove', onMouseMove);
      });
    });

    this.renderingLoopManager.start(this.ngZone, this.destroyRef);
  }

  onScrollForMove() {
    const container = this.container();
    if (!container) {
      throw new Error('failed to lookup container');
    }
    const horizontalScrollCalculator = this.horizontalScrollCalculator();
    this.currrentAction = 'moving';
    this.renderingLoopManager.registerOnceBeforeRenderHandler(() => {
      this.currrentAction = 'none';
      const pixelsPerMS = this.pixelsPerMs();
      const maxScrollLeft = horizontalScrollCalculator.maxScrollLeft(
        pixelsPerMS,
        this.viewportWidth(),
      );
      const scrollLeft = Math.min(
        container.nativeElement.scrollLeft,
        maxScrollLeft,
      );
      this.viewportScrollTop.set(container.nativeElement.scrollTop);
      this.viewportLeftTimeMS.set(
        horizontalScrollCalculator.scrollToViewportLeftTime(
          scrollLeft,
          pixelsPerMS,
        ),
      );
    });
  }

  onWheelForScaling(event: WheelEvent) {
    if (this.currrentAction !== 'none') return;
    this.currrentAction = 'scaling';
    this.renderingLoopManager.registerOnceBeforeRenderHandler(() => {
      const container = this.container();
      const indexArea = this.indexSplitArea();
      if (!container || !indexArea) {
        this.currrentAction = 'none';
        return;
      }
      const containerBox = container.nativeElement.getBoundingClientRect();
      const indexAreaBox = indexArea.nativeElement.getBoundingClientRect();
      const viewportRelativeMousePosition =
        event.clientX -
        containerBox.left -
        indexAreaBox.width -
        this.GUTTER_WIDTH;
      this.currrentAction = 'none';
      const calculator = this.horizontalScrollCalculator();

      const currentPixelsPerMs = this.pixelsPerMs();
      // Zoom factor: 1.001 per delta unit.
      // -deltaY because negative deltaY (scroll up) usually means zoom in.
      const vpWidth = this.viewportWidth();
      let scaleSensitivity =
        this.BASE_SCALE_SENSITIVITY * this.scrollSensitivity();
      if (event.ctrlKey) {
        scaleSensitivity *= this.spreadGestureSensitivity();
      }
      const newPixelsPerMs = Math.min(
        calculator.maxPixelPerMs(),
        Math.max(
          calculator.minPixelPerMs(vpWidth),
          currentPixelsPerMs *
            Math.pow(1 + scaleSensitivity, -Math.sign(event.deltaY)), // Only checks the sign of deltaY because the amount is completly different by the platform. https://developer.mozilla.org/en-US/docs/Web/API/Element/mousewheel_event#chrome
        ),
      );

      // Calculate new scroll position to keep the mouse pointer time consistent
      const newScrollLeft = calculator.calculateZoomScrollLeft(
        currentPixelsPerMs,
        newPixelsPerMs,
        viewportRelativeMousePosition,
        container.nativeElement.scrollLeft,
      );
      this.viewportScrollTop.set(container.nativeElement.scrollTop);
      this.viewportLeftTimeMS.set(
        calculator.scrollToViewportLeftTime(newScrollLeft, newPixelsPerMs),
      );
      this.pixelsPerMs.set(newPixelsPerMs);
    });
  }
}
