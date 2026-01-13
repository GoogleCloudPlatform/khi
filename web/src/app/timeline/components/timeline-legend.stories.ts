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
import { TimelineLegendComponent } from './timeline-legend.component';
import { ResourceTimeline } from 'src/app/store/timeline';
import {
  LogType,
  ParentRelationship,
  RevisionState,
  RevisionVerb,
  Severity,
} from 'src/app/generated';
import { ResourceRevision } from 'src/app/store/revision';
import { ResourceEvent } from 'src/app/store/event';

const meta: Meta<TimelineLegendComponent> = {
  title: 'Timeline/Legend',
  component: TimelineLegendComponent,
  tags: ['autodocs'],
  args: {
    expanded: true,
    legendType: 'revisions',
  },
};

export default meta;
type Story = StoryObj<TimelineLegendComponent>;

function createRevisionForRevisionState(state: RevisionState) {
  return new ResourceRevision(
    0,
    0,
    state,
    RevisionVerb.RevisionVerbCreate,
    '',
    '',
    false,
    false,
    0,
  );
}

const timeline = new ResourceTimeline(
  '',
  'core/v1#pods#default#foo',
  [
    createRevisionForRevisionState(RevisionState.RevisionStateInferred),
    createRevisionForRevisionState(RevisionState.RevisionStateProvisioning),
    createRevisionForRevisionState(RevisionState.RevisionStateDeleting),
    createRevisionForRevisionState(RevisionState.RevisionStateDeleted),
  ],
  [
    new ResourceEvent(0, 0, LogType.LogTypeAudit, Severity.SeverityInfo),
    new ResourceEvent(0, 0, LogType.LogTypeComputeApi, Severity.SeverityInfo),
    new ResourceEvent(0, 0, LogType.LogTypeGkeAudit, Severity.SeverityInfo),
  ],
  ParentRelationship.RelationshipChild,
);

const kind = new ResourceTimeline(
  '',
  'core/v1#default',
  [],
  [],
  ParentRelationship.RelationshipChild,
);

export const Default: Story = {
  args: {
    timeline,
  },
};

export const NonResourceOrSubresource: Story = {
  args: {
    timeline: kind,
  },
};

export const NoSelection: Story = {
  args: {
    timeline: null,
  },
};
