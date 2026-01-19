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

import { componentWrapperDecorator, Meta, StoryObj } from '@storybook/angular';
import {
  TimelineFrameComponent,
  TimelineHoverOverlayRequest,
} from './timeline-frame.component';
import {
  Component,
  computed,
  effect,
  inject,
  input,
  resource,
  signal,
} from '@angular/core';
import { ResourceTimeline } from 'src/app/store/timeline';
import {
  LogType,
  ParentRelationship,
  ParentRelationshipMetadata,
  RevisionState,
  RevisionVerb,
  Severity,
} from 'src/app/generated';
import { ResourceRevision } from 'src/app/store/revision';
import { ResourceEvent } from 'src/app/store/event';
import { TimelineChartMouseEvent } from './timeline-chart.component';
import { LogEntry } from 'src/app/store/log';
import { ReferenceType } from 'src/app/common/loader/interface';
import { TimelineHoverOverlay } from './timeline-hover-overlay.component';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { ProgressDialogService } from 'src/app/services/progress/progress-dialog.service';
import { BACKEND_API } from 'src/app/services/api/backend-api-interface';
import { BackendAPIImpl } from 'src/app/services/api/backend-api.service';
import { InspectionDataStoreService } from 'src/app/services/inspection-data-store.service';
import { filter, firstValueFrom } from 'rxjs';
import {
  TimelineHighlight,
  TimelineChartItemHighlight,
  TimelineHighlightType,
  TimelineChartItemHighlightType,
} from './interaction-model';

let i = 0;
function generateRevision(
  start: number,
  end: number,
  state: RevisionState,
): ResourceRevision {
  return new ResourceRevision(
    start,
    end,
    state,
    RevisionVerb.RevisionVerbCreate,
    '',
    '',
    false,
    false,
    i++,
  );
}

function generateEvent(
  ts: number,
  logType: LogType,
  severity: Severity,
): ResourceEvent {
  return new ResourceEvent(i++, ts, logType, severity);
}

let logIndex = 0;
function generateLog(ts: number, severity: Severity): LogEntry {
  return new LogEntry(
    logIndex++,
    '',
    LogType.LogTypeAudit,
    severity,
    ts,
    '',
    { type: ReferenceType.NullReference },
    [],
  );
}

const testAllLogs: LogEntry[] = [];
const testFilteredLogs: LogEntry[] = [];

const beginTime = new Date(2025, 0, 1, 0, 0, 0).getTime();
const endTime = new Date(2025, 0, 2, 0, 0, 0).getTime();
const severities = [
  Severity.SeverityFatal,
  Severity.SeverityError,
  Severity.SeverityWarning,
  Severity.SeverityInfo,
  Severity.SeverityUnknown,
];
for (let i = 0; i < 10000; i++) {
  const rnd = Math.random();
  const rnd2 = Math.random();
  const rnd3 = Math.random();
  const l = generateLog(
    beginTime + (endTime - beginTime) * rnd2,
    severities[Math.floor(rnd * severities.length)],
  );
  testAllLogs.push(l);
  if (rnd3 < 0.5) {
    testFilteredLogs.push(l);
  }
}

function filterChartItemHighlight(
  highlights: TimelineChartItemHighlight,
  predicate: (highlight: TimelineChartItemHighlightType) => boolean,
) {
  return Object.fromEntries(
    Object.entries(highlights).filter(([, value]) => predicate(value)),
  );
}

function filterTimelineHighlight(
  highlights: TimelineHighlight,
  predicate: (highlight: TimelineHighlightType) => boolean,
) {
  return Object.fromEntries(
    Object.entries(highlights).filter(([, value]) => predicate(value)),
  );
}

