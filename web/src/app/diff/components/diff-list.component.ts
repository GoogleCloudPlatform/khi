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
  Component,
  DestroyRef,
  computed,
  effect,
  inject,
  input,
  output,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  CdkVirtualScrollViewport,
  FixedSizeVirtualScrollStrategy,
  ScrollingModule,
  VIRTUAL_SCROLL_STRATEGY,
} from '@angular/cdk/scrolling';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Timeline, Revision } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { RendererConvertUtil } from 'src/app/timeline/components/canvas/convertutil';
import { BigIntTimeUtil } from 'src/app/utils/bigint-time-util';

const DIFF_LIST_ROW_HEIGHT_PX = 13;
const DIFF_LIST_SCROLL_MARGIN_PX = DIFF_LIST_ROW_HEIGHT_PX * 3;
const MAX_SCROLL_RETRY_FRAMES = 10;

class DiffListScrollStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    super(DIFF_LIST_ROW_HEIGHT_PX, 100, 1000);
  }
}

export enum PrincipalType {
  System = 'System',
  Node = 'Node',
  ServiceAccount = 'SA',
  User = 'User',
  Invalid = 'Invalid',
  NotAvailable = 'N/A',
}

export interface ResourceOperatorPrincipal {
  type: PrincipalType;
  full: string;
  short: string;
}

export interface DiffListRowViewModel {
  revision: ReadonlyDomainElement<Revision>;
  log: ReadonlyDomainElement<Log> | null;
  index: number;
  isContentChanged: boolean;
  timeLabel: string;
  author: ResourceOperatorPrincipal | null;
  verbLabel: string;
  verbStyle: {
    backgroundColor: string;
    color: string;
  };
}

/**
 * Formats a given timestamp in nanoseconds into a HH:mm:ss string, adjusted by a timezone shift.
 * @param timeNs The absolute time in nanoseconds since epoch.
 * @param timezoneShiftHours The timezone offset in hours to apply.
 * @returns A string representation of the time in HH:mm:ss format.
 */
export function formatTimeLabel(
  timeNs: bigint,
  timezoneShiftHours: number,
): string {
  if (timeNs === 0n) {
    return '??:??:??';
  }
  const timeInMs = BigIntTimeUtil.NsToNumberMs(timeNs);
  const d = new Date(timeInMs + timezoneShiftHours * 60 * 60 * 1000);
  const h = d.getUTCHours().toString().padStart(2, '0');
  const m = d.getUTCMinutes().toString().padStart(2, '0');
  const s = d.getUTCSeconds().toString().padStart(2, '0');
  return `${h}:${m}:${s}`;
}

/**
 * Parses a Kubernetes principal string (e.g., 'system:serviceaccount:...') into a structured ResourceOperatorPrincipal.
 * @param value The raw principal string from the resource requestor field.
 * @returns A structured object containing the principal type, short name, and full original name.
 */
export function parsePrincipal(value: string): ResourceOperatorPrincipal {
  const result: ResourceOperatorPrincipal = {
    type: PrincipalType.User,
    full: value,
    short: value,
  };
  if (value === '') {
    result.type = PrincipalType.NotAvailable;
    result.full = '';
    result.short = '';
  }
  if (value.startsWith('system:serviceaccount:')) {
    result.type = PrincipalType.ServiceAccount;
    result.short = value.split('system:serviceaccount:')[1];
  } else if (value.startsWith('system:node:')) {
    result.type = PrincipalType.Node;
    result.short = value.split('system:node:')[1];
  } else if (value.startsWith('system:')) {
    result.type = PrincipalType.System;
    result.short = value.split('system:')[1];
  }
  return result;
}

/**
 * Component for displaying a virtual scrolling list of resource revisions.
 */
