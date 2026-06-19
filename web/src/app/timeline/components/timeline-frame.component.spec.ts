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
} from 'src/app/timeline/components/style-model-v2';

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
  getTimelineType: () => {
    throw new Error();
  },
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
    fixture.componentRef.setInput('allLogsWithoutFilter', mockLogs);

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
