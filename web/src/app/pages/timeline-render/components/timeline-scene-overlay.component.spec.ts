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
import { By } from '@angular/platform-browser';
import {
  TimelineExcerptKind,
  TimelineExcerptLineKind,
  TimelineMarkerStyle,
  TimelineSpanPattern,
} from 'src/app/common/schema/timeline-render-types';
import { TimelineSceneOverlayComponent } from 'src/app/pages/timeline-render/components/timeline-scene-overlay.component';
import {
  TimelineSessionJumpTarget,
  buildTimelineSceneViewModel,
} from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';

describe('TimelineSceneOverlayComponent', () => {
  let fixture: ComponentFixture<TimelineSceneOverlayComponent>;
  let component: TimelineSceneOverlayComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimelineSceneOverlayComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TimelineSceneOverlayComponent);
    component = fixture.componentInstance;
  });

  it('renders markers, spans, row role badges, and excerpt pins with bound palette CSS variables and emits jump events', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: 420,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: [
        { id: 10, heightPx: 50 },
        { id: 20, heightPx: 50 },
      ],
      overlay: {
        palette: [
          {
            id: 'trigger',
            main: '#b45309',
            onMain: '#fafafa',
            soft: 'rgba(180, 83, 9, 0.13)',
            surface: '#f8eee6',
          },
          {
            id: 'victim',
            main: '#d32f2f',
            onMain: '#fafafa',
            soft: 'rgba(211, 47, 47, 0.13)',
            surface: '#fbeaea',
          },
        ],
        markers: [
          {
            id: 'm1',
            badge: '1',
            label: 'Readiness probe failed',
            style: TimelineMarkerStyle.PIN,
            paletteId: 'victim',
            timeMs: 1_020_000,
            timeLabel: '00:17:00',
            timelineId: 10,
            logId: 77,
          },
        ],
        spans: [
          {
            id: 's1',
            label: 'Outage window',
            startTimeMs: 1_010_000,
            endTimeMs: 1_050_000,
            durationLabel: '40s',
            pattern: TimelineSpanPattern.SOLID,
            paletteId: 'victim',
          },
        ],
        rowHighlights: [
          {
            id: 'rh1',
            roleLabel: 'TRIGGER · Node',
            timelineIds: [10],
            paletteId: 'trigger',
          },
        ],
        excerpts: [
          {
            id: 'ex1',
            kind: TimelineExcerptKind.LOG,
            badge: 'E1',
            paletteId: 'victim',
            timeMs: 1_020_000,
            timeLabel: '00:17:00',
            timelineId: 10,
            resourcePath: 'pod · default/api-0',
            title: 'k8s-event · Error',
            subtitle: '4 lines',
            logTimestamp: '2026-10-01T00:17:00Z',
            insertId: 'evt-probe-77',
            logId: 77,
            lines: [
              {
                kind: TimelineExcerptLineKind.SUMMARY,
                text: 'Probe failed: HTTP 503',
              },
            ],
          },
        ],
      },
    });

    fixture.componentRef.setInput('scene', vm);
    fixture.componentRef.setInput('interactiveMode', true);
    fixture.detectChanges();

    const pillEl = fixture.debugElement.query(By.css('.marker-pill'))
      .nativeElement as HTMLElement;
    expect(pillEl.style.getPropertyValue('--palette-main')).toBe('#d32f2f');
    const stemEl = fixture.debugElement.query(By.css('.marker-stem'))
      .nativeElement as HTMLElement;
    expect(getComputedStyle(stemEl).boxShadow).not.toBe('none');
    expect(fixture.debugElement.queryAll(By.css('.span-box')).length).toBe(1);
    expect(
      fixture.debugElement.queryAll(By.css('.span-bracket.left')).length,
    ).toBe(1);
    expect(
      fixture.debugElement.queryAll(By.css('.span-bracket.right')).length,
    ).toBe(1);
    expect(
      fixture.debugElement.query(By.css('.row-role-badge')).nativeElement
        .textContent,
    ).toContain('TRIGGER · Node');

    const pinEl = fixture.debugElement.query(By.css('.excerpt-pin-badge'));
    const pinNative = pinEl.nativeElement as HTMLElement;
    expect(pinNative.style.getPropertyValue('--palette-main')).toBe('#d32f2f');
    expect(getComputedStyle(pinNative).backgroundColor).toBe(
      'rgb(211, 47, 47)',
    );

    const jumpedTargets: TimelineSessionJumpTarget[] = [];
    component.jumpToSession.subscribe((t) => jumpedTargets.push(t));

    pinEl.triggerEventHandler('click');
    expect(jumpedTargets).toEqual([
      { timelineId: 10, revisionIndex: undefined, logId: 77 },
    ]);
  });

  it('shifts span label chip horizontally when a marker stem coincides with the span start edge', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: 420,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: [
        { id: 10, heightPx: 50 },
        { id: 20, heightPx: 50 },
      ],
      overlay: {
        markers: [
          {
            id: 'm1',
            badge: '1',
            label: 'Outage began',
            style: TimelineMarkerStyle.PIN,
            paletteId: 'default',
            timeMs: 1_010_000,
            timeLabel: '00:16:50',
            timelineId: 20,
          },
        ],
        spans: [
          {
            id: 's1',
            label: 'Outage window',
            startTimeMs: 1_010_000,
            endTimeMs: 1_050_000,
            durationLabel: '40s',
            pattern: TimelineSpanPattern.SOLID,
            paletteId: 'default',
          },
        ],
      },
    });

    fixture.componentRef.setInput('scene', vm);
    fixture.detectChanges();

    const leftBracketEl = fixture.debugElement.query(
      By.css('.span-bracket.left'),
    ).nativeElement as HTMLElement;
    expect(leftBracketEl.classList.contains('coincident')).toBe(true);

    const labelChipEl = fixture.debugElement.query(By.css('.span-label-chip'))
      .nativeElement as HTMLElement;
    expect(
      labelChipEl.parentElement?.classList.contains('scene-overlay-root'),
    ).toBe(true);
    expect(labelChipEl.style.left).toBe(`${vm.spans[0].chipStageLeftPx}px`);
    expect(labelChipEl.style.maxWidth).toBe(`${vm.spans[0].chipMaxWidthPx}px`);
  });
});
