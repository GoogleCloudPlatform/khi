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

import {
  Component,
  computed,
  input,
  output,
  OutputEmitterRef,
} from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { RevisionStateStyle } from 'src/app/store/domain/style';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { Revision, Event, Timeline } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { MatTooltip } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import {
  TimelineChartItemHighlight,
  TimelineChartItemHighlightType,
} from './interaction-model';
import { RendererConvertUtil } from './canvas/convertutil';
import { TimelineChartMouseEvent } from './timeline-chart.component';

export interface TimelineHoverOverlay {
  timeline: ReadonlyDomainElement<Timeline>;
  revisions: ReadonlyDomainElement<Revision>[];
  events: ReadonlyDomainElement<Event>[];
  initialRevision: ReadonlyDomainElement<Revision> | null;
}

enum StatusContinousMode {
  StartAndEnd,
  Start,
  Middle,
  End,
}

interface TimelineHoverOverlayLogItem {
  uniqueID: string;
  log: ReadonlyDomainElement<Log>;
  revision?: ReadonlyDomainElement<Revision>;
  event?: ReadonlyDomainElement<Event>;
  logIndex: number;
  isRevision: boolean;
  revisionStateColor: string;
  revisionStateLabel: string;
  revisionStateIcon: string;
  revisionStateStyle: RevisionStateStyle;
  lastRevisionLog: ReadonlyDomainElement<Log> | null;
  logTypeColor: string;
  logTypeLabel: string;
  verbTypeColor: string;
  verbTypeLabel: string;
  showSeverity: boolean;
  severityColor: string;
  severityLabel: string;
  timeMs: number;
  timeLabel: string;
  summary: string;
  statusContinous: StatusContinousMode;
  highlightType: TimelineChartItemHighlightType;
  lastRevisionHightlightType: TimelineChartItemHighlightType;
}

interface InitialRevisionStateItem {
  revisionStateStyle: RevisionStateStyle;
  revisionStateColor: string;
  statusContinous: StatusContinousMode;
}

interface TimelineHoverOverlayViewModel {
  timeline: ReadonlyDomainElement<Timeline> | null;
  logs: TimelineHoverOverlayLogItem[];
  initialRevisionState: InitialRevisionStateItem | null;
}

/**
 * The `TimelineHoverOverlayComponent` displays a detailed overlay when hovering over a timeline.
 * It shows a list of logs (revisions and events) around mouse points associated with the focused timeline,
 * providing context about the resource's history at that point in time.
 */
@Component({
  selector: 'khi-timeline-hover-overlay',
  templateUrl: './timeline-hover-overlay.component.html',
  styleUrls: ['./timeline-hover-overlay.component.scss'],
  imports: [MatIconModule, KHIIconRegistrationModule, MatTooltip, CommonModule],
})
export class TimelineHoverOverlayComponent {
  protected readonly StatusContinousMode = StatusContinousMode;
  protected readonly RevisionStateStyle = RevisionStateStyle;
  protected readonly TimelineChartItemHighlightType =
    TimelineChartItemHighlightType;

  /**
   * The data for the overlay, including the focused timeline and associated revisions/events.
   */
  timelineHoverOverlay = input<TimelineHoverOverlay | null>(null);

  /**
   * Timezone shift in hours to adjust the displayed timestamps.
   */
  timezoneShiftHours = input(0);

  /**
   * Emitted when hovering over a specific element in the overlay list.
   */
  hoverOnElement = output<TimelineChartMouseEvent>();

  /**
   * Emitted when clicking on a specific element in the overlay list.
   */
  clickOnElement = output<TimelineChartMouseEvent>();

  /**
   * Map of highlights to apply to specific log indices in the overlay.
   */
  highlights = input<TimelineChartItemHighlight>({});

  handleMouse(
    e: MouseEvent,
    timeline: ReadonlyDomainElement<Timeline>,
    log: TimelineHoverOverlayLogItem,
    o: OutputEmitterRef<TimelineChartMouseEvent>,
  ) {
    const base: TimelineChartMouseEvent = {
      event: e,
      timeline,
      timeMS: log.timeMs,
      clientX: e.clientX,
      clientY: e.clientY,
    };
    if (log.isRevision) {
      o.emit({
        ...base,
        revisionIndex: timeline.revisions.indexOf(log.revision!),
      });
    } else {
      o.emit({
        ...base,
        eventIndex: timeline.events.indexOf(log.event!),
      });
    }
  }

