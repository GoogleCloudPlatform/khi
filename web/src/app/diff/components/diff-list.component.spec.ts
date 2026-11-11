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

import { CdkVirtualScrollViewport } from '@angular/cdk/scrolling';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import {
  DiffListComponent,
  formatTimeLabel,
  parsePrincipal,
  PrincipalType,
} from 'src/app/diff/components/diff-list.component';
import { Log } from 'src/app/store/domain/log';
import { Revision, Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

function createMockData(rowCount: number): {
  timeline: ReadonlyDomainElement<Timeline>;
  logs: ReadonlyDomainElement<Log>[];
} {
  const logs = Array.from({ length: rowCount }, (_, i) => ({
    id: i + 1,
    logIndex: i,
    summary: `Log ${i}`,
  })) as unknown as ReadonlyDomainElement<Log>[];

  const revisions = Array.from({ length: rowCount }, (_, i) => ({
    id: i + 1,
    index: i,
    timelineId: 10,
    structId: i + 1,
    changedTime: BigInt(Date.UTC(2025, 0, 1, 0, 0, i)) * 1000000n,
    principal: 'system:kube-controller-manager',
    verb: {
      id: 1,
      label: 'UPDATE',
      backgroundColor: { r: 0.2, g: 0.4, b: 0.8, a: 1.0 },
      foregroundColor: { r: 1.0, g: 1.0, b: 1.0, a: 1.0 },
      visible: true,
    },
    logIndex: i,
  })) as unknown as ReadonlyDomainElement<Revision>[];

  const timeline = {
    id: 10,
    revisions,
  } as unknown as ReadonlyDomainElement<Timeline>;

  return { timeline, logs };
}

describe('DiffListComponent', () => {
  let fixture: ComponentFixture<DiffListComponent>;
  let component: DiffListComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DiffListComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(DiffListComponent);
    fixture.nativeElement.style.height = '100px';
    fixture.nativeElement.style.width = '300px';
    component = fixture.componentInstance;

    fixture.componentRef.setInput('timeline', null);
    fixture.componentRef.setInput('logs', []);
    fixture.componentRef.setInput('selectedLogIndex', -1);
    fixture.componentRef.setInput('selectedRevision', null);
    fixture.componentRef.setInput('highlightedLogIndices', new Set<number>());
    fixture.componentRef.setInput('timezoneShift', 0);
    fixture.autoDetectChanges();
    await fixture.whenStable();
  });

  it('keeps virtual scroll viewport mounted and scrolls initial URL selection into view', async () => {
    const viewportDebug = fixture.debugElement.query(
      By.directive(CdkVirtualScrollViewport),
    );
    expect(viewportDebug).not.toBeNull();
    const viewport = viewportDebug.injector.get(CdkVirtualScrollViewport);

    let scrollHeightAtCall = 0;
    const scrollSpy = spyOn(viewport, 'scrollToOffset').and.callFake(
      (offset: number) => {
        scrollHeightAtCall = viewport.elementRef.nativeElement.scrollHeight;
        viewport.elementRef.nativeElement.scrollTop = offset;
      },
    );

    const { timeline, logs } = createMockData(40);
    fixture.componentRef.setInput('timeline', timeline);
    fixture.componentRef.setInput('logs', logs);
    fixture.componentRef.setInput('selectedLogIndex', 25);
    fixture.componentRef.setInput('selectedRevision', timeline.revisions[25]);
    fixture.detectChanges();

    await new Promise(requestAnimationFrame);
    await new Promise(requestAnimationFrame);
    fixture.detectChanges();

    // rowIndex = 25, rowHeight = 13 -> scrollToIndex(25) calls scrollToOffset(325, 'smooth').
    expect(scrollSpy).toHaveBeenCalledTimes(1);
    expect(scrollSpy).toHaveBeenCalledWith(325, 'smooth');
    expect(scrollHeightAtCall).toBeGreaterThanOrEqual(40 * 13 - 1);

    const viewportRect =
      viewport.elementRef.nativeElement.getBoundingClientRect();
    const selectedRowEl = fixture.nativeElement.querySelector(
      '.revision-row.selected',
    ) as HTMLElement | null;
    expect(selectedRowEl).not.toBeNull();
    const rowRect = selectedRowEl!.getBoundingClientRect();
    expect(rowRect.height).toBeCloseTo(13, 0);
    expect(rowRect.top).toBeGreaterThanOrEqual(viewportRect.top - 1);
    expect(rowRect.bottom).toBeLessThanOrEqual(viewportRect.bottom + 1);
  });

  it('does not overshoot first visible revision when timeline starts with an inferred logIndex -1 revision', async () => {
    const viewport = fixture.debugElement
      .query(By.directive(CdkVirtualScrollViewport))
      .injector.get(CdkVirtualScrollViewport);
    const scrollSpy = spyOn(viewport, 'scrollToIndex');

    const { timeline, logs } = createMockData(20);
    const inferredRev = {
      ...timeline.revisions[0],
      id: 999,
      index: 0,
      logIndex: -1,
    } as unknown as ReadonlyDomainElement<Revision>;
    const shiftedRevisions = [
      inferredRev,
      ...timeline.revisions.map((rev, idx) => ({
        ...rev,
        index: idx + 1,
      })),
    ] as unknown as ReadonlyDomainElement<Revision>[];
    const timelineWithInferred = {
      id: 10,
      revisions: shiftedRevisions,
    } as unknown as ReadonlyDomainElement<Timeline>;

    // Revision index 1 is the very first renderable row (y = 0), so it is already visible at scrollTop = 0.
    fixture.componentRef.setInput('timeline', timelineWithInferred);
    fixture.componentRef.setInput('logs', logs);
    fixture.componentRef.setInput('selectedLogIndex', 0);
    fixture.componentRef.setInput(
      'selectedRevision',
      timelineWithInferred.revisions[1],
    );
    fixture.detectChanges();
    await new Promise(requestAnimationFrame);

    expect(scrollSpy).not.toHaveBeenCalled();
    expect(component.rowViewModels().length).toBe(20);
    expect(component.rowViewModels()[0].index).toBe(1);
  });

  it('scrolls once logs arrive after timeline and revision selection', async () => {
    const viewport = fixture.debugElement
      .query(By.directive(CdkVirtualScrollViewport))
      .injector.get(CdkVirtualScrollViewport);
    const scrollSpy = spyOn(viewport, 'scrollToIndex');

    const { timeline, logs } = createMockData(30);
    fixture.componentRef.setInput('timeline', timeline);
    fixture.componentRef.setInput('selectedLogIndex', 20);
    fixture.componentRef.setInput('selectedRevision', timeline.revisions[20]);
    fixture.detectChanges();
    await new Promise(requestAnimationFrame);
    expect(scrollSpy).not.toHaveBeenCalled();

    fixture.componentRef.setInput('logs', logs);
    fixture.detectChanges();
    await new Promise(requestAnimationFrame);

    expect(scrollSpy).toHaveBeenCalledTimes(1);
    expect(scrollSpy).toHaveBeenCalledWith(20, 'smooth');
  });

  it('suppresses scroll on direct row click without blocking subsequent external navigation', async () => {
    const viewport = fixture.debugElement
      .query(By.directive(CdkVirtualScrollViewport))
      .injector.get(CdkVirtualScrollViewport);

    const { timeline, logs } = createMockData(30);
    fixture.componentRef.setInput('timeline', timeline);
    fixture.componentRef.setInput('logs', logs);
    fixture.detectChanges();
    await new Promise(requestAnimationFrame);

    const scrollSpy = spyOn(viewport, 'scrollToIndex');

    // Direct row click inside DiffListComponent should not auto-scroll.
    component['_selectRevision'](timeline.revisions[10]);
    fixture.componentRef.setInput('selectedLogIndex', 10);
    fixture.componentRef.setInput('selectedRevision', timeline.revisions[10]);
    fixture.detectChanges();
    await new Promise(requestAnimationFrame);
    expect(scrollSpy).not.toHaveBeenCalled();

    // Clicking the already-selected row again must not suppress a later external selection change.
    component['_selectRevision'](timeline.revisions[10]);
    fixture.detectChanges();

    fixture.componentRef.setInput('selectedLogIndex', 18);
    fixture.componentRef.setInput('selectedRevision', timeline.revisions[18]);
    fixture.detectChanges();
    await new Promise(requestAnimationFrame);

    expect(scrollSpy).toHaveBeenCalledTimes(1);
    expect(scrollSpy).toHaveBeenCalledWith(18, 'smooth');

    // Unrelated timezoneShift updates should not re-trigger scroll.
    scrollSpy.calls.reset();
    fixture.componentRef.setInput('timezoneShift', 9);
    fixture.detectChanges();
    await new Promise(requestAnimationFrame);
    expect(scrollSpy).not.toHaveBeenCalled();
  });
});

