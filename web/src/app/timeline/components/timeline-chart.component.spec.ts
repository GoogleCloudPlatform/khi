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
import { TimelineChartComponent } from 'src/app/timeline/components/timeline-chart.component';
import {
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { TimelineType } from 'src/app/store/domain/style';

@Component({
  selector: 'khi-testing-timeline-chart',
  template: '',
})
class TestingTimelineChartComponent extends TimelineChartComponent {
  override ngAfterViewInit(): Promise<void> {
    return Promise.resolve();
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

describe('TimelineChartComponent', () => {
  let component: TestingTimelineChartComponent;
  let fixture: ComponentFixture<TestingTimelineChartComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestingTimelineChartComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TestingTimelineChartComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('chartStyle', generateDefaultChartStyle());
    fixture.componentRef.setInput(
      'rulerStyle',
      generateDefaultRulerStyle(mockStyleStore),
    );
    fixture.detectChanges();
  });

  it('should create the component with invalidate initially true', () => {
    expect(component).toBeTruthy();
    expect(component.invalidate).toBeTrue();
  });

  it('should set invalidate to true when leftEdgeTime changes', () => {
    component.invalidate = false;
    expect(component.invalidate).toBeFalse();

    fixture.componentRef.setInput('leftEdgeTime', 5000);
    fixture.detectChanges();

    expect(component.invalidate).toBeTrue();
  });

  it('should set invalidate to true when pixelsPerMs changes', () => {
    component.invalidate = false;
    expect(component.invalidate).toBeFalse();

    fixture.componentRef.setInput('pixelsPerMs', 2.5);
    fixture.detectChanges();

    expect(component.invalidate).toBeTrue();
  });
});
