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
  TimelineChartStyle,
  TimelineRulerStyle,
} from 'src/app/timeline/components/style-model';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { createMockInspectionData } from 'src/app/store/mock/inspection-data.mock';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

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

interface TimelineFrameStoryViewModelNotReady {
  readonly ready: false;
}

interface TimelineFrameStoryViewModelReady {
  readonly ready: true;
  readonly timelines: readonly ReadonlyDomainElement<Timeline>[];
  readonly logs: ReadonlyDomainElement<Log>[];
  readonly filteredLogIds: IdBitset;
  readonly minLogTimeNs: bigint;
  readonly maxLogTimeNs: bigint;
  readonly chartStyle: TimelineChartStyle;
  readonly rulerStyle: TimelineRulerStyle;
  readonly styleStore: StyleStoreLike;
}

type TimelineFrameStoryViewModel =
  TimelineFrameStoryViewModelNotReady | TimelineFrameStoryViewModelReady;

@Component({
  template: `
    @let vm = viewModel();
    @if (vm.ready) {
      <khi-timeline-frame
        [timelines]="vm.timelines"
        [minQueryLogTimeNs]="vm.minLogTimeNs"
        [maxQueryLogTimeNs]="vm.maxLogTimeNs"
        [viewportLeftTimeNs]="viewportLeftTimeNs()"
        [pixelsPerNs]="pixelsPerNs()"
        [timelineHighlights]="timelineHighlights()"
        [timelineChartItemHighlights]="timelineChartItemHighlights()"
        [cursorTimeNs]="timeCursorNs()"
        [secondaryCursorTimeNs]="secondaryTimeCursorNs()"
        [timezoneShiftHours]="9"
        [allLogs]="vm.logs"
        [filteredLogIds]="vm.filteredLogIds"
        [timelineHoverOverlayRequest]="timelineHoverRequest()"
        [chartStyle]="vm.chartStyle"
        [rulerStyle]="vm.rulerStyle"
        [styleStore]="vm.styleStore"
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
        const minTimeNs = inspectionData.metadata?.header
          ? BigInt(inspectionData.metadata.header.startTimeUnixSeconds) *
            1_000_000_000n
          : 0n;
        const maxTimeNs = inspectionData.metadata?.header
          ? BigInt(inspectionData.metadata.header.endTimeUnixSeconds) *
            1_000_000_000n
          : 1_000_000n;
        this.pixelsPerNs.set(3000 / Math.max(1, Number(maxTimeNs - minTimeNs)));
        this.viewportLeftTimeNs.set(minTimeNs);
      }
    });
  }

  viewModel = computed<TimelineFrameStoryViewModel>(() => {
    const data = this.khiInspectionData.value();
    if (!data) {
      return {
        ready: false,
      };
    }
    const minTimeNs = data.metadata?.header
      ? BigInt(data.metadata.header.startTimeUnixSeconds) * 1_000_000_000n
      : 0n;
    const maxTimeNs = data.metadata?.header
      ? BigInt(data.metadata.header.endTimeUnixSeconds) * 1_000_000_000n
      : 0n;
    const logs = Array.from(data.logStore.logs());
    return {
      ready: true,
      timelines: data.timelineStore.timelines,
      logs,
      filteredLogIds: IdBitset.fromAll(logs.map((log) => log.id)),
      minLogTimeNs: minTimeNs,
      maxLogTimeNs: maxTimeNs,
      chartStyle: generateDefaultChartStyle(),
      rulerStyle: generateDefaultRulerStyle(data.styleStore),
      styleStore: data.styleStore,
    };
  });

  viewportLeftTimeNs = signal(0n);

  pixelsPerNs = signal(1e-6);

  timelineHighlights = signal<TimelineHighlight>({});

  timelineChartItemHighlights = signal<TimelineChartItemHighlight>({});

  timeCursorNs = signal(0n);

  secondaryTimeCursorNs = signal(0n);

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
    const pixelsPerNs = this.pixelsPerNs();
    const rangeNs = BigInt(Math.round(300 / pixelsPerNs)); // select 300 pixel around
    if (e.revisionIndex !== undefined) {
      const revision = e.timeline.revisions[e.revisionIndex];
      this.timelineChartItemHighlights.set({
        ...filterChartItemHighlight(
          this.timelineChartItemHighlights(),
          (h) => h !== TimelineChartItemHighlightType.Selected,
        ),
        [revision.logIndex]: TimelineChartItemHighlightType.Selected,
      });
      this.timeCursorNs.set(revision.changedTime);
      this.secondaryTimeCursorNs.set(
        revision.log.timestamp !== revision.changedTime
          ? revision.log.timestamp
          : 0n,
      );

      const events = e.timeline.lookupEventsInRangeNs(
        e.timeNs - rangeNs,
        e.timeNs + rangeNs,
      );
      const revisions = e.timeline.lookupRevisionsInRangeNs(
        e.timeNs - rangeNs,
        e.timeNs + rangeNs,
      );
      this.timelineHoverRequest.set({
        timelineId: e.timeline.id,
        timeNs: e.timeNs,
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
      this.timeCursorNs.set(event.timestamp);
      this.secondaryTimeCursorNs.set(0n);
      const events = e.timeline.lookupEventsInRangeNs(
        event.timestamp - rangeNs,
        event.timestamp + rangeNs,
      );
      const revisions = e.timeline.lookupRevisionsInRangeNs(
        event.timestamp - rangeNs,
        event.timestamp + rangeNs,
      );
      this.timelineHoverRequest.set({
        timelineId: e.timeline.id,
        timeNs: e.timeNs,
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
  title: 'Timeline/Main/TimelineFrame',
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
