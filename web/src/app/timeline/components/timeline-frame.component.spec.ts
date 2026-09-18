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
import { TimelineChartComponent } from './timeline-chart.component';
import { TimelineRulerComponent } from './timeline-ruler.component';

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
  imports: [TimelineFrameComponent],
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

  it('should clamp selected time range to minQueryLogTimeMS and maxQueryLogTimeMS when inspection bounds are set', () => {
    frameFixture.componentRef.setInput('minQueryLogTimeMS', 2000);
    frameFixture.componentRef.setInput('maxQueryLogTimeMS', 5000);
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
});
