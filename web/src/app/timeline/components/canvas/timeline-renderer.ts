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

import { ResourceTimeline } from 'src/app/store/timeline';
import { GLRenderer } from './glcontextmanager';
import {
  TimelineRevisionsRenderer,
  TimelineRevisionsSharedResources,
} from './timeline-revisions-renderer';
import { LRUCache } from 'src/app/common/lru-cache';
import { TimelineRendererSharedResource } from './timeline-shared-resource';
import {
  HitTestResult,
  TimelineHitTestSharedResource,
} from './hittest-shared-resource';
import {
  TimelineEventsRenderer,
  TimelineEventsSharedResources,
} from './timeline-events-renderer';
import { TimelineChartViewModel } from '../timeline-chart.viewmodel';
import { TimelineChartItemHighlight } from '../interaction-model';
import { TimelineChartStyle } from '../style-model';

/**
 * Interface for renderers that can be disposed.
 */
export interface IDisposableRenderer {
  dispose(gl: WebGL2RenderingContext): void;
}

export interface TimelineRect {
  dpr: number;
  offsetY: number;
  width: number;
  height: number;
}

interface HitTestRequest {
  x: number;
  y: number;
  resolver: (result: HitTestResult) => void;
}

export interface TimelineRendererRenderArgs {
  leftEdgeTime: number;
  pixelsPerMs: number;
}

/**
 * The main renderer for the timeline chart.
 * It manages the WebGL rendering of timeline revisions and events, handling hit testing and resource management.
 */
export class TimelineRenderer implements GLRenderer<TimelineRendererRenderArgs> {
  /**
   * Maximum number of renderers to keep in cache.
   * This limits the memory usage for rendering resources (VAOs/VBOs).
   */
  private readonly MAX_RENDERER_ROW_COUNT = 400;
  public width = 0;
  public height = 0;
  public dpr = 1;

  chartViewModel: TimelineChartViewModel | null = null;
  chartStyle: TimelineChartStyle | null = null;

  /**
   * Resources shared across all timeline renderers (view state, common textures).
   */
  private timelineSharedResource: TimelineRendererSharedResource =
    new TimelineRendererSharedResource();

  /**
   * Shared resources specific to revision rendering (shaders, styles).
   */
  private revisionSharedResource: TimelineRevisionsSharedResources =
    new TimelineRevisionsSharedResources(this.timelineSharedResource);

  /**
   * Shared resources specific to event rendering (shaders, styles).
   */
  private eventSharedResource: TimelineEventsSharedResources =
    new TimelineEventsSharedResources();

  /**
   * Shared resources and logic for hit testing.
   */
  private hittestSharedResource: TimelineHitTestSharedResource =
    new TimelineHitTestSharedResource();

  /**
   * LRU chache for revision renderers allowing efficient reuse of VAOs/VBOs for timelines.
   */
  private revisionRenderers!: LRUCache<string, TimelineRevisionsRenderer>;

  /**
   * LRU cache for event renderers allowing efficient reuse of VAOs/VBOs for timelines.
   */
  private eventRenderers!: LRUCache<string, TimelineEventsRenderer>;

  /**
   * Queue of renderers that need to be disposed (GPU resources freed) when evicted from cache.
   */
  private disposeQueue: IDisposableRenderer[] = [];

  /**
   * Queue of pending hit test requests causing by user interaction.
   */
  private hitTestRequests: HitTestRequest[] = [];

  /**
   * Current highlight state of log elements (e.g. selected, hovered).
   */
  private logElementHighlights: TimelineChartItemHighlight = {};

  /**
   * Current filter state of log elements.
   * When activeLogsIndices not includes a log index, then the log must be shown as disabled.
   */
  private activeLogsIndices: Set<number> = new Set();

  private highlightUpdated = false;

  /**
   * Sets up the WebGL resources for the renderer.
   * Initializes shared resources and clears renderers cache.
   *
   * @param gl The WebGL2 rendering context.
   */
  async setup(gl: WebGL2RenderingContext): Promise<void> {
    await this.timelineSharedResource.setup(gl);
    await this.revisionSharedResource.setup(gl);
    await this.eventSharedResource.setup(gl);
    this.hittestSharedResource.setup(gl);
    this.revisionRenderers = new LRUCache(
      this.MAX_RENDERER_ROW_COUNT,
      (renderer) => this.disposeQueue.push(renderer),
    ); // This must be reinitialized in setup() to restore resources after context lost
    this.eventRenderers = new LRUCache(
      this.MAX_RENDERER_ROW_COUNT,
      (renderer) => this.disposeQueue.push(renderer),
    ); // This must be reinitialized in setup() to restore resources after context lost
  }

