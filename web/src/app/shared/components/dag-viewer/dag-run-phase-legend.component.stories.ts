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
import { DagRunPhaseLegendComponent } from 'src/app/shared/components/dag-viewer/dag-run-phase-legend.component';

const meta: Meta<DagRunPhaseLegendComponent> = {
  title: 'Shared/DagViewer/DagRunPhaseLegend',
  component: DagRunPhaseLegendComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<DagRunPhaseLegendComponent>;

export const Default: Story = {
  args: {},
};
