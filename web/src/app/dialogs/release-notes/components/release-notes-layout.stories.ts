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

import { Meta, moduleMetadata, StoryObj } from '@storybook/angular';
import { ReleaseNotesLayoutComponent } from 'src/app/dialogs/release-notes/components/release-notes-layout.component';

const SAMPLE_RELEASE_NOTES = `# 🚀 Kubernetes History Inspector (KHI) Update

Welcome to the latest version of KHI! This update significantly improves the efficiency and visibility of large-scale log inspections.

### 🌟 What's New
- **Timeline Exclusion Filter**: Added a feature to exclude unnecessary events and noise from the timeline view.
- **Enhanced Search & Filtering**: Made the filter builder UI more intuitive for specifying complex query conditions.

### ⚡ Improvements
- **Faster Parsing**: Accelerated the Go backend log parsing process by approximately 20% for large log datasets.
- **Memory Efficiency**: Reduced the memory footprint during timeline rendering in the frontend.

### 🛠️ Fixes & Polish
- Fixed an issue where some labels overlapped during timeline rendering.
- Optimized icon and background contrast in dark mode.
`;

export default {
  title: 'Dialogs/ReleaseNotes/ReleaseNotesLayout',
  component: ReleaseNotesLayoutComponent,
  decorators: [
    moduleMetadata({
      imports: [ReleaseNotesLayoutComponent],
    }),
  ],
} as Meta<ReleaseNotesLayoutComponent>;

type Story = StoryObj<ReleaseNotesLayoutComponent>;

export const Default: Story = {
  args: {
    markdownContent: SAMPLE_RELEASE_NOTES,
    doNotShowAgain: false,
  },
};

export const CheckedSuppress: Story = {
  args: {
    markdownContent: SAMPLE_RELEASE_NOTES,
    doNotShowAgain: true,
  },
};
