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

import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { TimelineFrameComponent } from 'src/app/timeline/components/timeline-frame.component';
import { Timeline, Event } from 'src/app/store/domain/timeline';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { Log } from 'src/app/store/domain/log';
import { LogStore } from 'src/app/store/domain/log-store';
import {
  TimelineHighlightType,
  TimelineChartItemHighlightType,
} from 'src/app/timeline/components/interaction-model';
import { TimelineType } from 'src/app/store/domain/style';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import {
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model';
import { TimeRangeFilter } from 'src/app/services/view-state.service';
import { GLContextManager } from './canvas/glcontextmanager';
import { TimelineChartComponent } from './timeline-chart.component';
import { TimelineRulerComponent } from './timeline-ruler.component';
import { CaptureShiftKeyDirective } from 'src/app/common/capture-shiftkey.directive';
import { TimelineHoverOverlayComponent } from './timeline-hover-overlay.component';

import { ReadonlyDomainElement } from 'src/app/store/domain/types';

const mockTimelineType: TimelineType = {
  id: 0,
  label: 'mock-type',
  description: 'mock type description',
  icon: 'timeline',
  backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
  typeChipBackgroundColor: { r: 0, g: 0, b: 0, a: 1 },
  typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
  visible: true,
  sortPriority: 0,
  height: 24,
};

class MockTimeline extends Timeline {
  private mockEvents: Event[] = [];

  constructor(id: number) {
    super(id, null as unknown as TimelineStore);
  }

  public setEvents(events: Event[]): void {
    this.mockEvents = events;
  }

  override get events(): readonly Event[] {
    return this.mockEvents;
  }

  override get revisions(): readonly never[] {
    return [];
  }

  override get type(): ReadonlyDomainElement<TimelineType> {
    return mockTimelineType;
  }

  override get name(): string {
    return `timeline-${this.id}`;
  }

  override get layer(): number {
    return 0;
  }

  override get parent(): ReadonlyDomainElement<Timeline> | null {
    return null;
  }

  override get childrenCount(): number {
    return 0;
  }
}

class MockEvent extends Event {
  private readonly mockLogIndex: number;

  constructor(id: number, timelineId: number, logIndex: number) {
    super(id, timelineId, null as unknown as TimelineStore);
    this.mockLogIndex = logIndex;
  }

  override get logIndex(): number {
    return this.mockLogIndex;
  }
}

class MockLog extends Log {
  private readonly _logIndex: number;

  constructor(logIndex: number) {
    super(0, null as unknown as LogStore);
    this._logIndex = logIndex;
  }

  override get logIndex(): number {
    return this._logIndex;
  }
}

@Component({
  selector: 'khi-testing-timeline-frame',
  standalone: true,
  template: '',
})
class TestingTimelineFrameComponent extends TimelineFrameComponent {
  // eslint-disable-next-line @angular-eslint/no-empty-lifecycle-method
  override ngAfterViewInit(): void {}

  public getSelectedLogTimelineExposed(): ReadonlyDomainElement<Timeline> | null {
    return this.selectedLogTimeline();
  }
}

const mockStyleStore: StyleStoreLike = {
  severities: [],
  logTypes: [],
  verbs: [],
  revisionStates: [],
  timelineTypes: [],
  getSeverity: () => {
    throw new Error();
  },
  getLogType: () => {
    throw new Error();
  },
  getVerb: () => {
    throw new Error();
  },
  getRevisionState: () => {
    throw new Error();
  },
  getTimelineType: () => undefined as unknown as TimelineType,
  getIconAtlas: () => undefined,
};

describe('TimelineFrameComponent', () => {
  let component: TestingTimelineFrameComponent;
  let fixture: ComponentFixture<TestingTimelineFrameComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestingTimelineFrameComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TestingTimelineFrameComponent);
    component = fixture.componentInstance;

    fixture.componentRef.setInput('chartStyle', generateDefaultChartStyle());
    fixture.componentRef.setInput(
      'rulerStyle',
      generateDefaultRulerStyle(mockStyleStore),
    );
    fixture.componentRef.setInput('styleStore', mockStyleStore);

    const mockLogs: Log[] = [];
    for (let i = 0; i <= 20; i++) {
      mockLogs.push(new MockLog(i));
    }
    fixture.componentRef.setInput('allLogs', mockLogs);

    fixture.detectChanges();
  });

  it('should resolve the first timeline containing the log if no timeline is selected', () => {
    const timelineA = new MockTimeline(1);
    const timelineB = new MockTimeline(2);

    const eventA = new MockEvent(101, 1, 10);
    const eventB = new MockEvent(102, 2, 10);

    timelineA.setEvents([eventA]);
    timelineB.setEvents([eventB]);

    fixture.componentRef.setInput('timelines', [timelineA, timelineB]);
    fixture.componentRef.setInput('timelineChartItemHighlights', {
      10: TimelineChartItemHighlightType.Selected,
    });
    fixture.detectChanges();

    expect(component.getSelectedLogTimelineExposed()).toBe(timelineA);
  });

  it('should prioritize the currently selected timeline if it contains the log', () => {
    const timelineA = new MockTimeline(1);
    const timelineB = new MockTimeline(2);

    const eventA = new MockEvent(101, 1, 10);
    const eventB = new MockEvent(102, 2, 10);

    timelineA.setEvents([eventA]);
    timelineB.setEvents([eventB]);

    fixture.componentRef.setInput('timelines', [timelineA, timelineB]);
    fixture.componentRef.setInput('timelineHighlights', {
      2: TimelineHighlightType.Selected, // Timeline B is selected
    });
    fixture.componentRef.setInput('timelineChartItemHighlights', {
      10: TimelineChartItemHighlightType.Selected,
    });
    fixture.detectChanges();

    expect(component.getSelectedLogTimelineExposed()).toBe(timelineB);
  });

  it('should fall back to the first timeline containing the log if the selected timeline does not contain it', () => {
    const timelineA = new MockTimeline(1);
    const timelineB = new MockTimeline(2);
    const timelineC = new MockTimeline(3); // Unrelated timeline

    const eventA = new MockEvent(101, 1, 10);
    const eventB = new MockEvent(102, 2, 10);

    timelineA.setEvents([eventA]);
    timelineB.setEvents([eventB]);

    fixture.componentRef.setInput('timelines', [
      timelineA,
      timelineB,
      timelineC,
    ]);
    fixture.componentRef.setInput('timelineHighlights', {
      3: TimelineHighlightType.Selected, // Timeline C is selected, does not contain log index 10
    });
    fixture.componentRef.setInput('timelineChartItemHighlights', {
      10: TimelineChartItemHighlightType.Selected,
    });
    fixture.detectChanges();

    expect(component.getSelectedLogTimelineExposed()).toBe(timelineA);
  });
});