@Component({
  selector: 'khi-diff-list',
  templateUrl: './diff-list.component.html',
  styleUrls: ['./diff-list.component.scss'],
  imports: [
    CommonModule,
    ScrollingModule,
    CdkVirtualScrollViewport,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  providers: [
    { provide: VIRTUAL_SCROLL_STRATEGY, useClass: DiffListScrollStrategy },
  ],
})
export class DiffListComponent {
  /**
   * The selected timeline containing revisions.
   */
  readonly timeline = input.required<ReadonlyDomainElement<Timeline> | null>();

  /**
   * The index of the currently selected log.
   */
  readonly selectedLogIndex = input.required<number>();

  /**
   * The currently selected revision, used to disambiguate when multiple revisions share the same log index.
   */
  readonly selectedRevision = input<ReadonlyDomainElement<Revision> | null>(
    null,
  );

  /**
   * Set of indices for highlighted logs.
   */
  readonly highlightedLogIndices = input.required<Set<number>>();

  /**
   * Array of all log entries.
   */
  readonly logs = input.required<ReadonlyDomainElement<Log>[]>();

  /**
   * Timezone shift in hours to adjust the displayed timestamps.
   */
  readonly timezoneShift = input.required<number>();

  /**
   * Computed array of ViewModels representing each row in the list.
   */
  readonly rowViewModels = computed<DiffListRowViewModel[]>(() => {
    const tl = this.timeline();
    const ls = this.logs();
    if (!tl || ls.length === 0) return [];

    const shift = this.timezoneShift();
    const rows: DiffListRowViewModel[] = [];

    for (let index = 0; index < tl.revisions.length; index++) {
      const rev = tl.revisions[index];
      const log = rev.logIndex !== -1 ? (ls[rev.logIndex] ?? null) : null;
      if (log === null) {
        continue;
      }

      let isContentChanged = true;
      if (index > 0) {
        const prevRev = tl.revisions[index - 1];
        isContentChanged = rev.structId !== prevRev.structId;
      }

      const verb = rev.verb;
      const bg = RendererConvertUtil.hdrColorToCSSColor([
        verb.backgroundColor.r,
        verb.backgroundColor.g,
        verb.backgroundColor.b,
        verb.backgroundColor.a,
      ]);
      const fg = RendererConvertUtil.hdrColorToCSSColor([
        verb.foregroundColor.r,
        verb.foregroundColor.g,
        verb.foregroundColor.b,
        verb.foregroundColor.a,
      ]);

      rows.push({
        revision: rev,
        log,
        index,
        isContentChanged,
        timeLabel: formatTimeLabel(rev.changedTime, shift),
        author: rev.principal ? parsePrincipal(rev.principal) : null,
        verbLabel: verb.label,
        verbStyle: {
          backgroundColor: bg,
          color: fg,
        },
      });
    }

    return rows;
  });

  /**
   * Emitted when a revision is selected by clicking.
   */
  readonly selectRevision = output<ReadonlyDomainElement<Revision>>();

  /**
   * Emitted when a revision is highlighted by hovering.
   */
  readonly highlightRevision = output<ReadonlyDomainElement<Revision>>();

  /**
   * Emitted when keyboard navigation (Up/Down) commands are received.
   */
  readonly moveSelection = output<'next' | 'prev'>();

  private readonly destroyRef = inject(DestroyRef);
  private readonly viewPort = viewChild(CdkVirtualScrollViewport);

  private lastScrolledSelectionKey: string | null = null;
  private pendingScrollRafId: number | null = null;

  constructor() {
    this.destroyRef.onDestroy(() => {
      this.cancelPendingScroll();
    });

    effect(() => {
      const rows = this.rowViewModels();
      const index = this.selectedLogIndex();
      const selectedRev = this.selectedRevision();
      const timeline = this.timeline();
      const viewPort = this.viewPort();

      if (timeline === null) {
        this.lastScrolledSelectionKey = null;
        this.cancelPendingScroll();
        return;
      }

      const revisionIndex = this.resolveRevisionIndex(
        timeline,
        selectedRev,
        index,
      );
      if (revisionIndex === -1) {
        this.lastScrolledSelectionKey = null;
        this.cancelPendingScroll();
        return;
      }

      if (rows.length === 0 || !viewPort) {
        this.cancelPendingScroll();
        return;
      }

      const rowArrayIndex = rows.findIndex(
        (row) => row.index === revisionIndex,
      );
      if (rowArrayIndex === -1) {
        this.lastScrolledSelectionKey = null;
        this.cancelPendingScroll();
        return;
      }

      const selectionKey = `${timeline.id}:${revisionIndex}`;
      if (selectionKey === this.lastScrolledSelectionKey) {
        return;
      }

      this.lastScrolledSelectionKey = selectionKey;
      this.scheduleScrollToRow(viewPort, rowArrayIndex, rows.length);
    });
  }

  protected _selectRevision(r: ReadonlyDomainElement<Revision>) {
    const timeline = this.timeline();
    if (timeline !== null) {
      const revisionIndex = r.index ?? timeline.revisions.indexOf(r);
      if (revisionIndex !== -1) {
        this.cancelPendingScroll();
        this.lastScrolledSelectionKey = `${timeline.id}:${revisionIndex}`;
      }
    }
    this.selectRevision.emit(r);
  }

  protected _highlightRevision(r: ReadonlyDomainElement<Revision>) {
    this.highlightRevision.emit(r);
  }

  /**
   * Handles keyboard navigation for the list.
   */
  public keyDown(keyEvent: KeyboardEvent) {
    if (keyEvent.key === 'ArrowDown') {
      this.moveSelection.emit('next');
      keyEvent.preventDefault();
    }
    if (keyEvent.key === 'ArrowUp') {
      this.moveSelection.emit('prev');
      keyEvent.preventDefault();
    }
  }

  private resolveRevisionIndex(
    timeline: ReadonlyDomainElement<Timeline>,
    selectedRev: ReadonlyDomainElement<Revision> | null,
    selectedLogIndex: number,
  ): number {
    if (selectedRev !== null && selectedRev.timelineId === timeline.id) {
      return selectedRev.index ?? timeline.revisions.indexOf(selectedRev);
    }
    return timeline.revisions.findIndex(
      (rev) => rev.logIndex === selectedLogIndex,
    );
  }

  private scheduleScrollToRow(
    viewPort: CdkVirtualScrollViewport,
    rowArrayIndex: number,
    expectedRowCount: number,
  ): void {
    this.cancelPendingScroll();
    const expectedContentHeight = expectedRowCount * DIFF_LIST_ROW_HEIGHT_PX;
    let remainingFrames = MAX_SCROLL_RETRY_FRAMES;

    // Defers scrolling until CdkVirtualScrollViewport updates its spacer height and layout metrics.
    const attemptScroll = () => {
      this.pendingScrollRafId = null;
      viewPort.checkViewportSize();

      const element = viewPort.elementRef.nativeElement;
      const viewportHeight = element.clientHeight;
      const isLayoutReady =
        viewportHeight > 0 && element.scrollHeight >= expectedContentHeight - 1;
      if (!isLayoutReady && remainingFrames > 0) {
        remainingFrames--;
        this.pendingScrollRafId = requestAnimationFrame(attemptScroll);
        return;
      }

      const rowTop = rowArrayIndex * DIFF_LIST_ROW_HEIGHT_PX;
      const rowBottom = rowTop + DIFF_LIST_ROW_HEIGHT_PX;
      if (viewportHeight <= 0) {
        viewPort.scrollToOffset(
          Math.max(0, rowTop - DIFF_LIST_SCROLL_MARGIN_PX),
          'smooth',
        );
        return;
      }

      const marginPx = Math.min(
        DIFF_LIST_SCROLL_MARGIN_PX,
        Math.max(0, Math.floor((viewportHeight - DIFF_LIST_ROW_HEIGHT_PX) / 2)),
      );
      const maxAllowedScrollTop = Math.max(
        0,
        expectedContentHeight - viewportHeight,
      );
      const minScrollTop = Math.min(
        maxAllowedScrollTop,
        Math.max(0, rowBottom - viewportHeight + marginPx),
      );
      const maxScrollTop = Math.min(
        maxAllowedScrollTop,
        Math.max(0, rowTop - marginPx),
      );
      const currentScrollTop = element.scrollTop;
      if (
        currentScrollTop >= minScrollTop &&
        currentScrollTop <= maxScrollTop
      ) {
        return;
      }

      const centeredScrollTop = Math.min(
        maxAllowedScrollTop,
        Math.max(
          0,
          Math.round(rowTop - (viewportHeight - DIFF_LIST_ROW_HEIGHT_PX) / 2),
        ),
      );
      viewPort.scrollToOffset(centeredScrollTop, 'smooth');
    };

    this.pendingScrollRafId = requestAnimationFrame(attemptScroll);
  }

  private cancelPendingScroll(): void {
    if (this.pendingScrollRafId !== null) {
      cancelAnimationFrame(this.pendingScrollRafId);
      this.pendingScrollRafId = null;
    }
  }
}
