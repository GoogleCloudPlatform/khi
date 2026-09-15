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
import { MetadataPlanComponent } from './metadata-plan.component';

const meta: Meta<MetadataPlanComponent> = {
  title: 'Dialogs/InspectionMetadata/MetadataPlan',
  component: MetadataPlanComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<MetadataPlanComponent>;

export const Default: Story = {
  args: {
    plan: {
      taskGraph: `digraph G {
  rankdir=LR;
  node [shape=box, style="rounded,filled", fillcolor="#f1f3f4", fontname="Roboto"];
  "init" -> "fetch_metadata";
  "fetch_metadata" -> "query_audit_logs";
  "fetch_metadata" -> "query_events";
  "query_audit_logs" -> "process_timeline";
  "query_events" -> "process_timeline";
  "process_timeline" -> "generate_khi";
}`,
    },
  },
};
