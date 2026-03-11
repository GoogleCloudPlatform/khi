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
  computed,
  effect,
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
import { ResourceRevision } from '../../store/revision';
import { ResourceTimeline } from '../../store/timeline';
import { LogEntry } from '../../store/log';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';

class DiffListScrollStrategy extends FixedSizeVirtualScrollStrategy {
  constructor() {
    super(13, 100, 1000);
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
  revision: ResourceRevision;
  log: LogEntry | null;
  index: number;
  isContentChanged: boolean;
  timeLabel: string;
  author: ResourceOperatorPrincipal | null;
}

/**
 * Formats a given timestamp in milliseconds into a HH:mm:ss string, adjusted by a timezone shift.
 * @param timeInMs The absolute time in milliseconds since epoch.
 * @param timezoneShiftHours The timezone offset in hours to apply.
 * @returns A string representation of the time in HH:mm:ss format.
 */
export function formatTimeLabel(
  timeInMs: number,
  timezoneShiftHours: number,
): string {
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
  readonly timeline = input.required<ResourceTimeline | null>();

  /**
   * The index of the currently selected log.
   */
  readonly selectedLogIndex = input.required<number>();

  /**
   * Set of indices for highlighted logs.
   */
  readonly highlightedLogIndices = input.required<Set<number>>();

  /**
   * Array of all log entries.
   */
  readonly logs = input.required<LogEntry[]>();

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

    return tl.revisions.map((rev, index) => {
      let isContentChanged = true;
      if (index > 0) {
        const prevRev = tl.revisions[index - 1];
        isContentChanged = rev.resourceContent !== prevRev.resourceContent;
      }

      const log = rev.logIndex !== -1 ? ls[rev.logIndex] : null;

      return {
        revision: rev,
        log: log,
        index: index,
        isContentChanged: isContentChanged,
        timeLabel: formatTimeLabel(rev.startAt, shift),
        author: rev.requestor ? parsePrincipal(rev.requestor) : null,
      };
    });
  });

  /**
   * Emitted when a revision is selected by clicking.
   */
  readonly selectRevision = output<ResourceRevision>();

  /**
   * Emitted when a revision is highlighted by hovering.
   */
  readonly highlightRevision = output<ResourceRevision>();

  /**
   * Emitted when keyboard navigation (Up/Down) commands are received.
   */
  readonly moveSelection = output<'next' | 'prev'>();

  private readonly viewPort = viewChild(CdkVirtualScrollViewport);

  private disableScrollForNext = false;

  constructor() {
    effect(() => {
      const index = this.selectedLogIndex();
      const timeline = this.timeline();
      const viewPort = this.viewPort();

      if (this.disableScrollForNext) {
        this.disableScrollForNext = false;
        return;
      }
      if (timeline === null) {
        return;
      }
      const revisionIndex = timeline.revisions.findIndex(
        (rev) => rev.logIndex === index,
      );
      if (revisionIndex !== -1) {
        viewPort?.scrollToIndex(revisionIndex, 'smooth');
      }
    });
  }

  protected _selectRevision(r: ResourceRevision) {
    this.disableScrollForNext = true;
    this.selectRevision.emit(r);
  }

  protected _highlightRevision(r: ResourceRevision) {
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
}
