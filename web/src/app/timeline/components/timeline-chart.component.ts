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
  DestroyRef,
  effect,
  ElementRef,
  inject,
  input,
  NgZone,
  output,
  OutputEmitterRef,
  signal,
  viewChild,
} from '@angular/core';
import { GLContextManager } from './canvas/glcontextmanager';
import { TimelineBackgroundRenderer } from './canvas/timeline-background-renderer';
import {
  TimelineRenderer,
  TimelineRendererRenderArgs,
} from './canvas/timeline-renderer';
import { HitTestResult } from './canvas/hittest-shared-resource';
import { RenderingLoopManager } from './canvas/rendering-loop-manager';
import { TimelineRulerViewModel } from './timeline-ruler.viewmodel';
import { TimelineChartViewModel } from './timeline-chart.viewmodel';
import {
  TimelineChartItemHighlight,
  TimelineHighlight,
} from './interaction-model';
import {
  TimelineRulerStyle,
  generateDefaultRulerStyle,
  TimelineChartStyle,
  generateDefaultChartStyle,
} from './style-model';

/**
 * Represents a mouse event that occurred on the timeline chart, including hit test results.
 */
export interface TimelineChartMouseEvent extends HitTestResult {
  /**
   * The original DOM MouseEvent.
   */
  event: MouseEvent;
  /**
   * The time in milliseconds corresponding to the mouse position.
   */
  timeMS: number;
}

/**
 * The `TimelineChartComponent` renders the main timeline visualization, including the background grid,
 * timeline rows, events, and revisions. It uses a hybrid rendering approach with 2D Canvas for the background
 * and WebGL for the high-performance timeline content.
 */
@Component({
  selector: 'khi-timeline-chart',
  templateUrl: './timeline-chart.component.html',
  styleUrls: ['./timeline-chart.component.scss'],
  imports: [],
})
export class TimelineChartComponent implements AfterViewInit {
  private readonly container =
    viewChild<ElementRef<HTMLDivElement>>('container');
  private readonly background2DCanvas =
    viewChild<ElementRef<HTMLCanvasElement>>('background2DCanvas');
  private readonly glCanvas =
    viewChild<ElementRef<HTMLCanvasElement>>('glCanvas');

  private readonly ngZone = inject(NgZone);

  private readonly renderingLoopManager = inject(RenderingLoopManager);

  private readonly destroyRef = inject(DestroyRef);

  private readonly timelineRenderer = new TimelineRenderer();

  /**
   * Configuration for the timeline ruler style.
   */
  readonly rulerStyle = input<TimelineRulerStyle>(generateDefaultRulerStyle());

  /**
   * The view model data for the timeline ruler, containing ticks and labels.
   */
  readonly rulerViewModel = input<TimelineRulerViewModel>();

  /**
   * Configuration for the timeline chart style.
   */
  readonly chartStyle = input<TimelineChartStyle>(generateDefaultChartStyle());

  /**
   * The view model data for the timeline chart, containing timeline rows and items.
   */
  readonly chartViewModel = input<TimelineChartViewModel>();

  /**
   * Highlights for specific time ranges or timeline rows.
   */
  readonly timelineHighlights = input<TimelineHighlight>({});

  /**
   * The current time at the left edge of the viewport (scroll position) in milliseconds.
   */
  readonly leftEdgeTime = input<number>(0);

  /**
   * The current zoom level in pixels per millisecond.
   */
  readonly pixelsPerMs = input<number>(1);

  /**
   * A set of log indices that are currently active (e.g., matching a filter).
   * Inactive logs may be rendered differently (e.g., dimmed).
   */
  readonly activeLogsIndices = input<Set<number>>(new Set());

  /**
   * Highlights for specific items (logs/events) within the timeline.
   */
  readonly timelineChartItemHighlights = input<TimelineChartItemHighlight>({});

  /**
   * Emitted when the mouse moves over a timeline item.
   */
  readonly mouseMoveOnTimelineItem = output<TimelineChartMouseEvent>();

  /**
   * Emitted when a timeline item is clicked.
   */
  readonly clickOnTimelineItem = output<TimelineChartMouseEvent>();

  /**
   * Flag to indicate that the timeline needs to be redrawn.
   */
  private readonly invalidate = signal(true);

  private resizeObserver: ResizeObserver | null = null;

  private contextManager!: GLContextManager<TimelineRendererRenderArgs>;