  /**
   * Renders the timeline chart including revisions and events.
   * Also processes pending hit test requests.
   *
   * @param gl The WebGL2 rendering context.
   * @param args Rendering arguments including time range and zoom level.
   */
  render(gl: WebGL2RenderingContext, args: TimelineRendererRenderArgs): void {
    if (!this.chartViewModel || !this.chartStyle) return;
    gl.viewport(0, 0, this.width, this.height);
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
    this.timelineSharedResource.beforeRender(gl, {
      width: this.width,
      height: this.height,
      devicePixelRatio: this.dpr,
      pixelsPerMs: args.pixelsPerMs,
      leftEdgeTime: args.leftEdgeTime,
    });
    this.revisionSharedResource.beforeRender(gl);
    this.eventSharedResource.beforeRender(gl);
    if (this.highlightUpdated) {
      this.iterateRevisionRenderers(gl, (r) =>
        r.updateDynamicBuffer(
          gl,
          this.logElementHighlights,
          this.activeLogsIndices,
        ),
      );
      this.iterateEventRenderers(gl, (r) =>
        r.updateDynamicBuffer(
          gl,
          this.logElementHighlights,
          this.activeLogsIndices,
        ),
      );
      this.highlightUpdated = false;
    }
    this.iterateRevisionRenderers(gl, (r, rect) => r.renderColor(gl, rect));
    this.iterateEventRenderers(gl, (r, rect) => r.renderColor(gl, rect));

    if (this.hitTestRequests.length > 0) {
      this.processHitTestRequest(gl);
    }
    gl.finish();
  }

  /**
   * Resizes the renderer surface.
   *
   * @param width The width of the canvas in pixels.
   * @param height The height of the canvas in pixels.
   * @param devicePixelRatio The device pixel ratio.
   */
  resize(width: number, height: number, devicePixelRatio: number): void {
    this.width = width;
    this.height = height;
    this.dpr = devicePixelRatio;
    this.hittestSharedResource.resize(width, height);
  }

  /**
   * Updates the chart view model and style, triggering a re-render/highlight update.
   *
   * @param chartViewModel The new view model to render.
   * @param chartStyle The visual style configuration.
   * @param logElementHighlights The set of highlighted log elements.
   */
  update(
    chartViewModel: TimelineChartViewModel,
    chartStyle: TimelineChartStyle,
    logElementHighlights: TimelineChartItemHighlight,
    activeLogsIndices: Set<number>,
  ) {
    this.chartViewModel = chartViewModel;
    this.chartStyle = chartStyle;
    this.revisionSharedResource.updateChartStyle(chartStyle);
    this.eventSharedResource.updateChartStyle(chartStyle);
    this.logElementHighlights = logElementHighlights;
    this.highlightUpdated = true;
    this.activeLogsIndices = activeLogsIndices;
  }

  /**
   * Requests a hit test at the specified coordinates.
   *
   * @param x The x-coordinate for the hit test.
   * @param y The y-coordinate for the hit test.
   * @returns A promise that resolves to the hit test result.
   */
  hittest(x: number, y: number): Promise<HitTestResult> {
    return new Promise((resolve) => {
      this.hitTestRequests.push({
        x: x,
        y: y,
        resolver: resolve,
      });
    });
  }