@Component({
  template: `<khi-timeline-frame
    [timelines]="viewModel().timelines"
    [minQueryLogTimeMS]="viewModel().minLogTime"
    [maxQueryLogTimeMS]="viewModel().maxLogTime"
    [viewportLeftTimeMS]="viewportLeftTimeMS()"
    [pixelsPerMs]="pixelsPerMs()"
    [timelineHighlights]="timelineHighlights()"
    [timelineChartItemHighlights]="timelineChartItemHighlights()"
    [cursorTimeMS]="timeCursorMS()"
    [timezoneShiftHours]="9"
    [allLogsWithoutFilter]="viewModel().logs"
    [filteredLogs]="viewModel().filteredLogs"
    [timelineHoverOverlayRequest]="timelineHoverRequest()"
    (hoverOnTimeline)="hoverOnTimeline($event)"
    (clickOnTimeline)="clickOnTimeline($event)"
    (hoverOnTimelineItem)="hoverOnTimelineChartItem($event)"
    (clickOnTimelineItem)="clickOnTimelineChartItem($event)"
  ></khi-timeline-frame>`,
  imports: [TimelineFrameComponent],
  providers: [
    ...ProgressDialogService.providers(),
    InspectionDataLoaderService,
    InspectionDataStoreService,
    { provide: BACKEND_API, useClass: BackendAPIImpl },
  ],
})
class TimelineFrameStoriesComponent {
  private readonly dataLoader = inject(InspectionDataLoaderService);

  private readonly dataStore = inject(InspectionDataStoreService);

  readonly khiFilePath = input('./assets/demo/khi-demo.khi');

  readonly khiInspectionData = resource({
    params: () => ({ filePath: this.khiFilePath() }),
    loader: async ({ params }) => {
      const response = await fetch(params.filePath);
      this.dataLoader.loadInspectionDataDirect(await response.arrayBuffer());
      const v = await firstValueFrom(
        this.dataStore.inspectionData.pipe(filter((v) => v !== null)),
      );
      return v;
    },
  });

  constructor() {
    effect(() => {
      const inspectionData = this.khiInspectionData.value();
      if (inspectionData) {
        this.pixelsPerMs.set(
          3000 / (inspectionData.range.end - inspectionData.range.begin),
        );
        this.viewportLeftTimeMS.set(inspectionData.range.begin);
      }
    });
  }

  viewModel = computed(() => {
    if (!this.khiInspectionData.hasValue()) {
      return this.generateDemoTimelineViewModel();
    }
    const timeline = this.khiInspectionData.value();
    if (!timeline) {
      return {
        timelines: [],
        logs: [],
        filteredLogs: [],
        minLogTime: 0,
        maxLogTime: 0,
      };
    }
    return {
      timelines: timeline.timelines,
      logs: timeline.logs,
      filteredLogs: timeline.logs,
      minLogTime: timeline.range.begin,
      maxLogTime: timeline.range.end,
    };
  });

  viewportLeftTimeMS = signal(new Date(2025, 0, 1, 0, 0, 0).getTime());

  pixelsPerMs = signal(300 / 60 / 60);

  timelineHighlights = signal<TimelineHighlight>({});

  timelineChartItemHighlights = signal<TimelineChartItemHighlight>({});

  timeCursorMS = signal(0);

  timelineHoverRequest = signal<TimelineHoverOverlayRequest | null>(null);

  hoverOnTimeline(t: ResourceTimeline) {
    const highlights = this.timelineHighlights();
    if (highlights[t.timelineId] === TimelineHighlightType.Selected) {
      return;
    }
    this.timelineHighlights.set({
      ...highlights,
      [t.timelineId]: TimelineHighlightType.Hovered,
    });
  }

  clickOnTimeline(t: ResourceTimeline) {
    const highlights = this.timelineHighlights();
    this.timelineHighlights.set({
      ...highlights,
      [t.timelineId]: TimelineHighlightType.Selected,
    });
  }