  private backgroundRenderer!: TimelineBackgroundRenderer;

  constructor() {
    effect(() => {
      this.updateRendererParams();
    });
  }

  async ngAfterViewInit(): Promise<void> {
    const bgCanvas = this.background2DCanvas()!.nativeElement;
    const bgContext = bgCanvas.getContext('2d', { colorSpace: 'display-p3' });
    if (!bgContext) {
      throw new Error('Failed to get 2D context');
    }
    this.backgroundRenderer = new TimelineBackgroundRenderer(bgContext);
    this.contextManager = new GLContextManager<TimelineRendererRenderArgs>(
      this.glCanvas()!.nativeElement,
      this.timelineRenderer,
    );
    await this.contextManager.setup();
    this.destroyRef.onDestroy(() => {
      this.contextManager.dispose();
    });

    this.resizeObserver = new ResizeObserver(() => {
      this.ngZone.runOutsideAngular(() => {
        this.handleResize();
      });
    });
    this.resizeObserver.observe(this.container()!.nativeElement);
    this.destroyRef.onDestroy(() => {
      this.resizeObserver?.disconnect();
    });

    this.updateRendererParams();
    this.renderingLoopManager.registerRenderHandler(this.destroyRef, () => {
      if (!this.invalidate()) {
        return;
      }
      this.contextManager.render({
        leftEdgeTime: this.leftEdgeTime(),
        pixelsPerMs: this.pixelsPerMs(),
      });
      this.backgroundRenderer.render(this.leftEdgeTime(), this.pixelsPerMs());
      this.invalidate.set(false);
    });
  }

  /**
   * Handles mouse events on the timeline container.
   * Performs hit testing against the WebGL content and emits corresponding events.
   *
   * @param e The native MouseEvent.
   * @param event The output emitter to emit the result to.
   */
  protected onMouseEvent(
    e: MouseEvent,
    event: OutputEmitterRef<TimelineChartMouseEvent>,
  ) {
    const container = this.container()!.nativeElement;
    const rect = container.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const timeMS = this.leftEdgeTime() + x / this.pixelsPerMs();
    this.timelineRenderer.hittest(x, y).then((result) => {
      event.emit({
        event: e,
        ...result,
        timeMS,
      });
    });
    this.invalidate.set(true); // hittest needs redraw
  }

  private handleResize() {
    const container = this.container()!.nativeElement;
    const rect = container.getBoundingClientRect();
    const canvas = this.background2DCanvas()!.nativeElement;
    const glCanvas = this.glCanvas()!.nativeElement;
    const dpr = window.devicePixelRatio || 1;

    canvas.style.width = `${rect.width}px`;
    canvas.style.height = `${rect.height}px`;
    glCanvas.style.width = `${rect.width}px`;
    glCanvas.style.height = `${rect.height}px`;

    // Changing actual canvas size or renderer size may clear the current canvas and cause flickering effect.
    // Delay changing the actual size until the next render.
    this.renderingLoopManager.registerOnceBeforeRenderHandler(() => {
      canvas.width = rect.width * dpr;
      canvas.height = rect.height * dpr;
      glCanvas.width = rect.width * dpr;
      glCanvas.height = rect.height * dpr;
      this.backgroundRenderer.resize(rect.width, rect.height, dpr);
      this.timelineRenderer.resize(rect.width, rect.height, dpr);
      this.invalidate.set(true);
    });
  }

  private updateRendererParams() {
    const rulerViewModel = this.rulerViewModel();
    const rulerStyle = this.rulerStyle();
    const chartViewModel = this.chartViewModel();
    const chartStyle = this.chartStyle();
    const timelineHighlights = this.timelineHighlights();
    const logElementHighlights = this.timelineChartItemHighlights();
    const activeLogsIndices = this.activeLogsIndices();
    this.invalidate.set(true);
    if (
      rulerViewModel === undefined ||
      rulerStyle === undefined ||
      chartViewModel === undefined ||
      chartStyle === undefined ||
      this.backgroundRenderer === undefined
    ) {
      return;
    }
    this.backgroundRenderer.update(
      rulerViewModel,
      chartViewModel,
      rulerStyle,
      chartStyle,
      timelineHighlights,
    );
    this.timelineRenderer.update(
      chartViewModel,
      chartStyle,
      logElementHighlights,
      activeLogsIndices,
    );
  }
}
