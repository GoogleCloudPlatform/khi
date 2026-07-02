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
} from 'src/app/timeline/components/timeline-frame.component';
import { Component, computed, effect, resource, signal } from '@angular/core';
import { Timeline } from 'src/app/store/domain/timeline';
import { TimelineChartMouseEvent } from 'src/app/timeline/components/timeline-chart.component';
import { TimelineHoverOverlay } from 'src/app/timeline/components/timeline-hover-overlay.component';
import {
  TimelineHighlight,
  TimelineChartItemHighlight,
  TimelineHighlightType,
  TimelineChartItemHighlightType,
} from 'src/app/timeline/components/interaction-model';
import {
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model';
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';

function msToNs(ms: number): bigint {
  return BigInt(Math.floor(ms)) * 1000000n;
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
  template: `
    @if (viewModel().ready) {
      <khi-timeline-frame
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
        [chartStyle]="viewModel().chartStyle!"
        [rulerStyle]="viewModel().rulerStyle!"
        [styleStore]="viewModel().styleStore!"
        (hoverOnTimeline)="hoverOnTimeline($event)"
        (clickOnTimeline)="clickOnTimeline($event)"
        (hoverOnTimelineItem)="hoverOnTimelineChartItem($event)"
        (clickOnTimelineItem)="clickOnTimelineChartItem($event)"
      ></khi-timeline-frame>
    }
  `,
  imports: [TimelineFrameComponent],
})
class TimelineFrameStoriesComponent {
  readonly khiInspectionData = resource({
    loader: async () => {
      return await createMockInspectionData();
    },
  });

  constructor() {
    effect(() => {
      const inspectionData = this.khiInspectionData.value();
      if (inspectionData) {
        const minTimeMs = inspectionData.metadata?.header
          ? inspectionData.metadata.header.startTimeUnixSeconds * 1000
          : 0;
        const maxTimeMs = inspectionData.metadata?.header
          ? inspectionData.metadata.header.endTimeUnixSeconds * 1000
          : 1;
        this.pixelsPerMs.set(3000 / (maxTimeMs - minTimeMs));
        this.viewportLeftTimeMS.set(minTimeMs);
      }
    });
  }

  viewModel = computed(() => {
    const data = this.khiInspectionData.value();
    if (!data) {
      return {
        ready: false,
        timelines: [],
        logs: [],
        filteredLogs: [],
        minLogTime: 0,
        maxLogTime: 0,
        chartStyle: undefined,
        rulerStyle: undefined,
        styleStore: undefined,
      };
    }
    const minTimeMs = data.metadata?.header
      ? data.metadata.header.startTimeUnixSeconds * 1000
      : 0;
    const maxTimeMs = data.metadata?.header
      ? data.metadata.header.endTimeUnixSeconds * 1000
      : 0;
    const logs = Array.from(data.logStore.logs());
    return {
      ready: true,
      timelines: data.timelineStore.timelines,
      logs,
      filteredLogs: logs,
      minLogTime: minTimeMs,
      maxLogTime: maxTimeMs,
      chartStyle: generateDefaultChartStyle(),
      rulerStyle: generateDefaultRulerStyle(data.styleStore),
      styleStore: data.styleStore,
    };
  });

  viewportLeftTimeMS = signal(0);

  pixelsPerMs = signal(1);

  timelineHighlights = signal<TimelineHighlight>({});

  timelineChartItemHighlights = signal<TimelineChartItemHighlight>({});

  timeCursorMS = signal(0);

  timelineHoverRequest = signal<TimelineHoverOverlayRequest | null>(null);

  hoverOnTimeline(t: Timeline) {
    const highlights = this.timelineHighlights();
    if (highlights[t.id] === TimelineHighlightType.Selected) {
      return;
    }
    this.timelineHighlights.set({
      ...highlights,
      [t.id]: TimelineHighlightType.Hovered,
    });
  }

  clickOnTimeline(t: Timeline) {
    const highlights = this.timelineHighlights();
    this.timelineHighlights.set({
      ...highlights,
      [t.id]: TimelineHighlightType.Selected,
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
    if (timelineHighlights[e.timeline.id] === TimelineHighlightType.Selected) {
      return;
    }
    this.timelineHighlights.set({
      ...filterTimelineHighlight(
        timelineHighlights,
        (h) => h !== TimelineHighlightType.Hovered,
      ),
      [e.timeline.id]: TimelineHighlightType.Hovered,
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
      this.timeCursorMS.set(revision.legacyChangedTimeMs);

      const events = e.timeline.lookupEventsInRangeNs(
        msToNs(e.timeMS - timeRange),
        msToNs(e.timeMS + timeRange),
      );
      const revisions = e.timeline.lookupRevisionsInRangeNs(
        msToNs(e.timeMS - timeRange),
        msToNs(e.timeMS + timeRange),
      );
      this.timelineHoverRequest.set({
        timelineId: e.timeline.id,
        timeMs: e.timeMS,
        overlay: {
          timeline: e.timeline,
          revisions: revisions,
          events: events,
          initialRevision: null,
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
      this.timeCursorMS.set(event.legacyTimestamp);
      const events = e.timeline.lookupEventsInRangeNs(
        msToNs(event.legacyTimestamp - timeRange),
        msToNs(event.legacyTimestamp + timeRange),
      );
      const revisions = e.timeline.lookupRevisionsInRangeNs(
        msToNs(event.legacyTimestamp - timeRange),
        msToNs(event.legacyTimestamp + timeRange),
      );
      this.timelineHoverRequest.set({
        timelineId: e.timeline.id,
        timeMs: e.timeMS,
        overlay: {
          timeline: e.timeline,
          revisions: revisions,
          events: events,
          initialRevision: null,
        } as TimelineHoverOverlay,
      });
    }
    this.timelineHighlights.set({
      [e.timeline.id]: TimelineHighlightType.Selected,
    });
  }
}

const meta: Meta<TimelineFrameStoriesComponent> = {
  title: 'Timeline/Frame',
  component: TimelineFrameStoriesComponent,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
  argTypes: {
    viewModel: {
      control: false,
    },
    timelineHighlights: {
      control: false,
    },
    timelineChartItemHighlights: {
      control: false,
    },
    timelineHoverRequest: {
      control: false,
    },
  },
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
