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
  Meta,
  moduleMetadata,
  StoryObj,
  componentWrapperDecorator,
} from '@storybook/angular';
import {
  ParentRelationship,
  RevisionState,
  RevisionVerb,
} from 'src/app/generated';
import { TimelineChartComponent } from './timeline-chart.component';
import { Component, DestroyRef, inject, NgZone, OnInit } from '@angular/core';
import { RenderingLoopManager } from './canvas/rendering-loop-manager';
import { DemoViewModelBuilder } from './misc/demo-builder';

@Component({
  selector: 'khi-rendering-loop-starter',
  template: `<ng-content></ng-content>`,
  standalone: true,
})
class RenderingLoopStarter implements OnInit {
  private readonly renderingLoopManager = inject(RenderingLoopManager);
  private readonly ngZone = inject(NgZone);
  private readonly destroyRef = inject(DestroyRef);

  ngOnInit() {
    this.renderingLoopManager.start(this.ngZone, this.destroyRef);
  }
}

const START_TIME = new Date(2025, 0, 1, 0, 0, 0).getTime();
const DURATION = 60 * 60 * 1000; // 1 hour

function generateMockTimelineChartViewModel(): DemoViewModelBuilder {
  const builder = new DemoViewModelBuilder(START_TIME, START_TIME + DURATION);
  builder.createTimeline('core/v1#pod', ParentRelationship.RelationshipChild);
  builder.createTimeline(
    'core/v1#pod#default',
    ParentRelationship.RelationshipChild,
  );
  builder.createTimeline(
    'core/v1#pod#default#pod-1',
    ParentRelationship.RelationshipChild,
    builder.createRevision(
      START_TIME + DURATION * 0.1,
      START_TIME + DURATION * 0.2,
      RevisionState.RevisionStateProvisioning,
      RevisionVerb.RevisionVerbCreate,
    ),
    builder.createRevision(
      START_TIME + DURATION * 0.2,
      START_TIME + DURATION * 0.3,
      RevisionState.RevisionStateExisting,
      RevisionVerb.RevisionVerbUpdate,
    ),
    builder.createRevision(
      START_TIME + DURATION * 0.3,
      START_TIME + DURATION * 0.4,
      RevisionState.RevisionStateDeleting,
      RevisionVerb.RevisionVerbDelete,
    ),
    builder.createRevision(
      START_TIME + DURATION * 0.4,
      START_TIME + DURATION * 0.5,
      RevisionState.RevisionStateDeleted,
      RevisionVerb.RevisionVerbDelete,
    ),
  );
  builder.createTimeline(
    'core/v1#pod#default#pod-2',
    ParentRelationship.RelationshipChild,
    builder.createRevision(
      START_TIME + DURATION * 0.1,
      START_TIME + DURATION * 0.2,
      RevisionState.RevisionStateInferred,
      RevisionVerb.RevisionVerbCreate,
    ),
  );
  builder.createTimeline(
    'core/v1#pod#default#pod-2#ready-status',
    ParentRelationship.RelationshipChild,
    builder.createRevision(
      START_TIME + DURATION * 0.1,
      START_TIME + DURATION * 0.2,
      RevisionState.RevisionStateConditionNotGiven,
      RevisionVerb.RevisionVerbCreate,
    ),
    builder.createRevision(
      START_TIME + DURATION * 0.2,
      START_TIME + DURATION * 0.3,
      RevisionState.RevisionStateConditionNoAvailableInfo,
      RevisionVerb.RevisionVerbCreate,
    ),
    builder.createRevision(
      START_TIME + DURATION * 0.3,
      START_TIME + DURATION * 0.4,
      RevisionState.RevisionStateConditionUnknown,
      RevisionVerb.RevisionVerbCreate,
    ),
    builder.createRevision(
      START_TIME + DURATION * 0.4,
      START_TIME + DURATION * 0.5,
      RevisionState.RevisionStateConditionTrue,
      RevisionVerb.RevisionVerbCreate,
    ),
    builder.createRevision(
      START_TIME + DURATION * 0.5,
      START_TIME + DURATION * 0.6,
      RevisionState.RevisionStateConditionFalse,
      RevisionVerb.RevisionVerbCreate,
    ),
  );
  return builder;
}

const builder = generateMockTimelineChartViewModel();

const meta: Meta<TimelineChartComponent> = {
  title: 'Timeline/TimelineChart',
  component: TimelineChartComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [RenderingLoopStarter],
    }),
    componentWrapperDecorator(
      (story) => `
      <khi-rendering-loop-starter style="height: 100vh; display: grid;">
         ${story}
      </khi-rendering-loop-starter>`,
    ),
  ],
  parameters: {
    layout: 'fullscreen',
  },
  args: {
    chartViewModel: builder.getChartViewModel(),
    rulerViewModel: builder.getRulerViewModel(window.innerWidth),
    activeLogsIndices: builder.getAllActiveLogIndices(),
    leftEdgeTime: START_TIME,
    pixelsPerMs: window.innerWidth / DURATION,
    timelineHighlights: {},
    timelineChartItemHighlights: {},
  },
};

export default meta;
type Story = StoryObj<TimelineChartComponent>;

export const Default: Story = {
  args: {},
};
