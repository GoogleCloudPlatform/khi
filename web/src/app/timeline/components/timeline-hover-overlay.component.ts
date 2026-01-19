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
import {
  logTypeColors,
  LogTypeMetadata,
  logTypes,
  revisionStatecolors,
  RevisionStateMetadata,
  revisionStates,
  RevisionStateStyle,
  RevisionVerbMetadata,
  severities,
  Severity,
  severityColors,
  SeverityMetadata,
} from 'src/app/generated';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { ResourceEvent } from 'src/app/store/event';
import { LogEntry } from 'src/app/store/log';
import { ResourceRevision } from 'src/app/store/revision';
import { MatTooltip } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import {
  TimelineChartItemHighlight,
  TimelineChartItemHighlightType,
} from './interaction-model';
import { RendererConvertUtil } from './canvas/convertutil';
import { TimelineChartMouseEvent } from './timeline-chart.component';
import { ResourceTimeline } from 'src/app/store/timeline';

export interface TimelineHoverOverlay {
  timeline: ResourceTimeline;
  revisions: ResourceRevision[];
  events: ResourceEvent[];
}

enum StatusContinousMode {
  StartAndEnd,
  Start,
  Middle,
  End,
}

interface TimelineHoverOverlayLogItem {
  uniqueID: string;
  log: LogEntry;
  revision?: ResourceRevision;
  event?: ResourceEvent;
  logIndex: number;
  isRevision: boolean;
  revisionStateColor: string;
  revisionStateLabel: string;
  revisionStateIcon: string;
  revisionStateStyle: RevisionStateStyle;
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
}

interface TimelineHoverOverlayViewModel {
  timeline: ResourceTimeline | null;
  logs: TimelineHoverOverlayLogItem[];
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
  StatusContinousMode = StatusContinousMode;
  RevisionStateStyle = RevisionStateStyle;
  TimelineChartItemHighlightType = TimelineChartItemHighlightType;
  /**
   * The data for the overlay, including the focused timeline and associated revisions/events.
   */
  timelineHoverOverlay = input<TimelineHoverOverlay | null>(null);

  /**
   * Complete list of log entries to look up details.
   */
  logs = input<LogEntry[]>([]);

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
    timeline: ResourceTimeline,
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
      };
    }
    const logs = this.logs();
    const highlights = this.highlights();
    const revisions = timelineHoverOverlay.revisions;
    const events = timelineHoverOverlay.events;
    const viewModel: TimelineHoverOverlayViewModel = {
      logs: [],
      timeline: timelineHoverOverlay.timeline,
    };

    // 1. Convert Revisions to LogItems
    for (let i = 0; i < revisions.length; i++) {
      const revision = revisions[i];
      const revisionStateColor =
        revisionStatecolors[revisionStates[revision.stateRaw]];
      const log = logs[revision.logIndex];
      const logColor = logTypeColors[logTypes[log.logType]];
      const logTypeLabel = LogTypeMetadata[log.logType].label;
      const revisionStateMetadata = RevisionStateMetadata[revision.stateRaw];
      const revisionStateLabel = revisionStateMetadata.label;
      const revisionStateIcon = revisionStateMetadata.icon;
      const severityColor = severityColors[severities[log.severity]];
      const severityLabel =
        SeverityMetadata[log.severity].label[0].toUpperCase();
      const verbTypeColor =
        RevisionVerbMetadata[revision.lastMutationVerb].color;
      const verbTypeLabel =
        RevisionVerbMetadata[revision.lastMutationVerb].label;
      const highlightType =
        highlights[revision.logIndex] ?? TimelineChartItemHighlightType.None;
      viewModel.logs.push({
        uniqueID: `revision-${i}`,
        logIndex: revision.logIndex,
        log: log,
        revision: revision,
        isRevision: true,
        revisionStateColor:
          RendererConvertUtil.hdrColorToCSSColor(revisionStateColor),
        revisionStateLabel: revisionStateLabel,
        revisionStateIcon: revisionStateIcon,
        revisionStateStyle: revisionStateMetadata.style,
        logTypeColor: RendererConvertUtil.hdrColorToCSSColor(logColor),
        logTypeLabel: logTypeLabel,
        verbTypeColor: RendererConvertUtil.hdrColorToCSSColor(verbTypeColor),
        verbTypeLabel: verbTypeLabel,
        timeMs: revision.startAt,
        timeLabel: this.formatTimeLabel(revision.startAt),
        summary: log.summary,
        severityColor: RendererConvertUtil.hdrColorToCSSColor(severityColor),
        showSeverity: log.severity !== Severity.SeverityUnknown,
        severityLabel: severityLabel,
        statusContinous: StatusContinousMode.StartAndEnd,
        highlightType: highlightType,
      });
    }

    // 2. Convert Events to LogItems
    for (let i = 0; i < events.length; i++) {
      const event = events[i];
      const log = logs[event.logIndex];
      const logColor = logTypeColors[logTypes[log.logType]];
      const logTypeLabel = LogTypeMetadata[log.logType].label;
      const severityColor = severityColors[severities[log.severity]];
      const severityLabel =
        SeverityMetadata[log.severity].label[0].toUpperCase();
      const highlightType =
        highlights[event.logIndex] ?? TimelineChartItemHighlightType.None;
      viewModel.logs.push({
        uniqueID: `event-${i}`,
        log: log,
        event: event,
        logIndex: event.logIndex,
        isRevision: false,
        revisionStateColor: '',
        revisionStateLabel: '',
        revisionStateIcon: '',
        revisionStateStyle: RevisionStateStyle.Normal,
        logTypeColor: RendererConvertUtil.hdrColorToCSSColor(logColor),
        logTypeLabel: logTypeLabel,
        verbTypeColor: '',
        verbTypeLabel: '',
        timeMs: event.ts,
        timeLabel: this.formatTimeLabel(event.ts),
        summary: log.summary,
        severityColor: RendererConvertUtil.hdrColorToCSSColor(severityColor),
        severityLabel: severityLabel,
        showSeverity: log.severity !== Severity.SeverityUnknown,
        statusContinous: StatusContinousMode.Middle,
        highlightType: highlightType,
      });
    }

    // 3. Sort by User Log Index (chronological order)
    viewModel.logs.sort((a, b) => a.logIndex - b.logIndex);

    // 4. Calculate Continuity and Inherit States
    // Iterate through the sorted logs to:
    // - Propagate the revision state to subsequent events (so events show the state of the resource at that time).
    // - Determine the `statusContinous` mode (Start, Middle, End) for drawing connecting lines.
    let lastRevisionStateColor = 'transparent';
    let lastRevisionStateLabel = "status doesn't exist";
    let lastRevisionStateIcon = '';
    let lastRevisionStateStyle = RevisionStateStyle.Normal;
    for (let i = 0; i < viewModel.logs.length; i++) {
      const log = viewModel.logs[i];
      if (log.isRevision) {
        lastRevisionStateColor = log.revisionStateColor;
        lastRevisionStateLabel = log.revisionStateLabel;
        lastRevisionStateIcon = log.revisionStateIcon;
        lastRevisionStateStyle = log.revisionStateStyle;
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
