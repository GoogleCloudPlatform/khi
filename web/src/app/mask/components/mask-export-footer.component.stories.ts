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
import { MaskExportFooterComponent } from 'src/app/mask/components/mask-export-footer.component';
import { MaskExportPhase } from 'src/app/mask/types/mask-viewmodel';

const meta: Meta<MaskExportFooterComponent> = {
  title: 'Mask/MaskExportFooter',
  component: MaskExportFooterComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MaskExportFooterComponent>;

export const ReadyToExport: Story = {
  args: {
    ruleCount: 2,
    enabledRuleCount: 2,
    isExporting: false,
    exportProgress: null,
    statusMessage: 'Downloaded masked.khi with 2 mask rules.',
    errorMessage: '',
  },
};

export const ExportingInProgress: Story = {
  args: {
    ruleCount: 2,
    enabledRuleCount: 2,
    isExporting: true,
    exportProgress: {
      phase: MaskExportPhase.GeneratingArchive,
      phaseLabel: 'Generating masked .khi archive...',
      progressPercentage: 58,
      message: 'Masking string pool (16,384 / 28,400 strings)...',
    },
    statusMessage: '',
    errorMessage: '',
  },
};
