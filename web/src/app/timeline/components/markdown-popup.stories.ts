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
import { MarkdownPopupComponent } from 'src/app/timeline/components/markdown-popup.component';

export default {
  title: 'Timeline/MarkdownPopup',
  component: MarkdownPopupComponent,
  decorators: [
    moduleMetadata({
      imports: [MarkdownPopupComponent],
    }),
  ],
} as Meta<MarkdownPopupComponent>;

type Story = StoryObj<MarkdownPopupComponent>;

export const Default: Story = {
  args: {
    markdown: `# Sample Header
This is a sample markdown description containing **bold text**, *italic text*, and code snippets:

\`\`\`yaml
kind: Pod
metadata:
  name: sample-pod
\`\`\`

- List item 1
- List item 2
`,
  },
};