  /**
   * processHitTestRequest draws IDs map texture and processes hit test requests.
   * It renders a simplified version of the scene to a texture where colors represent IDs,
   * then reads back pixel data to identify intersected objects.
   *
   * @param gl The WebGL2 rendering context.
   */
  private processHitTestRequest(gl: WebGL2RenderingContext) {
    if (!this.chartViewModel || !this.chartStyle) return;
    this.hittestSharedResource.beforeRender(gl);
    this.iterateRevisionRenderers(gl, (r, rect) => {
      rect.dpr = 1.0; // hit test buffer is rendered in without considering dpr
      r.renderHittest(gl, rect);
    });
    this.iterateEventRenderers(gl, (r, rect) => {
      rect.dpr = 1.0; // hit test buffer is rendered in without considering dpr
      r.renderHittest(gl, rect);
    });
    this.hittestSharedResource.afterRender(gl);
    for (const request of this.hitTestRequests) {
      let offsetY = 0;
      let hit = false;
      for (let i = 0; i < this.chartViewModel.timelinesInDrawArea.length; i++) {
        const tl = this.chartViewModel.timelinesInDrawArea[i];
        const height = this.chartStyle.heightsByLayer[tl.layer];
        if (request.y < offsetY + height) {
          const result = this.hittestSharedResource.hittest(
            gl,
            request.x,
            request.y,
            tl,
          );
          request.resolver(result);
          hit = true;
          break;
        }
        offsetY += height;
      }
      if (!hit) {
        request.resolver({
          timeline: null,
          clientX: request.x,
          clientY: request.y,
        });
      }
    }
    this.hitTestRequests = [];
  }

  /**
   * Iterates over all timelines in the current drawing area and executes a callback for each.
   * It calculates the drawing rectangle for each timeline layer.
   *
   * @param onRender Callback to execute for each visible timeline.
   */
  private renderItems(
    onRender: (t: ResourceTimeline, rect: TimelineRect) => void,
  ) {
    if (!this.chartViewModel || !this.chartStyle) return;
    const drawRect: TimelineRect = {
      dpr: this.dpr,
      offsetY: this.height,
      width: this.width,
      height: 0,
    };
    for (const t of this.chartViewModel.timelinesInDrawArea) {
      drawRect.height = this.chartStyle.heightsByLayer[t.layer];
      drawRect.offsetY -= drawRect.height;
      onRender(t, drawRect);
    }
  }

  /**
   * Iterates through visible timelines and renders their revisions.
   *
   * @param gl The WebGL2 rendering context.
   * @param render Callback to perform the actual rendering.
   */
  private iterateRevisionRenderers(
    gl: WebGL2RenderingContext,
    render: (r: TimelineRevisionsRenderer, rect: TimelineRect) => void,
  ) {
    this.renderItems((t, rect) => {
      const revisionRenderer = this.ensureRevisionRenderer(gl, t);
      render(revisionRenderer, rect);
    });
  }

  /**
   * Iterates through visible timelines and renders their events.
   *
   * @param gl The WebGL2 rendering context.
   * @param render Callback to perform the actual rendering.
   */
  private iterateEventRenderers(
    gl: WebGL2RenderingContext,
    render: (r: TimelineEventsRenderer, rect: TimelineRect) => void,
  ) {
    this.renderItems((t, rect) => {
      const eventRenderer = this.ensureEventRenderer(gl, t);
      render(eventRenderer, rect);
    });
  }

  /**
   * Retrieves or creates a revision renderer for a specific timeline.
   *
   * @param gl The WebGL2 rendering context.
   * @param t The resource timeline.
   * @returns The revision renderer instance.
   */
  private ensureRevisionRenderer(
    gl: WebGL2RenderingContext,
    t: ResourceTimeline,
  ): TimelineRevisionsRenderer {
    let renderer = this.revisionRenderers.get(t.resourcePath);
    if (renderer) {
      return renderer;
    }
    renderer = new TimelineRevisionsRenderer(
      t,
      this.revisionSharedResource,
      this.timelineSharedResource,
    );
    renderer.setup(gl);
    this.revisionRenderers.put(t.resourcePath, renderer);
    return renderer;
  }

  /**
   * Retrieves or creates an event renderer for a specific timeline.
   *
   * @param gl The WebGL2 rendering context.
   * @param t The resource timeline.
   * @returns The event renderer instance.
   */
  private ensureEventRenderer(
    gl: WebGL2RenderingContext,
    t: ResourceTimeline,
  ): TimelineEventsRenderer {
    let renderer = this.eventRenderers.get(t.resourcePath);
    if (renderer) {
      return renderer;
    }
    renderer = new TimelineEventsRenderer(
      t,
      this.eventSharedResource,
      this.timelineSharedResource,
    );
    renderer.setup(gl);
    this.eventRenderers.put(t.resourcePath, renderer);
    return renderer;
  }
}
