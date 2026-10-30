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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import {
  TimelineHoverOverlay,
  TimelineHoverOverlayComponent,
} from './timeline-hover-overlay.component';
import { TimelineChartItemHighlightType } from './interaction-model';
import { RevisionStateStyle } from 'src/app/store/domain/style';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { Timeline, Revision } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';

describe('TimelineHoverOverlayComponent', () => {
  let component: TimelineHoverOverlayComponent;
  let fixture: ComponentFixture<TimelineHoverOverlayComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimelineHoverOverlayComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TimelineHoverOverlayComponent);
    component = fixture.componentInstance;
  });

  function createMockLog(
    logIndex: number,
    timeMs: number,
  ): ReadonlyDomainElement<Log> {
    return {
      id: logIndex,
      logIndex,
      timestamp: BigInt(timeMs) * 1000000n,
      legacyTimestampMs: timeMs,
      summary: `log-${logIndex}`,
      logType: {
        id: 1,
        label: 'Audit',
        description: 'Audit',
        backgroundColor: { r: 0.1, g: 0.3, b: 0.6, a: 1.0 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1.0 },
      },
      severity: {
        id: 1,
        shortLabel: 'I',
        label: 'Info',
        backgroundColor: { r: 0.2, g: 0.6, b: 0.2, a: 1.0 },
        foregroundColor: { r: 1, g: 1, b: 1, a: 1.0 },
      },
    } as unknown as ReadonlyDomainElement<Log>;
  }

  function createMockRevision(
    id: number,
    log: ReadonlyDomainElement<Log>,
    changedTimeMs: number,
  ): ReadonlyDomainElement<Revision> {
    return {
      id,
      timelineId: 1,
      index: id,
      legacyChangedTimeMs: changedTimeMs,
      changedTime: BigInt(changedTimeMs) * 1000000n,
      log,
      state: {
        id: 1,
        label: 'Running',
        icon: 'check_circle',
        style: RevisionStateStyle.NORMAL,
        backgroundColor: { r: 0.1, g: 0.5, b: 0.1, a: 1.0 },
      },
      verb: {
        id: 1,
        label: 'PATCH',
        backgroundColor: { r: 0.2, g: 0.4, b: 0.8, a: 1.0 },
      },
    } as unknown as ReadonlyDomainElement<Revision>;
  }

  it('should disambiguate highlights for revisions sharing the same log', () => {
    const sharedLog = createMockLog(10, 10000);
    const rev1 = createMockRevision(1, sharedLog, 4000);
    const rev2 = createMockRevision(2, sharedLog, 8500);
    const initialRev = createMockRevision(0, sharedLog, 0);

    const timeline = {
      id: 1,
      name: 'main',
      revisions: [initialRev, rev1, rev2],
      events: [],
    } as unknown as ReadonlyDomainElement<Timeline>;

    const overlay: TimelineHoverOverlay = {
      timeline,
      revisions: [initialRev, rev1, rev2],
      events: [],
      initialRevision: null,
      cursorTime: null,
      selectedRevisionId: 2,
      highlightedRevisionId: 1,
    };

    fixture.componentRef.setInput('timelineHoverOverlay', overlay);
    fixture.componentRef.setInput('highlights', {
      10: TimelineChartItemHighlightType.Selected,
    });
    fixture.detectChanges();

    const vm = component.viewModel();
    expect(vm.logs.length).toBe(3);

    expect(vm.logs[0].highlightType).toBe(TimelineChartItemHighlightType.None);
    expect(vm.logs[1].highlightType).toBe(
      TimelineChartItemHighlightType.Hovered,
    );
    expect(vm.logs[2].highlightType).toBe(
      TimelineChartItemHighlightType.Selected,
    );
  });
});
