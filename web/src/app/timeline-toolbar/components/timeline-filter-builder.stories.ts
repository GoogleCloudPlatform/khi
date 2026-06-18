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

import { Meta, StoryObj, moduleMetadata } from '@storybook/angular';
import { TimelineFilterBuilderComponent } from './timeline-filter-builder.component';
import { CommonModule } from '@angular/common';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { TimelineType } from 'src/app/store/domain/style';

const createMockType = (
  label: string,
  color: [number, number, number, number],
): TimelineType => ({
  id: Math.random(),
  label,
  description: `Mock ${label}`,
  icon: 'timeline',
  backgroundColor: { r: color[0], g: color[1], b: color[2], a: color[3] },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
  typeChipBackgroundColor: {
    r: color[0],
    g: color[1],
    b: color[2],
    a: color[3],
  },
  typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
  visible: true,
  sortPriority: 0,
  height: 24,
});

const MOCK_TIMELINE_TYPES: TimelineType[] = [
  createMockType('K8sResource', [0.25, 0.32, 0.71, 1]),
  createMockType('K8sNamespace', [0.78, 0.18, 0.36, 1]),
  createMockType('Gke', [0.15, 0.68, 0.37, 1]),
];

const meta: Meta<TimelineFilterBuilderComponent> = {
  title: 'Timeline/Components/TimelineFilterBuilder',
  component: TimelineFilterBuilderComponent,
  tags: ['autodocs'],
  decorators: [
    moduleMetadata({
      imports: [
        CommonModule,
        MatSelectModule,
        MatInputModule,
        MatFormFieldModule,
        MatButtonToggleModule,
        MatButtonModule,
        MatIconModule,
        NoopAnimationsModule,
      ],
    }),
  ],
};

export default meta;
type Story = StoryObj<TimelineFilterBuilderComponent>;

export const Default: Story = {
  args: {
    timelineTypes: MOCK_TIMELINE_TYPES,
    candidates: [],
    selectedTimelineType: '*',
    filterMode: 'regex',
    regexValue: '',
    selectedCandidates: [],
    showDeleteButton: false,
  },
};

export const TypeSelectedRegexMode: Story = {
  args: {
    timelineTypes: MOCK_TIMELINE_TYPES,
    candidates: ['Pod', 'Service', 'Deployment'],
    selectedTimelineType: 'K8sResource',
    filterMode: 'regex',
    regexValue: 'Pod',
    selectedCandidates: [],
    showDeleteButton: false,
  },
};

export const TypeSelectedSelectionMode: Story = {
  args: {
    timelineTypes: MOCK_TIMELINE_TYPES,
    candidates: ['Pod', 'Service', 'Deployment'],
    selectedTimelineType: 'K8sResource',
    filterMode: 'selection',
    regexValue: '',
    selectedCandidates: ['Pod'],
    showDeleteButton: false,
  },
};

export const EditableWithDeleteButton: Story = {
  args: {
    timelineTypes: MOCK_TIMELINE_TYPES,
    candidates: ['Pod', 'Service', 'Deployment'],
    selectedTimelineType: 'K8sResource',
    filterMode: 'regex',
    regexValue: 'Pod',
    selectedCandidates: [],
    showDeleteButton: true,
  },
};