  /**
   * Computes the view model for the overlay.
   *
   * This process involves:
   * 1. extracting Revisions and Events from the input `timelineHoverOverlay`.
   * 2. converting them into a unified `TimelineHoverOverlayLogItem` format.
   * 3. sorting all items by log index to ensure chronological order.
   * 4. determining the continuity of revision states across events (e.g., an event between two revisions inherits the state of the preceding revision).
   */
  viewModel = computed<TimelineHoverOverlayViewModel>(() => {
    const timelineHoverOverlay = this.timelineHoverOverlay();
    if (!timelineHoverOverlay) {
      return {
        logs: [],
        timeline: null,
        initialRevisionState: null,
      };
    }
    const highlights = this.highlights();
    const revisions = timelineHoverOverlay.revisions;
    const events = timelineHoverOverlay.events;
    const viewModel: TimelineHoverOverlayViewModel = {
      logs: [],
      timeline: timelineHoverOverlay.timeline,
      initialRevisionState: null,
    };

    // 1. Convert Revisions to LogItems
    for (let i = 0; i < revisions.length; i++) {
      const revision = revisions[i];
      const log = revision.log;
      const highlightType =
        highlights[log.logIndex] ?? TimelineChartItemHighlightType.None;
      viewModel.logs.push({
        uniqueID: `revision-${i}`,
        logIndex: log.logIndex,
        log: log,
        revision: revision,
        isRevision: true,
        revisionStateColor: RendererConvertUtil.hdrColorToCSSColor([
          revision.state.backgroundColor.r,
          revision.state.backgroundColor.g,
          revision.state.backgroundColor.b,
          revision.state.backgroundColor.a,
        ]),
        revisionStateLabel: revision.state.label,
        revisionStateIcon: revision.state.icon,
        revisionStateStyle: revision.state.style,
        logTypeColor: RendererConvertUtil.hdrColorToCSSColor([
          log.logType.backgroundColor.r,
          log.logType.backgroundColor.g,
          log.logType.backgroundColor.b,
          log.logType.backgroundColor.a,
        ]),
        logTypeLabel: log.logType.label,
        verbTypeColor: RendererConvertUtil.hdrColorToCSSColor([
          revision.verb.backgroundColor.r,
          revision.verb.backgroundColor.g,
          revision.verb.backgroundColor.b,
          revision.verb.backgroundColor.a,
        ]),
        verbTypeLabel: revision.verb.label,
        timeMs: revision.legacyChangedTimeMs,
        timeLabel: this.formatTimeLabel(revision.legacyChangedTimeMs),
        summary: log.summary,
        severityColor: RendererConvertUtil.hdrColorToCSSColor([
          log.severity.backgroundColor.r,
          log.severity.backgroundColor.g,
          log.severity.backgroundColor.b,
          log.severity.backgroundColor.a,
        ]),
        showSeverity: log.severity.id !== 0,
        severityLabel:
          log.severity.shortLabel ||
          (log.severity.label ? log.severity.label[0].toUpperCase() : ''),
        statusContinous: StatusContinousMode.StartAndEnd,
        highlightType: highlightType,
        lastRevisionLog: log,
        lastRevisionHightlightType: highlightType,
      });
    }

    // 2. Convert Events to LogItems
    for (let i = 0; i < events.length; i++) {
      const event = events[i];
      const log = event.log;
      const highlightType =
        highlights[log.logIndex] ?? TimelineChartItemHighlightType.None;
      viewModel.logs.push({
        uniqueID: `event-${i}`,
        log: log,
        event: event,
        logIndex: log.logIndex,
        isRevision: false,
        revisionStateColor: '',
        revisionStateLabel: '',
        revisionStateIcon: '',
        revisionStateStyle: RevisionStateStyle.NORMAL,
        logTypeColor: RendererConvertUtil.hdrColorToCSSColor([
          log.logType.backgroundColor.r,
          log.logType.backgroundColor.g,
          log.logType.backgroundColor.b,
          log.logType.backgroundColor.a,
        ]),
        logTypeLabel: log.logType.label,
        verbTypeColor: '',
        verbTypeLabel: '',
        timeMs: event.legacyTimestamp,
        timeLabel: this.formatTimeLabel(event.legacyTimestamp),
        summary: log.summary,
        severityColor: RendererConvertUtil.hdrColorToCSSColor([
          log.severity.backgroundColor.r,
          log.severity.backgroundColor.g,
          log.severity.backgroundColor.b,
          log.severity.backgroundColor.a,
        ]),
        severityLabel:
          log.severity.shortLabel ||
          (log.severity.label ? log.severity.label[0].toUpperCase() : ''),
        showSeverity: log.severity.id !== 0,
        statusContinous: StatusContinousMode.Middle,
        highlightType: highlightType,
        lastRevisionLog: null,
        lastRevisionHightlightType: TimelineChartItemHighlightType.None,
      });
    }

    // 3. Sort by User Log Index (chronological order)
    viewModel.logs.sort((a, b) => a.timeMs - b.timeMs);

    // 4. Calculate Continuity and Inherit States
    // Iterate through the sorted logs to:
    // - Propagate the revision state to subsequent events (so events show the state of the resource at that time).
    // - Determine the `statusContinous` mode (Start, Middle, End) for drawing connecting lines.
    let lastRevisionStateColor = 'transparent';
    let lastRevisionStateLabel = "status doesn't exist";
    let lastRevisionStateIcon = '';
    let lastRevisionStateStyle = RevisionStateStyle.NORMAL;
    let lastRevisionLog = null;
    let lastRevisionHightlightType = TimelineChartItemHighlightType.None;

    if (timelineHoverOverlay.initialRevision) {
      const rev = timelineHoverOverlay.initialRevision;
      lastRevisionStateColor = RendererConvertUtil.hdrColorToCSSColor([
        rev.state.backgroundColor.r,
        rev.state.backgroundColor.g,
        rev.state.backgroundColor.b,
        rev.state.backgroundColor.a,
      ]);
      lastRevisionStateLabel = rev.state.label;
      lastRevisionStateIcon = rev.state.icon;
      lastRevisionStateStyle = rev.state.style;
      let continousMode = StatusContinousMode.Middle;
      if (viewModel.logs.length > 0 && viewModel.logs[0].isRevision) {
        continousMode = StatusContinousMode.End;
      }
      viewModel.initialRevisionState = {
        revisionStateStyle: lastRevisionStateStyle,
        revisionStateColor: lastRevisionStateColor,
        statusContinous: continousMode,
      };
    }

    for (let i = 0; i < viewModel.logs.length; i++) {
      const log = viewModel.logs[i];
      if (log.isRevision) {
        lastRevisionStateColor = log.revisionStateColor;
        lastRevisionStateLabel = log.revisionStateLabel;
        lastRevisionStateIcon = log.revisionStateIcon;
        lastRevisionStateStyle = log.revisionStateStyle;
        lastRevisionLog = log.log;
        lastRevisionHightlightType = log.highlightType;
        if (i < viewModel.logs.length - 1) {
          const nextLog = viewModel.logs[i + 1];
          if (!nextLog.isRevision) {
            log.statusContinous = StatusContinousMode.Start;
          }
        }
      } else {
        log.revisionStateColor = lastRevisionStateColor;
        log.revisionStateLabel = lastRevisionStateLabel;
        log.revisionStateIcon = lastRevisionStateIcon;
        log.revisionStateStyle = lastRevisionStateStyle;
        log.lastRevisionLog = lastRevisionLog;
        log.lastRevisionHightlightType = lastRevisionHightlightType;
        if (
          i === viewModel.logs.length - 1 ||
          viewModel.logs[i + 1].isRevision
        ) {
          log.statusContinous = StatusContinousMode.End;
        }
      }
      if (log.severityLabel === 'U') {
        log.severityLabel = '';
      }
    }
    return viewModel;
  });

  private formatTimeLabel(timeInMs: number): string {
    const timezoneShiftHours = this.timezoneShiftHours();
    const d = new Date(timeInMs + timezoneShiftHours * 60 * 60 * 1000);
    const h = d.getUTCHours().toString().padStart(2, '0');
    const m = d.getUTCMinutes().toString().padStart(2, '0');
    const s = d.getUTCSeconds().toString().padStart(2, '0');
    const S = d.getUTCMilliseconds().toString().padStart(3, '0');
    return `${h}:${m}:${s}.${S}`;
  }
}
