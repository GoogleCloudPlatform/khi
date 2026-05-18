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
import { TimelineChartItemHighlightType } from './interaction-model';
import { RevisionStateStyle } from 'src/app/store/domain/style';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { Timeline, Revision, Event } from 'src/app/store/domain/timeline';
import { Log } from 'src/app/store/domain/log';

const meta: Meta<TimelineHoverOverlayComponent> = {
  title: 'Timeline/Hover Overlay',
  component: TimelineHoverOverlayComponent,
  tags: ['autodocs'],
};

const mockSeverityInfo = {
  id: 1,
  shortLabel: 'I',
  label: 'Info',
  backgroundColor: { r: 0.2, g: 0.6, b: 0.2, a: 1.0 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1.0 },
};

const mockSeverityWarning = {
  id: 2,
  shortLabel: 'W',
  label: 'Warning',
  backgroundColor: { r: 0.8, g: 0.5, b: 0.1, a: 1.0 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1.0 },
};

const mockSeverityError = {
  id: 3,
  shortLabel: 'E',
  label: 'Error',
  backgroundColor: { r: 0.8, g: 0.2, b: 0.2, a: 1.0 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1.0 },
};

const mockLogTypeAudit = {
  id: 1,
  label: 'Audit',
  description: 'Audit log',
  backgroundColor: { r: 0.1, g: 0.3, b: 0.6, a: 1.0 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1.0 },
};

const mockLogTypeAutoscaler = {
  id: 2,
  label: 'Autoscaler',
  description: 'Autoscaler events',
  backgroundColor: { r: 0.4, g: 0.2, b: 0.5, a: 1.0 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1.0 },
};

const mockStateNormal = {
  id: 1,
  label: 'Normal',
  icon: 'check_circle',
  style: RevisionStateStyle.NORMAL,
  backgroundColor: { r: 0.1, g: 0.5, b: 0.1, a: 1.0 },
};

const mockStatePartial = {
  id: 2,
  label: 'Partial Info',
  icon: 'info',
  style: RevisionStateStyle.PARTIAL_INFO,
  backgroundColor: { r: 0.8, g: 0.5, b: 0.1, a: 1.0 },
};

const mockStateDeleted = {
  id: 3,
  label: 'Deleted',
  icon: 'delete',
  style: RevisionStateStyle.DELETED,
  backgroundColor: { r: 0.5, g: 0.5, b: 0.5, a: 1.0 },
};

const mockVerbUpdate = {
  id: 1,
  label: 'Update',
  backgroundColor: { r: 0.2, g: 0.4, b: 0.8, a: 1.0 },
};

function createMockLog(
  logIndex: number,
  summary: string,
  timeMs: number,
  logType: unknown,
  severity: unknown,
): ReadonlyDomainElement<Log> {
  return {
    id: logIndex,
    logIndex,
    timestamp: BigInt(timeMs) * 1000000n,
    legacyTimestampMs: timeMs,
    summary,
    logType,
    severity,
  } as unknown as ReadonlyDomainElement<Log>;
}

function createMockRevision(
  log: ReadonlyDomainElement<Log>,
  startAt: number,
  state: unknown,
): ReadonlyDomainElement<Revision> {
  return {
    id: log.logIndex,
    timelineId: 1,
    index: 0,
    legacyChangedTimeMs: startAt,
    changedTime: BigInt(startAt) * 1000000n,
    log,
    state,
    verb: mockVerbUpdate,
  } as unknown as ReadonlyDomainElement<Revision>;
}

function createMockEvent(
  log: ReadonlyDomainElement<Log>,
  ts: number,
): ReadonlyDomainElement<Event> {
  return {
    id: log.logIndex,
    timelineId: 1,
    log,
    legacyTimestamp: ts,
  } as unknown as ReadonlyDomainElement<Event>;
}

function createHoverOverlayDemoData(): TimelineHoverOverlay {
  const baseTime = new Date(2025, 0, 1, 12, 0, 0, 0).getTime();
  const log0 = createMockLog(
    0,
    'foo',
    baseTime,
    mockLogTypeAudit,
    mockSeverityInfo,
  );
  const log1 = createMockLog(
    1,
    'bar',
    baseTime + 100,
    mockLogTypeAutoscaler,
    mockSeverityWarning,
  );
  const log2 = createMockLog(
    2,
    'baz',
    baseTime + 200,
    mockLogTypeAudit,
    mockSeverityError,
  );

  const revisions = [
    createMockRevision(log0, baseTime, mockStateNormal),
    createMockRevision(log2, baseTime + 200, mockStatePartial),
  ];

  const events = [createMockEvent(log1, baseTime + 100)];

  const timeline: ReadonlyDomainElement<Timeline> = {
    id: 1,
    name: 'mock-resource',
    type: {
      id: 1,
      label: 'name',
      height: 1.0,
      backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
    },
    revisions,
    events,
  } as unknown as ReadonlyDomainElement<Timeline>;

  return {
    timeline,
    revisions,
    events,
    initialRevision: revisions[0],
  };
}

function createHoverOverlayDemoDataWithEventFirst(): TimelineHoverOverlay {
  const baseTime = new Date(2025, 0, 1, 12, 0, 0, 0).getTime();
  const logBackground = createMockLog(
    -1,
    'Background Revision',
    baseTime - 1000,
    mockLogTypeAudit,
    mockSeverityInfo,
  );
  const log0 = createMockLog(
    0,
    'Event before any revision in the viewport',
    baseTime + 100,
    mockLogTypeAutoscaler,
    mockSeverityWarning,
  );
  const log1 = createMockLog(
    1,
    'Another event',
    baseTime + 150,
    mockLogTypeAudit,
    mockSeverityInfo,
  );
  const log2 = createMockLog(
    2,
    'baz',
    baseTime + 200,
    mockLogTypeAudit,
    mockSeverityError,
  );

  const revisions = [
    createMockRevision(logBackground, baseTime - 1000, mockStateNormal),
    createMockRevision(log2, baseTime + 200, mockStateDeleted),
  ];

  const events = [
    createMockEvent(log0, baseTime + 100),
    createMockEvent(log1, baseTime + 150),
  ];

  const timeline: ReadonlyDomainElement<Timeline> = {
    id: 1,
    name: 'mock-resource-event-first',
    type: {
      id: 1,
      label: 'name',
      height: 1.0,
      backgroundColor: { r: 0, g: 0, b: 0, a: 0 },
    },
    revisions,
    events,
  } as unknown as ReadonlyDomainElement<Timeline>;

  return {
    timeline,
    revisions: [revisions[1]], // background revision is out of range
    events,
    initialRevision: revisions[0],
  };
}

export default meta;
type Story = StoryObj<TimelineHoverOverlayComponent>;

export const Default: Story = {
  args: {
    timelineHoverOverlay: createHoverOverlayDemoData(),
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

export const FirstItemIsEvent: Story = {
  args: {
    timelineHoverOverlay: createHoverOverlayDemoDataWithEventFirst(),
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

export const HoveredOnEventAndSelectedOnRevision: Story = {
  args: {
    timelineHoverOverlay: createHoverOverlayDemoData(),
    highlights: {
      0: TimelineChartItemHighlightType.Selected, // Revision (foo)
      1: TimelineChartItemHighlightType.Hovered, // Event (bar)
    },
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

export const HoveredOnRevisionAndSelectedOnEvent: Story = {
  args: {
    timelineHoverOverlay: createHoverOverlayDemoData(),
    highlights: {
      0: TimelineChartItemHighlightType.Hovered, // Revision (foo)
      1: TimelineChartItemHighlightType.Selected, // Event (bar)
    },
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