  hoverOnTimelineChartItem(e: TimelineChartMouseEvent) {
    const timelineHighlights = this.timelineHighlights();
    const timelineChartItemHighlights = this.timelineChartItemHighlights();
    if (e.timeline === null) {
      this.timelineChartItemHighlights.set({});
      this.timelineHighlights.set({
        ...filterTimelineHighlight(
          timelineHighlights,
          (h) => h !== TimelineHighlightType.Hovered,
        ),
      });
      return;
    }

    if (e.revisionIndex !== undefined) {
      const revision = e.timeline.revisions[e.revisionIndex];
      if (
        timelineChartItemHighlights[revision.logIndex] ===
        TimelineChartItemHighlightType.Selected
      ) {
        return;
      }
      this.timelineChartItemHighlights.set({
        ...filterChartItemHighlight(
          this.timelineChartItemHighlights(),
          (h) => h !== TimelineChartItemHighlightType.Hovered,
        ),
        [revision.logIndex]: TimelineChartItemHighlightType.Hovered,
      });
    } else if (e.eventIndex !== undefined) {
      const event = e.timeline.events[e.eventIndex];
      if (
        timelineChartItemHighlights[event.logIndex] ===
        TimelineChartItemHighlightType.Selected
      ) {
        return;
      }
      this.timelineChartItemHighlights.set({
        ...filterChartItemHighlight(
          this.timelineChartItemHighlights(),
          (h) => h !== TimelineChartItemHighlightType.Hovered,
        ),
        [event.logIndex]: TimelineChartItemHighlightType.Hovered,
      });
    }
    if (
      timelineHighlights[e.timeline.timelineId] ===
      TimelineHighlightType.Selected
    ) {
      return;
    }
    this.timelineHighlights.set({
      ...filterTimelineHighlight(
        timelineHighlights,
        (h) => h !== TimelineHighlightType.Hovered,
      ),
      [e.timeline!.timelineId]: TimelineHighlightType.Hovered,
    });
  }
  clickOnTimelineChartItem(e: TimelineChartMouseEvent) {
    if (e.timeline === null) {
      this.timelineChartItemHighlights.set({});
      this.timelineHighlights.set({});
      return;
    }
    const pixelsPerMs = this.pixelsPerMs();
    const timeRange = 300 / pixelsPerMs; // select 30 pixel around
    if (e.revisionIndex !== undefined) {
      const revision = e.timeline.revisions[e.revisionIndex];
      this.timelineChartItemHighlights.set({
        ...filterChartItemHighlight(
          this.timelineChartItemHighlights(),
          (h) => h !== TimelineChartItemHighlightType.Selected,
        ),
        [revision.logIndex]: TimelineChartItemHighlightType.Selected,
      });
      this.timeCursorMS.set(revision.startAt);

      const events = e.timeline.queryEventsInRange(
        e.timeMS - timeRange,
        e.timeMS + timeRange,
      );
      const revisions = e.timeline.queryRevisionsInRange(
        e.timeMS - timeRange,
        e.timeMS + timeRange,
      );
      this.timelineHoverRequest.set({
        timelineId: e.timeline.timelineId,
        timeMs: e.timeMS,
        overlay: {
          revisions: revisions,
          events: events,
        } as TimelineHoverOverlay,
      });
    } else if (e.eventIndex !== undefined) {
      const event = e.timeline.events[e.eventIndex];
      this.timelineChartItemHighlights.set({
        ...filterChartItemHighlight(
          this.timelineChartItemHighlights(),
          (h) => h !== TimelineChartItemHighlightType.Selected,
        ),
        [event.logIndex]: TimelineChartItemHighlightType.Selected,
      });
      this.timeCursorMS.set(event.ts);
      const events = e.timeline.queryEventsInRange(
        event.ts - timeRange,
        event.ts + timeRange,
      );
      const revisions = e.timeline.queryRevisionsInRange(
        event.ts - timeRange,
        event.ts + timeRange,
      );
      this.timelineHoverRequest.set({
        timelineId: e.timeline.timelineId,
        timeMs: e.timeMS,
        overlay: {
          timeline: e.timeline,
          revisions: revisions,
          events: events,
        } as TimelineHoverOverlay,
      });
    }
    this.timelineHighlights.set({
      [e.timeline.timelineId]: TimelineHighlightType.Selected,
    });
  }

