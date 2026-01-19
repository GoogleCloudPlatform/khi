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

import { Meta, StoryObj } from '@storybook/angular';
import {
  TimelineHoverOverlay,
  TimelineHoverOverlayComponent,
} from './timeline-hover-overlay.component';
import { LogEntry } from 'src/app/store/log';
import {
  LogType,
  ParentRelationship,
  RevisionState,
  RevisionVerb,
  Severity,
} from 'src/app/generated';
import { ReferenceType } from 'src/app/common/loader/interface';
import { ResourceRevision } from 'src/app/store/revision';
import { ResourceEvent } from 'src/app/store/event';
import { ResourceTimeline } from 'src/app/store/timeline';

const meta: Meta<TimelineHoverOverlayComponent> = {
  title: 'Timeline/Hover Overlay',
  component: TimelineHoverOverlayComponent,
  tags: ['autodocs'],
};

function createLogEntry(
  logIndex: number,
  summary: string,
  timeMS: number,
  logType: LogType,
  severity: Severity,
): LogEntry {
  return new LogEntry(
    logIndex,
    '',
    logType,
    severity,
    timeMS,
    summary,
    { type: ReferenceType.NullReference },
    [],
  );
}

function createRevision(
  log: LogEntry,
  startAt: number,
  stateRaw: number,
): ResourceRevision {
  return new ResourceRevision(
    startAt,
    startAt + 100,
    stateRaw,
    RevisionVerb.RevisionVerbUpdate,
    '',
    '',
    false,
    false,
    log.logIndex,
  );
}

function createEvent(log: LogEntry, ts: number): ResourceEvent {
  return new ResourceEvent(log.logIndex, ts, log.logType, log.severity);
}

function createHoverOverlayDemoData(): {
  overlay: TimelineHoverOverlay;
  logs: LogEntry[];
} {
  const result = {} as { overlay: TimelineHoverOverlay; logs: LogEntry[] };
  const baseTime = new Date(2025, 0, 1, 12, 0, 0, 0).getTime();
  result.logs = [
    createLogEntry(
      0,
      'foo',
      baseTime,
      LogType.LogTypeAudit,
      Severity.SeverityInfo,
    ),
    createLogEntry(
      1,
      'bar',
      baseTime + 100,
      LogType.LogTypeAutoscaler,
      Severity.SeverityWarning,
    ),
    createLogEntry(
      2,
      'baz',
      baseTime + 200,
      LogType.LogTypeAudit,
      Severity.SeverityError,
    ),
  ];
  const timeline = new ResourceTimeline(
    '',
    '',
    [
      createRevision(
        result.logs[0],
        baseTime,
        RevisionState.RevisionStateInferred,
      ),
      createRevision(
        result.logs[2],
        baseTime + 200,
        RevisionState.RevisionStateContainerStatusNotAvailable,
      ),
    ],
    [createEvent(result.logs[1], baseTime + 100)],
    ParentRelationship.RelationshipChild,
  );
  result.overlay = {
    timeline: timeline,
    revisions: timeline.revisions,
    events: timeline.events,
  };
  return result;
}

export default meta;
type Story = StoryObj<TimelineHoverOverlayComponent>;

export const Default: Story = {
  args: {
    timelineHoverOverlay: createHoverOverlayDemoData().overlay,
    logs: createHoverOverlayDemoData().logs,
  },
  argTypes: {
    hoverOnElement: {
      action: 'hoverOnElement',
    },
    clickOnElement: {
      action: 'clickOnElement',
    },
  },
};
