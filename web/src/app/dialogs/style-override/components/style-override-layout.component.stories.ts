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
import { StyleOverrideLayoutComponent } from 'src/app/dialogs/style-override/components/style-override-layout.component';
import { RevisionStateStyle } from 'src/app/store/domain/style';

const meta: Meta<StyleOverrideLayoutComponent> = {
  title: 'Dialogs/StyleOverrideLayout',
  component: StyleOverrideLayoutComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<StyleOverrideLayoutComponent>;

export const Default: Story = {
  args: {
    revisionStates: [
      {
        id: 1,
        label: 'State is "True"',
        icon: 'lightbulb',
        description: 'Resource condition is true',
        hexColor: '#004400',
        isOverridden: false,
        style: RevisionStateStyle.NORMAL,
        goColorCode: 'style.Color{R: 0.000, G: 0.267, B: 0.000, A: 1.0}',
      },
      {
        id: 2,
        label: 'State is "False"',
        icon: 'light_off',
        description: 'Resource condition is false',
        hexColor: '#ee4400',
        isOverridden: true,
        style: RevisionStateStyle.NORMAL,
        goColorCode: 'style.Color{R: 0.933, G: 0.267, B: 0.000, A: 1.0}',
      },
      {
        id: 3,
        label: 'State is "Unknown"',
        icon: 'siren_question',
        description: 'Resource condition is unknown',
        hexColor: '#663366',
        isOverridden: false,
        style: RevisionStateStyle.NORMAL,
        goColorCode: 'style.Color{R: 0.400, G: 0.200, B: 0.400, A: 1.0}',
      },
    ],
    timelineTypes: [
      {
        id: 1,
        label: 'Pod',
        icon: 'pod',
        description: 'Kubernetes Pod resources',
        hexColor: '#3f51b5',
        hexForegroundColor: '#ffffff',
        hexChipBackgroundColor: '#1a237e',
        hexChipForegroundColor: '#ffffff',
        height: 1.2,
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ],
    logTypes: [
      {
        id: 1,
        label: 'Audit',
        description: 'Kubernetes Audit Logs',
        hexColor: '#00bcd4',
        hexForegroundColor: '#ffffff',
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.000, G: 0.737, B: 0.831, A: 1.0}',
      },
    ],
  },
};