  generateDemoTimelineViewModel(): {
    timelines: ResourceTimeline[];
    minLogTime: number;
    maxLogTime: number;
    logs: LogEntry[];
  } {
    const beginTime = new Date(2025, 0, 1, 0, 0, 0).getTime();
    const vm: {
      timelines: ResourceTimeline[];
      minLogTime: number;
      maxLogTime: number;
      logs: LogEntry[];
      filteredLogs: LogEntry[];
    } = {
      timelines: new Array<ResourceTimeline>(0),
      minLogTime: beginTime,
      maxLogTime: beginTime + 1000 * 60 * 60 * 24,
      logs: testAllLogs,
      filteredLogs: testFilteredLogs,
    };
    const duration = 1000 * 60 * 60 * 1;
    for (let kind = 0; kind < 20; kind++) {
      vm.timelines.push(
        new ResourceTimeline(
          `t-${kind}`,
          `core/v1#kind-${kind}`,
          [],
          [],
          ParentRelationship.RelationshipChild,
        ),
      );
      for (let namespace = 0; namespace < 5; namespace++) {
        vm.timelines.push(
          new ResourceTimeline(
            `t-${kind}-n-${namespace}`,
            `core/v1#kind-${kind}#namespace-${namespace}`,
            [],
            [],
            ParentRelationship.RelationshipChild,
          ),
        );
        for (let name = 0; name < 5; name++) {
          const revisions: ResourceRevision[] = [];
          const revisionStates: RevisionState[] = [
            13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24,
          ];
          const logSeverities: Severity[] = [
            Severity.SeverityInfo,
            Severity.SeverityWarning,
            Severity.SeverityError,
            Severity.SeverityFatal,
          ];
          const events: ResourceEvent[] = [];
          for (let revision = 0; revision < 20; revision++) {
            revisions.push(
              generateRevision(
                beginTime + revision * duration,
                beginTime + (revision + 1) * duration,
                revisionStates[revision % revisionStates.length],
              ),
            );
          }
          for (let event = 0; event < 20; event++) {
            events.push(
              generateEvent(
                beginTime + event * duration,
                LogType.LogTypeAudit,
                logSeverities[event % logSeverities.length],
              ),
            );
          }
          vm.timelines.push(
            new ResourceTimeline(
              `t-${kind}-n-${namespace}-n-${name}`,
              `core/v1#kind-${kind}#namespace-${namespace}#name-${name}`,
              revisions,
              events,
              ParentRelationship.RelationshipChild,
            ),
          );
          for (let subresource = 0; subresource < 20; subresource++) {
            const revisions: ResourceRevision[] = [];
            const revisionStates: RevisionState[] = [
              RevisionState.RevisionStateContainerStatusNotAvailable,
              RevisionState.RevisionStateContainerRunningNonReady,
              RevisionState.RevisionStateContainerRunningReady,
              RevisionState.RevisionStateContainerTerminatedWithSuccess,
            ];
            const logSeverities: Severity[] = [
              Severity.SeverityInfo,
              Severity.SeverityWarning,
              Severity.SeverityError,
              Severity.SeverityFatal,
            ];
            const events: ResourceEvent[] = [];
            for (let revision = 0; revision < 20; revision++) {
              revisions.push(
                generateRevision(
                  beginTime + revision * duration,
                  beginTime + (revision + 1) * duration,
                  revisionStates[revision % revisionStates.length],
                ),
              );
            }
            for (let event = 0; event < 20; event++) {
              events.push(
                generateEvent(
                  beginTime + event * duration,
                  LogType.LogTypeAudit,
                  logSeverities[event % logSeverities.length],
                ),
              );
            }
            vm.timelines.push(
              new ResourceTimeline(
                `t-${kind}-n-${namespace}-n-${name}-sr-${subresource}`,
                `core/v1#kind-${kind}#namespace-${namespace}#name-${name}#subresource-${subresource}`,
                revisions,
                events,
                subresource % ParentRelationshipMetadata.length,
              ),
            );
          }
        }
      }
    }
    return vm;
  }
}

const meta: Meta<TimelineFrameStoriesComponent> = {
  title: 'Timeline/Frame',
  component: TimelineFrameStoriesComponent,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
  argTypes: {},
  decorators: [
    componentWrapperDecorator(
      (story) => `
      <div style="height: 100vh; width: 100%">
        ${story}
      </div>
    `,
    ),
  ],
};

export default meta;
type Story = StoryObj<TimelineFrameComponent>;

export const Default: Story = {
  args: {},
  argTypes: {
    timezoneShiftHours: {
      control: 'number',
    },
  },
};