describe('DiffListComponent utils', () => {
  describe('formatTimeLabel', () => {
    it('formats time with 0 timezone shift', () => {
      // 2024-01-01T12:34:56Z
      const timeNs = BigInt(Date.UTC(2024, 0, 1, 12, 34, 56)) * 1000000n;
      expect(formatTimeLabel(timeNs, 0)).toBe('12:34:56');
    });

    it('formats time with positive timezone shift', () => {
      // +9 hours (JST)
      const timeNs = BigInt(Date.UTC(2024, 0, 1, 12, 34, 56)) * 1000000n;
      expect(formatTimeLabel(timeNs, 9)).toBe('21:34:56');
    });

    it('formats time with negative timezone shift', () => {
      // -5 hours (EST)
      const timeNs = BigInt(Date.UTC(2024, 0, 1, 12, 34, 56)) * 1000000n;
      expect(formatTimeLabel(timeNs, -5)).toBe('07:34:56');
    });

    it('handles padding correctly for single digit times', () => {
      // 09:05:01
      const timeNs = BigInt(Date.UTC(2024, 0, 1, 9, 5, 1)) * 1000000n;
      expect(formatTimeLabel(timeNs, 0)).toBe('09:05:01');
    });

    it('returns ??:??:?? if timeNs is 0n', () => {
      expect(formatTimeLabel(0n, 0)).toBe('??:??:??');
    });
  });

  describe('parsePrincipal', () => {
    it('returns empty/NotAvailable when value is empty', () => {
      const principal = parsePrincipal('');
      expect(principal.type).toBe(PrincipalType.NotAvailable);
      expect(principal.short).toBe('');
      expect(principal.full).toBe('');
    });

    it('handles system:serviceaccount: principals', () => {
      const principal = parsePrincipal(
        'system:serviceaccount:kube-system:default',
      );
      expect(principal.type).toBe(PrincipalType.ServiceAccount);
      expect(principal.short).toBe('kube-system:default');
      expect(principal.full).toBe('system:serviceaccount:kube-system:default');
    });

    it('handles system:node: principals', () => {
      const principal = parsePrincipal('system:node:gke-cluster-pool-123');
      expect(principal.type).toBe(PrincipalType.Node);
      expect(principal.short).toBe('gke-cluster-pool-123');
      expect(principal.full).toBe('system:node:gke-cluster-pool-123');
    });

    it('handles system: principals', () => {
      const principal = parsePrincipal('system:kube-controller-manager');
      expect(principal.type).toBe(PrincipalType.System);
      expect(principal.short).toBe('kube-controller-manager');
      expect(principal.full).toBe('system:kube-controller-manager');
    });

    it('defaults to User type for other values', () => {
      const principal = parsePrincipal('user@example.com');
      expect(principal.type).toBe(PrincipalType.User);
      expect(principal.short).toBe('user@example.com');
      expect(principal.full).toBe('user@example.com');
    });
  });
});
