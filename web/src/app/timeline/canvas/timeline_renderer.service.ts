/**
 * Copyright 2024 Google LLC
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

import { Injectable } from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  ReplaySubject,
  Subject,
  combineLatest,
  debounceTime,
  filter,
  map,
  share,
  startWith,
  withLatestFrom,
} from 'rxjs';
import {
  CanvasMouseLocation,
  CanvasSize,
  GLRect,
  TIMELINE_ITEM_HEIGHTS,
  TimelineMouseLocation,
} from './types';
import {
  PerRowScrollingProperty,
  TimelinesScrollStrategy,
} from '../TimelineScrollStrategy';
import { ListRange } from '@angular/cdk/collections';
import { InspectionDataStoreService } from 'src/app/services/inspection-data-store.service';
import { ViewStateService } from 'src/app/services/view-state.service';
import { SharedGLResources } from './shared_gl_resource';
import { SelectionManagerService } from 'src/app/services/selection-manager.service';
import { TimelineRowWebGLRenderer } from './timeline_gl_row_renderer';
import { GLVerticalLineRenderer } from './gl_vertical_line_renderer';
import { TimeRange } from 'src/app/store/inspection-data';
import { LogEntry } from 'src/app/store/log';
import { ResourceTimeline } from 'src/app/store/timeline';
import { TimelineGLResourceManager } from './timeline_gl_resource_manager';

@Injectable()
export class TimelineRendererService {
  /**
   * The time in ms not to refresh screen when the previous render request was happened recently.
   */
  private static MINIMUM_REFRESH_PERIOD = 1;

  private static EVENT_HIT_CHECK_Y_PADDING = 0.2;

  private static HOVER_CURSOR_DISTANCE_PX = 10;

  private static PRELOAD_GL_RESOURCE_AREA = 10;

  private _canvasSize: Observable<CanvasSize> | null = null;
  private get canvasSize(): Observable<CanvasSize> {
    if (!this._canvasSize)
      throw new Error('timeline canvas is not initialzied');
    return this._canvasSize;
  }

  private canvasWrapper: HTMLElement | null = null;

  private currentCanvas: HTMLCanvasElement | null = null;

  private sharedGLResources: SharedGLResources | null = null;

  private selectedTimeVerticalLineRenderer: GLVerticalLineRenderer | null =
    null;

  /**
   * Whether GL related resources are under initializing or not. This can be false after the first initialization because WebGL context may lost and require re-initializing.
   */
  private initializingGLResources = false;

  private glContext = new ReplaySubject<WebGL2RenderingContext>(1);

  private timelineGLResourceManager = new TimelineGLResourceManager();

  /**
   * Emits mouse pointer events on this canvas at the pointer movement related events. Emits null when the mouse went away from this canvas.
   */
  private mousePointerLocation: Subject<CanvasMouseLocation | null> =
    new BehaviorSubject<CanvasMouseLocation | null>(null);

  /**
   * Emits mouse pointer events on this canvas at the click event. Emits null at first.
   */
  private mouseClickPointerLocation: Subject<CanvasMouseLocation | null> =
    new BehaviorSubject<CanvasMouseLocation | null>(null);

  constructor(
    private dataStore: InspectionDataStoreService,
    private selectionManager: SelectionManagerService,
    private scrollingStrategy: TimelinesScrollStrategy,
    private viewState: ViewStateService,
  ) {
    this.initSubscribers();
  }

  /**
   * Emits current timeline at the last pointer hovering event.
   */
  mousePointerTimeline = this.mousePointerLocation.pipe(
    withLatestFrom(
      this.viewState.timeOffset,
      this.viewState.pixelPerTime,
      this.scrollingStrategy.visibleTimelines,
      this.scrollingStrategy.stickyTimelines,
    ),
    map(([p, timeOffset, pixelPerTime, timelines, stickyTimelines]) => {
      return this.findTimelineFromPosition(
        p,
        timeOffset,
        pixelPerTime,
        timelines,
        stickyTimelines,
      );
    }),
  );

  /**
   * Emits current timeline at the last pointer click event.
   */
  mousePointerClickTimeline = this.mouseClickPointerLocation.pipe(
    withLatestFrom(
      this.viewState.timeOffset,
      this.viewState.pixelPerTime,
      this.scrollingStrategy.visibleTimelines,
      this.scrollingStrategy.stickyTimelines,
    ),
    map(([p, timeOffset, pixelPerTime, timelines, stickyTimelines]) => {
      return this.findTimelineFromPosition(
        p,
        timeOffset,
        pixelPerTime,
        timelines,
        stickyTimelines,
      );
    }),
  );

  private rowRenderers = combineLatest([
    this.glContext,
    this.dataStore.allTimelines,
    this.dataStore.$timeRange,
  ]).pipe(
    map(([gl, timelines, { begin }]) => {
      const rendererMap = new Map<ResourceTimeline, TimelineRowWebGLRenderer>();
      timelines.forEach((t) => {
        rendererMap.set(
          t,
          new TimelineRowWebGLRenderer(
            gl,
            this.timelineGLResourceManager,
            this.sharedGLResources!,
            t,
            begin,
          ),
        );
      });
      return rendererMap;
    }),
    startWith(new Map<ResourceTimeline, TimelineRowWebGLRenderer>()),
    share(),
  );

  private initSubscribers() {
    // Mouse pointer handler for mouse move related events
    this.mousePointerTimeline
      .pipe(
        withLatestFrom(
          this.viewState.pixelPerTime,
          this.viewState.devicePixelRatio,
        ),
      )
      .subscribe(([t, pixelPerTime, pixelRatio]) => {
        let nextHighlightLogs: number[] = [];
        const queryRange =
          (TimelineRendererService.HOVER_CURSOR_DISTANCE_PX * pixelRatio) /
          pixelPerTime;
        if (t?.timeline) {
          // Try finding event nearby if the mouse pointer was around center of the timeline
          // Fallback to find covering revision when it couldn't find anything.
          if (
            t.y >= TimelineRendererService.EVENT_HIT_CHECK_Y_PADDING &&
            t.y <= 1.0 - TimelineRendererService.EVENT_HIT_CHECK_Y_PADDING
          ) {
            const events = t.timeline.queryEventsInRange(
              t.time - queryRange,
              t.time + queryRange,
            );
            nextHighlightLogs = events.map((e) => e.logIndex);
          }
          if (!nextHighlightLogs.length) {
            const revisions = t.timeline.queryRevisionsInRange(
              t.time - queryRange,
              t.time + queryRange,
            );
            nextHighlightLogs = revisions.map((r) => r.logIndex);
          }
        }
        this.selectionManager.onHighlightLog(nextHighlightLogs);
        this.selectionManager.onHighlightTimeline(t?.timeline ?? null);
      });

    // Mouse click event handler
    this.mousePointerClickTimeline
      .pipe(
        withLatestFrom(
          this.viewState.pixelPerTime,
          this.viewState.devicePixelRatio,
        ),
      )
      .subscribe(([t, pixelPerTime, pixelRatio]) => {
        let eventSelected = false;
        const queryRange =
          (TimelineRendererService.HOVER_CURSOR_DISTANCE_PX * pixelRatio) /
          pixelPerTime;
        if (t?.timeline) {
          if (
            t.y >= TimelineRendererService.EVENT_HIT_CHECK_Y_PADDING &&
            t.y <= 1.0 - TimelineRendererService.EVENT_HIT_CHECK_Y_PADDING
          ) {
            const event = t.timeline.pickEventNearCenterOfRange(
              t.time - queryRange,
              t.time + queryRange,
            );
            if (event) {
              eventSelected = true;
              this.selectionManager.changeSelectionByEvent(t.timeline, event);
              return;
            }
          }
          if (!eventSelected) {
            const revision = t.timeline.getLatestRevisionOfTime(t.time);
            if (revision) {
              this.selectionManager.changeSelectionByRevision(
                t.timeline,
                revision,
              );
              return;
            }
          }
        }
        this.selectionManager.onSelectTimeline(t?.timeline ?? null);
      });
  }

  public attach(
    canvasWrapper: HTMLElement,
    canvasSizeObserver: Observable<CanvasSize>,
  ) {
    this.canvasWrapper = canvasWrapper;
    this._canvasSize = canvasSizeObserver.pipe(
      filter((v) => v.height > 0 && v.width > 0),
      debounceTime(TimelineRendererService.MINIMUM_REFRESH_PERIOD),
      share(),
    );
    this.refleshGL();

    // Redraw timelines in the response to the depending property change
    combineLatest([
      this.glContext,
      this.canvasSize,
      this.scrollingStrategy.offsetToFirstItemFromVisibleArea,
      this.scrollingStrategy.stickyTimelines,
      this.scrollingStrategy.visibleItemRange,
      this.scrollingStrategy.perRowScrollingProperties,
      this.rowRenderers,
      this.dataStore.$timeRange,
      this.selectionManager.selectedTimeline,
      this.selectionManager.highlightedTimeline,
      this.selectionManager.highlightedChildrenOfSelectedTimeline,
      this.dataStore.allLogs,
      this.selectionManager.selectedLogIndex,
      this.selectionManager.highlightLogIndices,
      this.dataStore.filteredOutLogIndicesSet,
      this.viewState.pixelPerTime,
      this.viewState.timeOffset,
      this.viewState.devicePixelRatio,
    ]).subscribe(
      ([
        gl,
        canvasSize,
        offsetToFirstItemFromVisibleArea,
        stickyTimelines,
        visibleItemRange,
        perRowScrollingProperties,
        rowRenderers,
        timeRange,
        selectedTimeline,
        highlightedTimeline,
        highlightedByParent,
        logs,
        selectedLog,
        highlightLogs,
        filteredLogs,
        pixelPerTime,
        timeOffset,
        pixelRatio,
      ]) => {
        this.onRender(
          gl,
          canvasSize,
          offsetToFirstItemFromVisibleArea,
          visibleItemRange,
          perRowScrollingProperties,
          rowRenderers,
          timeRange,
          stickyTimelines,
          selectedTimeline,
          highlightedTimeline,
          highlightedByParent,
          logs,
          selectedLog,
          filteredLogs,
          highlightLogs,
          pixelPerTime,
          timeOffset,
          pixelRatio,
        );
      },
    );
  }

  public async refleshGL() {
    if (this.canvasWrapper === null) {
      throw new Error(
        'canvasWrapper reference must be initialized before initializing WebGL',
      );
    }
    if (this.initializingGLResources) {
      return;
    }
    if (this.currentCanvas !== null) {
      this.currentCanvas.remove();
      this.currentCanvas = null;
    }
    this.initializingGLResources = true;
    const canvas = document.createElement('canvas');
    this.canvasWrapper.appendChild(canvas);
    this.canvasSize
      .pipe(withLatestFrom(this.viewState.devicePixelRatio))
      .subscribe(([size, pixelRatio]) => {
        canvas.width = size.width * pixelRatio;
        canvas.height = size.height * pixelRatio;
        canvas.style.width = size.width + 'px';
        canvas.style.height = size.height + 'px';
      });
    canvas.addEventListener('mousemove', (ev) => {
      const rect = canvas.getBoundingClientRect();
      this.mousePointerLocation.next({
        x: ev.clientX - rect.left,
        y: ev.clientY - rect.top,
      });
    });
    canvas.addEventListener('mouseleave', () => {
      this.mousePointerLocation.next(null);
    });
    canvas.addEventListener('click', (ev) => {
      const rect = canvas.getBoundingClientRect();
      this.mouseClickPointerLocation.next({
        x: ev.clientX - rect.left,
        y: ev.clientY - rect.top,
      });
    });
    const gl = canvas.getContext('webgl2');
    if (!gl) {
      alert(
        'failed to obtain WebGL2 rendering context. Please use the latest supported brwoser.',
      );
      throw new Error(
        'failed to obtain WebGL2 rendering context. Please use the latest supported brwoser.',
      );
    }
    this.sharedGLResources = await this.initSharedGLResources(gl);
    this.selectedTimeVerticalLineRenderer =
      await this.initVerticalLineRenderer(gl);
    this.currentCanvas = canvas;
    this.glContext.next(gl);
    this.initializingGLResources = false;
  }

  public onRender(
    gl: WebGL2RenderingContext,
    canvasSize: CanvasSize,
    offsetToFirstItemFromVisibleArea: number,
    visibleItemRange: ListRange,
    perRowScrollingProperties: PerRowScrollingProperty[],
    rowRenderers: Map<ResourceTimeline, TimelineRowWebGLRenderer>,
    timeRange: TimeRange,
    stickyTimelines: ResourceTimeline[],
    selectedTimeline: ResourceTimeline | null,
    highlightedTimeline: ResourceTimeline | null,
    highlightedByParentSelection: Set<ResourceTimeline>,
    logs: LogEntry[],
    selectedLog: number,
    filteredLog: Set<number>,
    highlightLogs: Set<number>,
    pixelPerTime: number,
    timeOffset: number,
    pixelRatio: number,
  ) {
    if (gl.isContextLost()) {
      if (!this.initializingGLResources) {
        console.warn(
          'Detected the webgl context being lost. Reinitializing gl resources',
        );
        this.refleshGL();
      }
      return;
    }

    this.sharedGLResources!.updateViewState(
      canvasSize.width * pixelRatio,
      canvasSize.height * pixelRatio,
      pixelPerTime * pixelRatio,
      pixelRatio,
      timeOffset - timeRange.begin,
    );
    // No items to render
    if (visibleItemRange.end - visibleItemRange.start === 0) {
      gl.disable(gl.SCISSOR_TEST);
      gl.clearColor(0, 0, 0, 0);
      gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
      gl.finish();
      console.warn('no items to render. Clear screen');
      return;
    }
    const firstItemOffset =
      perRowScrollingProperties.length > 0
        ? perRowScrollingProperties[visibleItemRange.start].offset
        : 0;
    for (let i = visibleItemRange.start; i < visibleItemRange.end; i++) {
      if (perRowScrollingProperties.length <= i) continue;
      const row = perRowScrollingProperties[i];
      if (!row.source) continue;
      const renderer = rowRenderers.get(row.source);
      if (!renderer) continue;

      // Calculate the draw area for each timelines
      const bottom =
        canvasSize.height - (row.offset - firstItemOffset) - row.height;
      const region: GLRect = {
        left: 0 * pixelRatio,
        width: canvasSize.width * pixelRatio,
        bottom: bottom * pixelRatio,
        height: row.height * pixelRatio,
      };
      renderer.loadGLResources();
      renderer.updateInteractiveBuffer(selectedLog, highlightLogs, filteredLog);
      renderer.render(
        pixelRatio,
        region,
        row.source === selectedTimeline,
        row.source === highlightedTimeline,
        highlightedByParentSelection.has(row.source),
      );
    }
    // render sticky timelines
    let offset = 0;
    for (const timeline of stickyTimelines) {
      const row: PerRowScrollingProperty = {
        height: TIMELINE_ITEM_HEIGHTS[timeline.layer],
        offset: offset,
        source: timeline,
      };
      if (!row.source) continue;
      const renderer = rowRenderers.get(row.source);
      if (!renderer) continue;
      // Calculate the draw area for each timelines
      const bottom =
        canvasSize.height -
        row.offset -
        row.height +
        offsetToFirstItemFromVisibleArea;
      const region: GLRect = {
        left: 0 * pixelRatio,
        width: canvasSize.width * pixelRatio,
        bottom: bottom * pixelRatio,
        height: row.height * pixelRatio,
      };
      renderer.loadGLResources();
      renderer.updateInteractiveBuffer(selectedLog, highlightLogs, filteredLog);
      renderer.render(
        pixelRatio,
        region,
        row.source === selectedTimeline,
        row.source === highlightedTimeline,
        highlightedByParentSelection.has(row.source),
      );
      offset += row.height;
    }

    if (selectedLog !== -1) {
      this.selectedTimeVerticalLineRenderer!.render(
        canvasSize,
        pixelRatio,
        this.sharedGLResources!,
        logs[selectedLog].time - timeRange.begin,
        4,
        [0, 1, 0, 0.9],
      );
    }
    gl.flush();
    gl.finish();

    // To prevent blocking rendering the timeline currently shown, preloading step is after calling gl.finish()
    for (
      let i = visibleItemRange.start;
      i >=
      Math.max(
        0,
        Math.min(visibleItemRange.start, perRowScrollingProperties.length) -
          TimelineRendererService.PRELOAD_GL_RESOURCE_AREA,
      );
      i--
    ) {
      const row = perRowScrollingProperties[i];
      if (!row.source) continue;
      rowRenderers.get(row.source)?.loadGLResources();
    }
    for (
      let i = visibleItemRange.end;
      i <
      Math.min(
        perRowScrollingProperties.length,
        visibleItemRange.end + TimelineRendererService.PRELOAD_GL_RESOURCE_AREA,
      );
      i++
    ) {
      const row = perRowScrollingProperties[i];
      if (!row.source) continue;
      rowRenderers.get(row.source)?.loadGLResources();
    }
  }

  private async initSharedGLResources(
    gl: WebGL2RenderingContext,
  ): Promise<SharedGLResources> {
    const shared = new SharedGLResources(gl);
    await shared.init();
    return shared;
  }

  private async initVerticalLineRenderer(
    gl: WebGL2RenderingContext,
  ): Promise<GLVerticalLineRenderer> {
    const lineRenderer = new GLVerticalLineRenderer(gl);
    await lineRenderer.init();
    return lineRenderer;
  }

  private findTimelineFromPosition(
    pos: CanvasMouseLocation | null,
    timeOffset: number,
    pixelPerTime: number,
    lastTimelines: ResourceTimeline[],
    lastStickyTimelines: ResourceTimeline[],
  ): TimelineMouseLocation | null {
    if (pos === null || lastTimelines.length === 0) {
      return null;
    }

    // Find the hitting timeline
    let timeline: ResourceTimeline | null = null;
    let offset = 0;
    for (let i = 0; i < lastStickyTimelines.length; i++) {
      offset += TIMELINE_ITEM_HEIGHTS[lastStickyTimelines[i].layer];
      if (pos.y <= offset) {
        timeline = lastStickyTimelines[i];
        break;
      }
    }
    if (timeline === null) {
      offset = 0;
      for (let i = 0; i < lastTimelines.length; i++) {
        offset += TIMELINE_ITEM_HEIGHTS[lastTimelines[i].layer];
        if (pos.y <= offset) {
          timeline = lastTimelines[i];
          break;
        }
      }
    }
    if (timeline === null) {
      return null;
    }

    // Calculate the location inside of the timeline
    const height = TIMELINE_ITEM_HEIGHTS[timeline.layer];
    offset -= height;
    const y = (pos.y - offset) / height;
    const time = timeOffset + pos.x / pixelPerTime;
    return {
      timeline: timeline,
      y,
      time,
    };
  }
}
