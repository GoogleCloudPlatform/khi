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

import {
  AfterViewInit,
  Component,
  ElementRef,
  Inject,
  Input,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import {
  BehaviorSubject,
  Observable,
  ReplaySubject,
  Subject,
  combineLatest,
  distinctUntilChanged,
  filter,
  map,
  merge,
  scan,
  startWith,
  switchMap,
  take,
  takeUntil,
  withLatestFrom,
} from 'rxjs';
import { BackgroundCanvas } from './background-canvas';
import { TimelinesScrollStrategy } from './TimelineScrollStrategy';
import { TimelinenCoordinateCalculator } from './timeline-coordinate-calculator';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { SelectionManagerService } from '../services/selection-manager.service';
import { ViewStateService } from '../services/view-state.service';
import { TimelineRendererService } from './canvas/timeline_renderer.service';
import { CanvasSize } from './canvas/types';
import { tweenNumber } from '../utils/observable-util';
import { CanvasKeyEventHandler } from './CanvasKeyEventHandler';
import { ResizingCalculator } from '../common/resizable-pane/resizing-calculator';
import {
  convertTimlineEntryToTimelineComponentViewModel,
  emptyTimelineComponentViewModel,
  TimelineComponentViewModel,
  TimelineViewModel,
} from './timeline.component.vm';
import { LogEntry } from '../store/log';
import {
  ResourceRevisionChangePair,
  ResourceTimeline,
} from '../store/timeline';
import {
  LogType,
  ParentRelationshipMetadataType,
  Severity,
} from '../generated';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from '../services/timeline-filter.service';
import { ToTextReferenceFromKHIFileBinary } from '../common/loader/reference-type';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';
import { NavigatorComponent } from './navigator/navigator.component';
import {
  LongTimestampFormatPipe,
  TimestampFormatPipe,
} from '../common/timestamp-format.pipe';
import { MatIconModule } from '@angular/material/icon';
import { KHICommonModule } from '../common/common.module';

interface HoverViewStateLog {
  time: number;
  logIndex: number;
  message: string;
  logTypeCss: string;
  revisionPair: ResourceRevisionChangePair | null;
}

/**
 * Varaibles used in view to show hovered information of highlighted log
 */
interface HoverViewState {
  visible: boolean;
  leftLocation: number;
  topLocation: number;
  height: number;
  beginTime: number;
  endTime: number;
  logCount: number;
  logs: HoverViewStateLog[];
  omitted: boolean;
  readableResourcePathUntilParent: string;
  resourceName: string;
  resourceRelationshipMetadata: ParentRelationshipMetadataType | null;
}

const DEFAULT_HOVER_VIEW_STATE: HoverViewState = {
  visible: false,
  leftLocation: 0,
  topLocation: 0,
  height: 0,
  beginTime: 0,
  endTime: 0,
  logCount: 0,
  logs: [],
  omitted: false,
  readableResourcePathUntilParent: '',
  resourceName: '',
  resourceRelationshipMetadata: null,
};

@Component({
  selector: 'khi-timeline',
  templateUrl: './timeline.component.html',
  styleUrls: ['./timeline.component.sass'],
  providers: [
    CanvasKeyEventHandler,
    TimelinesScrollStrategy,
    TimelineRendererService,
  ],
  imports: [
    CommonModule,
    KHICommonModule,
    MatTooltipModule,
    NavigatorComponent,
    LongTimestampFormatPipe,
    TimestampFormatPipe,
    MatIconModule,
  ],
})
export class TimelineComponent implements OnInit, AfterViewInit, OnDestroy {
  /**
   * Ignores scroll request when the target log entry is already in between [left edge + padding, right edge - padding].
   */
  private readonly HORIZONTAL_SCROLL_PADDING = 100;

  /**
   * Popup bottom margin shown when user hovers on an event or a revision.
   */
  private readonly HOVER_BOTTOM_MARGIN = 50;

  /**
   * The minimum height of the popup.
   */
  private readonly MINIMUM_HOVER_HEIGHT = 200;

  @Input()
  public resizer!: ResizingCalculator;

  private destoroyed = new Subject<void>();

  explorerWidth!: Observable<number>;

  /**
   * True during user holds shift key down
   */
  isInScaleModeSubject: BehaviorSubject<boolean> = new BehaviorSubject(false);

  isInScaleMode = this.isInScaleModeSubject.pipe(distinctUntilChanged());

  timeRange = this._inspectionDataStore.$timeRange;
  highlightLogs = this.selectionManager.highlightedLogs;

  $filteredLog = this._inspectionDataStore.filteredOutLogIndicesSet;

  private canvasMouseMoveEvent = new BehaviorSubject<MouseEvent | null>(null);

  public timelineCoordinateCalculator: TimelinenCoordinateCalculator =
    new TimelinenCoordinateCalculator(this._viewStateService);

  viewModel: Observable<TimelineComponentViewModel> = combineLatest([
    this.timelineScrollStrategy.visibleTimelines,
    this.timelineScrollStrategy.stickyTimelines,
    this.selectionManager.selectedTimeline,
    this.selectionManager.highlightedTimeline,
    this.selectionManager.highlightedChildrenOfSelectedTimeline,
  ]).pipe(
    map(
      ([
        visibleTimelines,
        stickyTimelines,
        selectedTimeline,
        highlightedTimeline,
        highlightedChildrenOfSelectedTimeline,
      ]) => ({
        scrollableTimelines: visibleTimelines.map((timeline) =>
          convertTimlineEntryToTimelineComponentViewModel(timeline),
        ),
        stickyTimelines: stickyTimelines.map((timeline) =>
          convertTimlineEntryToTimelineComponentViewModel(timeline),
        ),
        selectedTimelineResourcePath: selectedTimeline?.resourcePath ?? '',
        highlightedTimeline: highlightedTimeline
          ? convertTimlineEntryToTimelineComponentViewModel(highlightedTimeline)
          : null,
        highlightedTimelineResourcePath:
          highlightedTimeline?.resourcePath ?? '',
        highlightedChildrenOfSelectedTimelineResourcePath: new Set(
          [...highlightedChildrenOfSelectedTimeline].map(
            (timeline) => timeline.resourcePath,
          ),
        ),
      }),
    ),
    startWith(emptyTimelineComponentViewModel()),
  );

  @ViewChild('background')
  background!: ElementRef<HTMLCanvasElement>;
  backgroundCanvasRenderer!: BackgroundCanvas;
  @ViewChild('timelineChartCanvasWrapper')
  timelineChartCanvasWrapper!: ElementRef<HTMLDivElement>;

  @ViewChild('scrollViewport')
  scrollViewport!: ElementRef<HTMLDivElement>;
  @ViewChild('chartArea')
  chartArea!: ElementRef<HTMLDivElement>;
  chartAreaResizeObserver = new ResizeObserver(() => {
    const chartArea = this.chartArea.nativeElement;
    if (chartArea) {
      const rect = chartArea.getBoundingClientRect();
      this._viewStateService.setVisibleWidth(rect.width);
    }
  });

  @ViewChild('canvasParent')
  canvasParent!: ElementRef<HTMLDivElement>;
  canvasParentSizeSubject: Subject<CanvasSize> = new ReplaySubject(1);
  canvasParentSizeObserver!: ResizeObserver;

  constructor(
    private _viewStateService: ViewStateService,
    private _inspectionDataStore: InspectionDataStoreService,
    private selectionManager: SelectionManagerService,
    private timelineScrollStrategy: TimelinesScrollStrategy,
    private timelineRenderer: TimelineRendererService,
    private keyEventHandler: CanvasKeyEventHandler,
    @Inject(DEFAULT_TIMELINE_FILTER) private timelineFilter: TimelineFilter,
  ) {}
  ngOnDestroy(): void {
    this.destoroyed.next();
  }

  totalScrollableHeight: Observable<number> =
    this.timelineScrollStrategy.totalScrollableHeight;
  wrapperYOffset: Observable<number> =
    this.timelineScrollStrategy.offsetToRenderedContent;
  wrapperYHeight: Observable<number> =
    this.timelineScrollStrategy.visibleItemHeight;

  /**
   * A command to order the timeline scrolling to the time offset.
   */
  scrollToOffsetCommand: Subject<number> = new Subject();

  hoverViewState: Observable<HoverViewState> = combineLatest([
    this.canvasMouseMoveEvent,
    this.highlightLogs,
    this.viewModel,
    this.selectionManager.highlightedRevisionsOnCurrentTimeline,
  ]).pipe(
    map(([mouseEvent, highlightLogs, timelineSelection, revisions]) => {
      if (
        !mouseEvent ||
        !highlightLogs.length ||
        timelineSelection.highlightedTimelineResourcePath === ''
      )
        return DEFAULT_HOVER_VIEW_STATE;
      const MAX_VISIBLE_LOGS = 10;
      const highlightedLogTimestamps: number[] = [];
      const logs: HoverViewStateLog[] = [];
      if (revisions.length == 0) {
        // for event logs
        for (
          let i = 0;
          i < Math.min(highlightLogs.length, MAX_VISIBLE_LOGS);
          i++
        ) {
          const eventLog = highlightLogs[i];
          logs.push({
            message: eventLog.summary,
            logIndex: eventLog.logIndex,
            time: eventLog.time,
            logTypeCss: eventLog.logTypeLabel,
            revisionPair: null,
          });
          highlightedLogTimestamps.push(eventLog.time);
        }
      } else {
        const logByIndex: { [logIndex: number]: LogEntry } = {};
        for (
          let i = 0;
          i < Math.min(highlightLogs.length, MAX_VISIBLE_LOGS);
          i++
        ) {
          logByIndex[highlightLogs[i].logIndex] = highlightLogs[i];
        }
        // for revision logs
        for (let i = 0; i < Math.min(revisions.length, MAX_VISIBLE_LOGS); i++) {
          const revision = revisions[i];
          let associatedLog = logByIndex[revision.logIndex];
          if (!associatedLog && revision.logIndex >= 0) continue;
          if (revision.logIndex === -1) {
            // The revision has no associated log. The resource status was inferred from other logs.
            // Generates a pseudo log and only set the log time from revision start time.
            associatedLog = new LogEntry(
              -1,
              '',
              LogType.LogTypeUnknown,
              Severity.SeverityUnknown,
              revision.startAt,
              '',
              ToTextReferenceFromKHIFileBinary({
                offset: 0,
                len: 0,
                buffer: 0,
              }),
              [],
            );
          }
          const prev =
            logs.length > 0
              ? logs[logs.length - 1].revisionPair!.current
              : null;
          logs.push({
            message: associatedLog.summary,
            logIndex: associatedLog.logIndex,
            time: revision.startAt,
            logTypeCss: associatedLog.logTypeLabel,
            revisionPair: new ResourceRevisionChangePair(prev, revision),
          });
          highlightedLogTimestamps.push(associatedLog.time);
        }
      }
      const pathFragments =
        timelineSelection.highlightedTimelineResourcePath.split('#');
      const resourceName = pathFragments[pathFragments.length - 1];
      if (pathFragments.length > 2 && pathFragments[2] === 'cluster-scope') {
        pathFragments.splice(2, 1);
      }
      pathFragments.splice(0, 1); // omit the api version
      pathFragments.splice(pathFragments.length - 1, 1);
      return {
        visible: true,
        topLocation: mouseEvent.clientY,
        leftLocation: mouseEvent.clientX,
        height: Math.max(
          this.MINIMUM_HOVER_HEIGHT,
          document.body.clientHeight -
            mouseEvent.clientY -
            this.HOVER_BOTTOM_MARGIN,
        ),
        endTime: Math.max(...highlightedLogTimestamps),
        beginTime: Math.min(...highlightedLogTimestamps),
        logCount: highlightLogs.length,
        logs: logs,
        isRevisions: revisions.length > 0,
        omitted: highlightLogs.length > MAX_VISIBLE_LOGS,
        readableResourcePathUntilParent: pathFragments.join(' > '),
        resourceName: resourceName,
        resourceRelationshipMetadata:
          timelineSelection.highlightedTimeline?.relationshipMetadata ?? null,
      } as HoverViewState;
    }),
    startWith(DEFAULT_HOVER_VIEW_STATE),
  );

  ngOnInit(): void {
    this.timeRange
      .pipe(takeUntil(this.destoroyed))
      .subscribe(() => this.onTimelineRangeChanged());
    this._viewStateService.timelineStateResetCommand
      .pipe(takeUntil(this.destoroyed))
      .subscribe(() => this.resetTimelineScaleAndOffset());

    this.selectionManager.selectedLog
      .pipe(
        takeUntil(this.destoroyed),
        filter((log): log is LogEntry => !!log),
      )
      .subscribe((log) => {
        const currentSelectedTime = log.time;
        this.backgroundCanvasRenderer.setSelectedTimeStamp(currentSelectedTime);
        this.backgroundCanvasRenderer.invalidate();
      });

    this.selectionManager.selectedTimeline
      .pipe(
        takeUntil(this.destoroyed),
        filter((timeline): timeline is ResourceTimeline => !!timeline),
      )
      .subscribe((timeline) =>
        this.timelineScrollStrategy.scrollToTimeline(timeline),
      );
    this.explorerWidth = this.resizer.areaSize('explorer-view');
  }

  ngAfterViewInit(): void {
    this.chartAreaResizeObserver.observe(this.chartArea.nativeElement);
    this.timelineScrollStrategy.attach(this.scrollViewport.nativeElement);
    this.canvasParentSizeObserver = new ResizeObserver(() => {
      const parentSize =
        this.canvasParent.nativeElement.getBoundingClientRect();
      this.canvasParentSizeSubject.next({
        width: parentSize.width,
        height: parentSize.height,
      });
    });
    this.canvasParentSizeObserver.observe(this.canvasParent.nativeElement);
    this.timelineRenderer.attach(
      this.timelineChartCanvasWrapper.nativeElement,
      combineLatest([
        this._viewStateService.visibleWidth,
        // canvas height should be kept to reduce frequent resizing due to the scroll
        this.timelineScrollStrategy.visibleItemHeight.pipe(
          scan((a, b) => Math.max(a, b), 0),
        ),
      ]).pipe(map(([width, height]) => ({ width, height }))),
    );
    this.backgroundCanvasRenderer = new BackgroundCanvas(
      this.background.nativeElement,
      this.canvasParentSizeSubject,
      this.timelineCoordinateCalculator,
      this._inspectionDataStore,
      this.timelineFilter,
      this._viewStateService,
      this.selectionManager,
    );

    merge(
      this._viewStateService.timeOffset,
      this._viewStateService.pixelPerTime,
    )
      .pipe(takeUntil(this.destoroyed))
      .subscribe(() => {
        this.backgroundCanvasRenderer.invalidate();
      });

    this.scrollToOffsetCommand
      .pipe(
        takeUntil(this.destoroyed),
        withLatestFrom(this._viewStateService.timeOffset),
        switchMap(([destination, current]) =>
          tweenNumber(current, destination, 500),
        ),
      )
      .subscribe((offset) => {
        this._viewStateService.setTimeOffset(offset);
      });

    this.selectionManager.selectedRevision
      .pipe(
        takeUntil(this.destoroyed),
        withLatestFrom(
          this._viewStateService.visibleWidth,
          this._viewStateService.timeOffset,
          this._viewStateService.pixelPerTime,
        ),
      )
      .subscribe(([revision, width, offset, pixelPerTime]) => {
        if (!revision) return;
        const leftOffsetToRevision = (revision.startAt - offset) * pixelPerTime;
        const rightOffsetToRevision = (revision.endAt - offset) * pixelPerTime;
        let destinationOffsetPixels = leftOffsetToRevision;
        if (
          rightOffsetToRevision < this.HORIZONTAL_SCROLL_PADDING ||
          leftOffsetToRevision > width - this.HORIZONTAL_SCROLL_PADDING
        ) {
          destinationOffsetPixels =
            leftOffsetToRevision - this.HORIZONTAL_SCROLL_PADDING;
          this.scrollToOffsetCommand.next(
            destinationOffsetPixels / pixelPerTime + offset,
          );
        }
      });
    this.backgroundCanvasRenderer.invalidate();
  }

  _timelineDragStart() {
    const canvasMove = (e: MouseEvent) => {
      this._moveTimeOffset(-e.movementX);
    };
    window.addEventListener('mouseup', () => {
      window.removeEventListener('mousemove', canvasMove);
    });
    window.addEventListener('mousemove', canvasMove);
  }

  _resizeStart() {
    const resizeMove = (e: MouseEvent) => {
      const size = this.resizer.getAreaSize('explorer-view');
      this.resizer.resizeArea('explorer-view', size + e.movementX);
    };
    window.addEventListener('mouseup', () => {
      window.removeEventListener('mousemove', resizeMove);
    });
    window.addEventListener('mousemove', resizeMove);
  }

  onTimelineRangeChanged() {
    this.resetTimelineScaleAndOffset();
    if (!this.backgroundCanvasRenderer) return;
    this.backgroundCanvasRenderer.invalidate();
  }

  resetTimelineScaleAndOffset() {
    // Returns if there is no elements included in the timeline data
    this._inspectionDataStore.inspectionData
      .pipe(take(1))
      .subscribe((inspectionData) => {
        if (!!inspectionData) {
          const range = inspectionData.range;
          if (range.begin === range.end) return;
          const timeMargin = range.duration * 0.05;
          this._viewStateService.setTimeOffset(range.begin - timeMargin);
          if (this.scrollViewport) {
            const explorerSize = this.resizer.getAreaSize('explorer-view');
            const chartBodyElement = this.scrollViewport.nativeElement;
            const chartBodyRect = chartBodyElement.getBoundingClientRect();
            const timelineChartWidthInPx =
              chartBodyRect.width - explorerSize - 5;
            this._viewStateService.setPixelPerTime(
              timelineChartWidthInPx / (range.duration + 2 * timeMargin),
            );
          }
        }
      });
  }

  _onScrollTimeline(e: WheelEvent) {
    this._moveTimeOffset(e.deltaX); // Almost just for Mac OS
  }

  _onScrollRulerView(e: WheelEvent) {
    const rect = (e.target as HTMLDivElement).getBoundingClientRect();
    this._moveTimeOffset(e.deltaX);
    this._moveScale(-e.deltaY, e.x - rect.left);
  }

  /**
   * Move offset of left edge of timeline view
   * @param delta delta value of control(Scroll amount or drag distance).
   */
  _moveTimeOffset(delta: number) {
    const MINIMUM_PIXELS_IN_AREA = 300;
    this._inspectionDataStore.inspectionData
      .pipe(take(1))
      .subscribe((inspectionData) => {
        if (!!inspectionData) {
          const range = inspectionData.range;
          const minimalPixelsInAreaInTime =
            MINIMUM_PIXELS_IN_AREA / this._viewStateService.getPixelPerTime();
          const current = this._viewStateService.getTimeOffset();
          const nextUnlimitedTimeOffset =
            current + delta / this._viewStateService.getPixelPerTime();
          const maximumAllowedOffset = range.begin - minimalPixelsInAreaInTime;
          const minimumAllowedOffset = range.end - minimalPixelsInAreaInTime;
          let nextTimeOffset = Math.min(
            nextUnlimitedTimeOffset,
            minimumAllowedOffset,
          );
          nextTimeOffset = Math.max(nextTimeOffset, maximumAllowedOffset);
          this._viewStateService.setTimeOffset(nextTimeOffset);
        }
      });
  }

  _moveScale(delta: number, scaleCenter: number) {
    // The amount of scroll event is totally depends on OS and browser.
    // TODO: Currently only checks the sign of delta. Not supporting continuous scrolling on Mac.
    // https://developer.mozilla.org/en-US/docs/Web/API/Element/mousewheel_event#chrome
    const MAX_TIME_RANGE_WIDTH = 300;
    const SCALING_SPEED = 0.2;
    this._inspectionDataStore.inspectionData
      .pipe(take(1))
      .subscribe((inspectionData) => {
        if (!!inspectionData) {
          const range = inspectionData.range;
          const deltaSign = Math.sign(delta);
          const currentScale = this._viewStateService.getPixelPerTime();
          const nextScale = Math.min(
            1,
            Math.max(
              Number.MIN_VALUE,
              currentScale * (1 + deltaSign * SCALING_SPEED),
            ),
          );
          const minimalPixelPerTime = MAX_TIME_RANGE_WIDTH / range.duration;
          const cursorTime = scaleCenter / currentScale;
          const nextTime = scaleCenter / nextScale;
          const currentOffset = this._viewStateService.getTimeOffset();
          const isScalingUp = nextScale > currentScale;
          // Ignore scaling down when it reaches the minimum scale. This check is ignored on scaling up because it could be temporary smaller than the limit because of the parent window resizes.
          if (isScalingUp || nextScale > minimalPixelPerTime) {
            this._viewStateService.setPixelPerTime(
              Math.max(nextScale, minimalPixelPerTime),
            );
            this._viewStateService.setTimeOffset(
              cursorTime - nextTime + currentOffset,
            );
          }
        }
      });
  }

  onTimelineHeaderClick(timeline: ResourceTimeline) {
    this.selectionManager.onSelectTimeline(timeline);
  }

  onTimelineHeaderMouseOver(timeline: ResourceTimeline) {
    this.selectionManager.onHighlightTimeline(timeline);
  }

  onTimelineMouseMove(event: MouseEvent) {
    this.canvasMouseMoveEvent.next(event);
  }
  timelineTrackBy(index: number, timeline: TimelineViewModel): string {
    return timeline.resourcePath;
  }

  onChartScroll(mouseEvent: WheelEvent) {
    if (mouseEvent.shiftKey) {
      const rect = (
        mouseEvent.target as HTMLDivElement
      ).getBoundingClientRect();
      this._moveScale(-mouseEvent.deltaY, mouseEvent.x - rect.left);
      mouseEvent.preventDefault();
      mouseEvent.stopPropagation();
      this.isInScaleModeSubject.next(true);
      return false;
    } else {
      this.isInScaleModeSubject.next(false);
      return true;
    }
  }

  onShiftStatusChange(shiftStatus: boolean) {
    this.isInScaleModeSubject.next(shiftStatus);
  }

  onKeyDownOverCanvas(keyboardEvent: KeyboardEvent) {
    this.keyEventHandler.keydown(keyboardEvent);
  }
}
