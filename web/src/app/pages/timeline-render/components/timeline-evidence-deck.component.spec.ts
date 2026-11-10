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
} from 'src/app/common/schema/timeline-render-types';
import { TimelineEvidenceDeckComponent } from 'src/app/pages/timeline-render/components/timeline-evidence-deck.component';
import { buildTimelineSceneViewModel } from 'src/app/pages/timeline-render/timeline-render-scene-builder';
import { TimelineSessionJumpTarget } from 'src/app/pages/timeline-render/types/timeline-render-scene.viewmodel';

describe('TimelineEvidenceDeckComponent', () => {
  let fixture: ComponentFixture<TimelineEvidenceDeckComponent>;
  let component: TimelineEvidenceDeckComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimelineEvidenceDeckComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TimelineEvidenceDeckComponent);
    component = fixture.componentInstance;
  });

  it('renders diff/log cards with synchronized palette colors, Cloud Logging query metadata, query highlights, and jump buttons', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: 420,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: [{ id: 10, heightPx: 50 }],
      overlay: {
        palette: [
          {
            id: 'victim',
            main: '#d32f2f',
            onMain: '#fafafa',
            soft: 'rgba(211, 47, 47, 0.13)',
            surface: '#fbeaea',
          },
        ],
        excerpts: [
          {
            id: 'ex-diff',
            kind: TimelineExcerptKind.DIFF,
            badge: 'E1',
            paletteId: 'victim',
            timeMs: 1_010_000,
            timeLabel: '00:16:50',
            timelineId: 10,
            resourcePath: 'node · gke-node-1',
            title: 'Patch · Resource exists',
            subtitle: '+1 / -1 · 2 diff lines',
            logTimestamp: '2026-10-01T00:16:50.123456789Z',
            insertId: '1abc2def3ghi',
            revisionIndex: 2,
            lines: [
              {
                kind: TimelineExcerptLineKind.DELETED,
                oldLineNum: 14,
                text: 'status: True',
              },
              {
                kind: TimelineExcerptLineKind.ADDED,
                newLineNum: 14,
                text: 'status: False',
                parts: [{ text: 'status: ' }, { text: 'False', matched: true }],
              },
              {
                kind: TimelineExcerptLineKind.OMITTED,
                text: '8 unchanged lines omitted',
                omitted: 8,
              },
            ],
            note: 'Kubelet stopped posting status',
          },
          {
            id: 'ex-log-no-meta',
            kind: TimelineExcerptKind.LOG,
            badge: 'E2',
            paletteId: 'victim',
            timeMs: 1_020_000,
            timeLabel: '00:17:00',
            timelineId: 10,
            resourcePath: 'node · gke-node-1',
            title: 'k8s-event · Warning',
            subtitle: '1 line',
            logId: 44,
            lines: [
              {
                kind: TimelineExcerptLineKind.SUMMARY,
                text: 'Node status unknown',
              },
            ],
          },
        ],
      },
    });

    fixture.componentRef.setInput('deck', vm.deck);
    fixture.componentRef.setInput('stageWidthPx', vm.stageWidthPx);
    fixture.componentRef.setInput('interactiveMode', true);
    fixture.detectChanges();

    const cards = fixture.debugElement.queryAll(By.css('.evidence-card'));
    expect(cards.length).toBe(2);

    const cardEl = cards[0].nativeElement as HTMLElement;
    const tetherEl = fixture.debugElement.query(By.css('.tether-group'))
      .nativeElement as SVGGElement;

    expect(cardEl.style.getPropertyValue('--palette-main')).toBe('#d32f2f');
    expect(tetherEl.style.getPropertyValue('--palette-main')).toBe('#d32f2f');

    const queryRow = cards[0].query(By.css('.header-query-row'));
    expect(queryRow).not.toBeNull();
    const chipTexts = queryRow!
      .queryAll(By.css('.query-chip'))
      .map((el) => (el.nativeElement as HTMLElement).textContent?.trim());
    expect(chipTexts).toEqual([
      'timestamp="2026-10-01T00:16:50.123456789Z"',
      'insertId="1abc2def3ghi"',
    ]);
    expect(cardEl.textContent).not.toContain('rev#');
    expect(cardEl.textContent).not.toContain('logId');

    expect(cards[1].query(By.css('.header-query-row'))).toBeNull();

    expect(
      fixture.debugElement.query(By.css('mark.query-match')).nativeElement
        .textContent,
    ).toBe('False');
    expect(
      fixture.debugElement.query(By.css('.omitted-label')).nativeElement
        .textContent,
    ).toContain('8 unchanged lines omitted');
    expect(
      fixture.debugElement.query(By.css('.note-text')).nativeElement
        .textContent,
    ).toBe('Kubelet stopped posting status');

    const jumped: TimelineSessionJumpTarget[] = [];
    component.jumpToSession.subscribe((t) => jumped.push(t));

    cards[0].query(By.css('button.jump-btn'))!.triggerEventHandler('click');
    expect(jumped).toEqual([
      { timelineId: 10, revisionIndex: 2, logId: undefined },
    ]);
  });

  it('routes second-row tethers behind first-row cards to the measured top border of wrapped cards', () => {
    const vm = buildTimelineSceneViewModel({
      indexPaneWidthPx: 300,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: [{ id: 10, heightPx: 50 }],
      overlay: {
        excerpts: [
          {
            id: 'ex-1',
            kind: TimelineExcerptKind.DIFF,
            badge: 'E1',
            paletteId: 'default',
            timeMs: 1_010_000,
            timeLabel: '00:16:50',
            timelineId: 10,
            resourcePath: 'node · gke-node-1',
            title: 'Patch · Resource exists',
            subtitle: '+3 / -3 · 8 diff lines',
            lines: [
              { kind: TimelineExcerptLineKind.DELETED, text: 'a: 1' },
              { kind: TimelineExcerptLineKind.ADDED, text: 'a: 2' },
              { kind: TimelineExcerptLineKind.CONTEXT, text: 'b: 3' },
              { kind: TimelineExcerptLineKind.CONTEXT, text: 'c: 4' },
              { kind: TimelineExcerptLineKind.CONTEXT, text: 'd: 5' },
            ],
            note: 'Taller first-row card',
          },
          {
            id: 'ex-2',
            kind: TimelineExcerptKind.LOG,
            badge: 'E2',
            paletteId: 'default',
            timeMs: 1_020_000,
            timeLabel: '00:17:00',
            timelineId: 10,
            resourcePath: 'pod · default/app-0',
            title: 'container · ERROR',
            subtitle: '1 line',
            lines: [
              {
                kind: TimelineExcerptLineKind.SUMMARY,
                text: 'w'.repeat(220),
              },
            ],
          },
          {
            id: 'ex-3',
            kind: TimelineExcerptKind.LOG,
            badge: 'E3',
            paletteId: 'default',
            timeMs: 1_030_000,
            timeLabel: '00:17:10',
            timelineId: 10,
            resourcePath: 'pod · default/app-0',
            title: 'k8s-audit · ERROR',
            subtitle: '2 lines',
            lines: [
              { kind: TimelineExcerptLineKind.SUMMARY, text: '409 Conflict' },
            ],
          },
          {
            id: 'ex-4',
            kind: TimelineExcerptKind.LOG,
            badge: 'E4',
            paletteId: 'default',
            timeMs: 1_040_000,
            timeLabel: '00:17:20',
            timelineId: 10,
            resourcePath: 'pod · default/exporter',
            title: 'container · ERROR',
            subtitle: '1 line',
            lines: [
              {
                kind: TimelineExcerptLineKind.SUMMARY,
                text: 'DeadlineExceeded',
              },
            ],
          },
        ],
      },
    });

    fixture.componentRef.setInput('deck', vm.deck);
    fixture.componentRef.setInput('stageWidthPx', vm.stageWidthPx);
    fixture.detectChanges();
    component.syncMeasuredAnchors();
    fixture.detectChanges();

    const tetherSvg = fixture.debugElement.query(By.css('svg.tether-strip'))
      .nativeElement as SVGSVGElement;
    const cardsGrid = fixture.debugElement.query(By.css('.cards-grid'))
      .nativeElement as HTMLElement;
    expect(Number(getComputedStyle(tetherSvg).zIndex)).toBeLessThan(
      Number(getComputedStyle(cardsGrid).zIndex),
    );

    const cardElements = fixture.debugElement
      .queryAll(By.css('.evidence-card'))
      .map((de) => de.nativeElement as HTMLElement);
    const joints = fixture.debugElement
      .queryAll(By.css('circle.tether-joint'))
      .map((de) => de.nativeElement as SVGCircleElement);
    const curves = fixture.debugElement
      .queryAll(By.css('path.tether-curve'))
      .map((de) => de.nativeElement as SVGPathElement);

    expect(cardElements.length).toBe(4);
    expect(joints.length).toBe(4);

    const row0TopY = cardsGrid.offsetTop + cardElements[0].offsetTop;
    const row1TopY = cardsGrid.offsetTop + cardElements[2].offsetTop;
    expect(row1TopY).toBeGreaterThan(
      row0TopY + cardElements[0].offsetHeight + 10,
    );

    expect(Number(joints[0].getAttribute('cy'))).toBeCloseTo(row0TopY - 2, 1);
    expect(Number(joints[2].getAttribute('cy'))).toBeCloseTo(row1TopY - 2, 1);
    expect(Number(joints[3].getAttribute('cy'))).toBeCloseTo(row1TopY - 2, 1);
    expect(curves[2].getAttribute('d') ?? '').toContain(` ${row1TopY}`);
    expect(curves[3].getAttribute('d') ?? '').toContain(` ${row1TopY}`);
  });
});