describe('TimelineFrameComponent - time range selection', () => {
  let frameFixture: ComponentFixture<TimelineFrameComponent>;
  let frameComponent: TimelineFrameComponent;

  beforeEach(async () => {
    spyOn(GLContextManager.prototype, 'setup').and.resolveTo();
    spyOn(GLContextManager.prototype, 'dispose').and.stub();
    spyOn(TimelineChartComponent.prototype, 'ngAfterViewInit').and.stub();
    spyOn(TimelineRulerComponent.prototype, 'ngAfterViewInit').and.stub();
    TestBed.configureTestingModule({
      imports: [TimelineFrameComponent],
    });
    TestBed.overrideComponent(TimelineChartComponent, {
      set: { template: '' },
    });
    TestBed.overrideComponent(TimelineRulerComponent, {
      set: {
        template: '<div #container><canvas #backgroundCanvas></canvas></div>',
      },
    });
    await TestBed.compileComponents();

    frameFixture = TestBed.createComponent(TimelineFrameComponent);
    frameComponent = frameFixture.componentInstance;

    frameFixture.componentRef.setInput(
      'chartStyle',
      generateDefaultChartStyle(),
    );
    frameFixture.componentRef.setInput(
      'rulerStyle',
      generateDefaultRulerStyle(mockStyleStore),
    );
    frameFixture.componentRef.setInput('styleStore', mockStyleStore);
    frameFixture.componentRef.setInput('allLogs', []);
    frameFixture.detectChanges();
  });

  it('should render .time-range-body-overlay.active-range when timeRangeFilter input is provided', () => {
    frameFixture.componentRef.setInput('timeRangeFilter', {
      startTime: 1000000000000n,
      endTime: 2000000000000n,
    });
    frameFixture.detectChanges();

    const overlay = frameFixture.nativeElement.querySelector(
      '.time-range-body-overlay.active-range',
    );
    expect(overlay).not.toBeNull();
  });

  it('should render .time-range-body-overlay.preview-range when previewTimeRangeMs is updated from ruler', () => {
    const rulerDebugEl = frameFixture.debugElement.query(
      (el) => el.componentInstance instanceof TimelineRulerComponent,
    );
    rulerDebugEl.componentInstance.previewTimeRangeMs.set({
      startMs: 1000,
      endMs: 2000,
    });
    frameFixture.detectChanges();

    const overlay = frameFixture.nativeElement.querySelector(
      '.time-range-body-overlay.preview-range',
    );
    expect(overlay).not.toBeNull();

    rulerDebugEl.componentInstance.previewTimeRangeMs.set(null);
    frameFixture.detectChanges();

    expect(
      frameFixture.nativeElement.querySelector(
        '.time-range-body-overlay.preview-range',
      ),
    ).toBeNull();
  });

  it('should convert ruler time range selection to nanosecond TimeRangeFilter and emit timeRangeSelected', () => {
    let emittedRange: TimeRangeFilter | undefined;
    frameComponent.timeRangeSelected.subscribe((range) => {
      emittedRange = range;
    });

    const rulerDebugEl = frameFixture.debugElement.query(
      (el) => el.componentInstance instanceof TimelineRulerComponent,
    );
    rulerDebugEl.componentInstance.timeRangeSelected.emit({
      startMs: 1500,
      endMs: 3500,
    });
    frameFixture.detectChanges();

    expect(emittedRange).toEqual({
      startTime: 1500000000n,
      endTime: 3500000000n,
    });
  });

  it('should clamp selected time range to minQueryLogTimeNs and maxQueryLogTimeNs when inspection bounds are set', () => {
    frameFixture.componentRef.setInput('minQueryLogTimeNs', 2000000000n);
    frameFixture.componentRef.setInput('maxQueryLogTimeNs', 5000000000n);
    frameFixture.detectChanges();

    let emittedRange: TimeRangeFilter | undefined;
    frameComponent.timeRangeSelected.subscribe((range) => {
      emittedRange = range;
    });

    const rulerDebugEl = frameFixture.debugElement.query(
      (el) => el.componentInstance instanceof TimelineRulerComponent,
    );
    rulerDebugEl.componentInstance.timeRangeSelected.emit({
      startMs: 1000,
      endMs: 6000,
    });
    frameFixture.detectChanges();

    expect(emittedRange).toEqual({
      startTime: 2000000000n,
      endTime: 5000000000n,
    });
  });

  it('should propagate timeRangeCleared from child ruler component', () => {
    let clearedEmitted = false;
    frameComponent.timeRangeCleared.subscribe(() => {
      clearedEmitted = true;
    });

    const rulerDebugEl = frameFixture.debugElement.query(
      (el) => el.componentInstance instanceof TimelineRulerComponent,
    );
    rulerDebugEl.componentInstance.timeRangeCleared.emit();
    frameFixture.detectChanges();

    expect(clearedEmitted).toBeTrue();
  });

  function queryCharts(): TimelineChartComponent[] {
    return frameFixture.debugElement
      .queryAll(By.directive(TimelineChartComponent))
      .map((el) => el.componentInstance as TimelineChartComponent);
  }

  it('should hide legend and resizer handle when showLegend and showResizerHandle are false', () => {
    expect(frameFixture.nativeElement.querySelector('.legend')).not.toBeNull();
    expect(
      frameFixture.nativeElement.querySelector('.gutter-hint'),
    ).not.toBeNull();

    frameFixture.componentRef.setInput('showLegend', false);
    frameFixture.componentRef.setInput('showResizerHandle', false);
    frameFixture.detectChanges();

    expect(frameFixture.nativeElement.querySelector('.legend')).toBeNull();
    expect(frameFixture.nativeElement.querySelector('.gutter-hint')).toBeNull();
  });

  it('should compute readyToRender only when all child charts are ready', () => {
    const charts = queryCharts();
    for (const tc of [
      { ready: [false, false], want: false },
      { ready: [true, false], want: false },
      { ready: [true, true], want: true },
    ]) {
      expect(charts.length).toBe(tc.ready.length);
      charts.forEach((chart, i) => chart.readyToRender.set(tc.ready[i]));
      frameFixture.detectChanges();

      expect(frameComponent.readyToRender()).toBe(tc.want);
    }
  });

  it('should update viewport models and invalidate all child charts in syncViewportAndInvalidate', () => {
    const charts = queryCharts();
    charts.forEach((chart) => {
      chart.invalidate = false;
    });

    frameComponent.syncViewportAndInvalidate(2000, 2);

    expect({
      viewportLeftTimeMS: frameComponent.viewportLeftTimeMS(),
      pixelsPerMs: frameComponent.pixelsPerMs(),
      invalidated: charts.map((chart) => chart.invalidate),
    }).toEqual({
      viewportLeftTimeMS: 2000,
      pixelsPerMs: 2,
      invalidated: charts.map(() => true),
    });
  });

  it('should propagate Alt key status to child charts and hover overlay as hideEvents without enabling scaleMode', () => {
    const timelineA = new MockTimeline(1);
    frameFixture.componentRef.setInput('timelines', [timelineA]);
    frameFixture.componentRef.setInput('timelineHoverOverlayRequest', {
      timeMs: 1000,
      timelineId: 1,
      isMouseOnTimeline: true,
      overlay: {
        timeline: timelineA,
        revisions: [],
        events: [],
        initialRevision: null,
        cursorTime: 1000000000n,
      },
    });
    frameFixture.detectChanges();

    const shiftDirective = frameFixture.debugElement
      .query(By.directive(CaptureShiftKeyDirective))
      .injector.get(CaptureShiftKeyDirective);
    const emittedAltStatuses: boolean[] = [];
    frameComponent.altStatusChange.subscribe((status) => {
      emittedAltStatuses.push(status);
    });

    const charts = queryCharts();
    const hoverOverlay = frameFixture.debugElement.query(
      By.directive(TimelineHoverOverlayComponent),
    ).componentInstance as TimelineHoverOverlayComponent;

    expect(charts.map((chart) => chart.hideEvents())).toEqual([false, false]);
    expect(hoverOverlay.hideEvents()).toBeFalse();
    expect(frameComponent['scaleMode']()).toBeFalse();

    shiftDirective.altStatusChange.emit(true);
    frameFixture.detectChanges();

    expect(emittedAltStatuses).toEqual([true]);
    expect(charts.map((chart) => chart.hideEvents())).toEqual([true, true]);
    expect(hoverOverlay.hideEvents()).toBeTrue();
    expect(frameComponent['scaleMode']()).toBeFalse();

    shiftDirective.altStatusChange.emit(false);
    frameFixture.detectChanges();

    expect(emittedAltStatuses).toEqual([true, false]);
    expect(charts.map((chart) => chart.hideEvents())).toEqual([false, false]);
    expect(hoverOverlay.hideEvents()).toBeFalse();
  });

  it('should clamp cursorOffsetLeft to minQueryLogTimeNs when cursorTimeNs precedes the query start time', () => {
    frameFixture.componentRef.setInput('minQueryLogTimeNs', 2000000000n);
    frameFixture.componentRef.setInput('maxQueryLogTimeNs', 5000000000n);
    frameFixture.componentRef.setInput('pixelsPerMs', 1);
    frameFixture.componentRef.setInput('cursorTimeNs', 1000000000n);
    frameFixture.detectChanges();

    const offsetAtMinQueryTime = frameComponent[
      'horizontalScrollCalculator'
    ]().timeMSToOffsetLeft(2000, 1);
    expect(frameComponent['cursorOffsetLeft']()).toBe(offsetAtMinQueryTime);
  });

  it('should render secondary cursor line and delta label only when secondaryCursorTimeNs differs from cursorTimeNs', () => {
    frameFixture.componentRef.setInput('cursorTimeNs', 10000000000n);
    frameFixture.componentRef.setInput('secondaryCursorTimeNs', 0n);
    frameFixture.detectChanges();

    expect(
      frameFixture.nativeElement.querySelector(
        '.secondary-cursor-time-label-container',
      ),
    ).toBeNull();

    frameFixture.componentRef.setInput('secondaryCursorTimeNs', 14400000000n);
    frameFixture.detectChanges();

    const labelEl = frameFixture.nativeElement.querySelector(
      '.secondary-cursor-time-label',
    ) as HTMLElement | null;
    expect(labelEl).not.toBeNull();
    expect(labelEl?.textContent).toContain('+4.4s');
    expect(
      frameFixture.nativeElement.querySelector(
        '.secondary-cursor-vertical-line-body',
      ),
    ).not.toBeNull();
  });
});
