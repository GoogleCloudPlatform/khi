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
import { TimelineCornerIndicatorComponent } from './timeline-corner-indicator.component';

const meta: Meta<TimelineCornerIndicatorComponent> = {
  title: 'Timeline/CornerIndicator',
  component: TimelineCornerIndicatorComponent,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
  argTypes: {},
  decorators: [
    componentWrapperDecorator(
      (story) => `
      <div style="height: 60px; width: 300px;">
        ${story}
      </div>
    `,
    ),
  ],
};

export default meta;
type Story = StoryObj<TimelineCornerIndicatorComponent>;

export const Default: Story = {
  args: {
    isCurrentScalingMode: false,
  },
};

export const CurrentScalingMode: Story = {
  args: {
    isCurrentScalingMode: true,
  },
};
