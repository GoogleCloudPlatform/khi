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
import { MaskLayoutComponent } from 'src/app/mask/components/mask-layout.component';
import {
  MaskExportPhase,
  StringPoolIndexPhase,
} from 'src/app/mask/types/mask-viewmodel';

const meta: Meta<MaskLayoutComponent> = {
  title: 'Mask/MaskLayout',
  component: MaskLayoutComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MaskLayoutComponent>;

export const BuildingIndex: Story = {
  args: {
    isWorkbenchActive: true,
    indexStatus: {
      phase: StringPoolIndexPhase.Building,
      progressPercentage: 48,
      message: 'Extracting trigrams (12,400 / 25,800 strings)...',
    },
    livePreviewEnabled: true,
    searchQuery: 'prod',
    searchRegexError: '',
    candidates: [
      { value: 'prod-payments-ns', selected: true, inRuleList: true },
      { value: 'gke-prod-cluster-pool-1', selected: false, inRuleList: false },
      {
        value: 'serviceaccount:prod-runner@example.iam.gserviceaccount.com',
        selected: false,
        inRuleList: false,
      },
    ],
    totalCandidateCount: 3,
    isLoadingCandidates: false,
    rules: [
      {
        target: 'prod-payments-ns',
        replacement: '<masked_1>',
        defaultReplacement: '<masked_1>',
        isDeletion: false,
        isDefaultReplacement: true,
        enabled: true,
      },
    ],
    enabledRuleCount: 1,
    isExporting: false,
    statusMessage: '',
    errorMessage: '',
  },
};

export const ReadyWithCustomRules: Story = {
  args: {
    isWorkbenchActive: true,
    indexStatus: {
      phase: StringPoolIndexPhase.Ready,
      progressPercentage: 100,
      message: 'String pool search index ready.',
    },
    livePreviewEnabled: true,
    searchQuery: '^secret-pod-\\d+$',
    searchRegexError: '',
    candidates: [
      { value: 'secret-pod-1', selected: true, inRuleList: true },
      { value: 'secret-pod-2', selected: false, inRuleList: false },
    ],
    totalCandidateCount: 2,
    isLoadingCandidates: false,
    rules: [
      {
        target: 'secret-pod-1',
        replacement: '',
        defaultReplacement: '<masked_1>',
        isDeletion: true,
        isDefaultReplacement: false,
        enabled: true,
      },
    ],
    enabledRuleCount: 1,
    isExporting: false,
    statusMessage: '',
    errorMessage: '',
  },
};

export const InvalidRegexQuery: Story = {
  args: {
    isWorkbenchActive: true,
    indexStatus: {
      phase: StringPoolIndexPhase.Ready,
      progressPercentage: 100,
      message: 'String pool search index ready.',
    },
    livePreviewEnabled: true,
    searchQuery: '^10\\.\\d+\\.[unclosed',
    searchRegexError:
      'invalid regex query: error parsing regexp: missing closing ]: `[unclosed`',
    candidates: [
      { value: '10.128.0.14', selected: false, inRuleList: false },
      { value: '10.128.0.29', selected: false, inRuleList: false },
    ],
    totalCandidateCount: 2,
    isLoadingCandidates: false,
    rules: [],
    enabledRuleCount: 0,
    isExporting: false,
    statusMessage: '',
    errorMessage: '',
  },
};

export const ExportingMaskedArchive: Story = {
  args: {
    isWorkbenchActive: true,
    indexStatus: {
      phase: StringPoolIndexPhase.Ready,
      progressPercentage: 100,
      message: 'String pool search index ready.',
    },
    livePreviewEnabled: true,
    searchQuery: '',
    searchRegexError: '',
    candidates: [
      { value: 'prod-payments-ns', selected: true, inRuleList: true },
      { value: '10.128.0.14', selected: true, inRuleList: true },
    ],
    totalCandidateCount: 2,
    isLoadingCandidates: false,
    rules: [
      {
        target: 'prod-payments-ns',
        replacement: '<masked_1>',
        defaultReplacement: '<masked_1>',
        isDeletion: false,
        isDefaultReplacement: true,
        enabled: true,
      },
      {
        target: '10.128.0.14',
        replacement: '<masked_2>',
        defaultReplacement: '<masked_2>',
        isDeletion: false,
        isDefaultReplacement: true,
        enabled: true,
      },
    ],
    enabledRuleCount: 2,
    isExporting: true,
    exportProgress: {
      phase: MaskExportPhase.GeneratingArchive,
      phaseLabel: 'Generating masked .khi archive...',
      progressPercentage: 58,
      message: 'Masking structured log strings (24,576 / 42,100 strings)...',
    },
    statusMessage: '',
    errorMessage: '',
  },
};

export const NoWorkbenchLoaded: Story = {
  args: {
    isWorkbenchActive: false,
    indexStatus: {
      phase: StringPoolIndexPhase.Idle,
      progressPercentage: 0,
      message: '',
    },
    livePreviewEnabled: true,
    searchQuery: '',
    searchRegexError: '',
    candidates: [],
    totalCandidateCount: 0,
    isLoadingCandidates: false,
    rules: [],
    enabledRuleCount: 0,
    isExporting: false,
    statusMessage: '',
    errorMessage: '',
  },
};
