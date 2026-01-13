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
import { TimelineIndexComponent } from './timeline-index.component';
import { componentWrapperDecorator } from '@storybook/angular';
import { ParentRelationship, revisionStates } from 'src/app/generated';
import { ResourceTimeline } from 'src/app/store/timeline';
import { TimelineHighlightType } from './interaction-model';

const meta: Meta<TimelineIndexComponent> = {
  title: 'Timeline/Index',
  component: TimelineIndexComponent,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
  decorators: [
    componentWrapperDecorator(
      (story) => `
      <div style="height: 100vh; width: 200px">
        ${story}
      </div>
    `,
    ),
  ],
  args: {
    timelines: createTimelines(),
  },
  argTypes: {
    hoverOnTimeline: {
      action: 'hoverOnTimeline',
    },
    clickOnTimeline: {
      action: 'clickOnTimeline',
    },
  },
};

export default meta;
type Story = StoryObj<TimelineIndexComponent>;

function createTimeline(
  tid: string,
  path: string,
  relationship: ParentRelationship,
): ResourceTimeline {
  return new ResourceTimeline(tid, path, [], [], relationship);
}

function createTimelines(): ResourceTimeline[] {
  const result = [
    createTimeline(
      't-kind',
      'core/v1#foo',
      ParentRelationship.RelationshipChild,
    ),
    createTimeline(
      't-namespace',
      'core/v1#foo#bar',
      ParentRelationship.RelationshipChild,
    ),
    createTimeline(
      't-resource',
      'core/v1#foo#bar#resource-1',
      ParentRelationship.RelationshipChild,
    ),
  ];
  for (let i = 0; i < revisionStates.length; i++) {
    result.push(
      createTimeline(`t-sub${i}`, `core/v1#foo#bar#resource-1#sub${i}`, i),
    );
  }
  return result;
}

export const Default: Story = {};

export const SelectionAndHover: Story = {
  args: {
    highlights: {
      't-resource': TimelineHighlightType.Selected,
      't-sub0': TimelineHighlightType.ChildrenOfSelected,
      't-sub1': TimelineHighlightType.ChildrenOfSelected,
      't-sub2': TimelineHighlightType.ChildrenOfSelected,
      't-sub5': TimelineHighlightType.Hovered,
    },
  },
};

export const SelectingKind: Story = {
  args: {
    highlights: {
      't-kind': TimelineHighlightType.Selected,
    },
  },
};

export const SelectingNamespace: Story = {
  args: {
    highlights: {
      't-namespace': TimelineHighlightType.Selected,
    },
  },
};
