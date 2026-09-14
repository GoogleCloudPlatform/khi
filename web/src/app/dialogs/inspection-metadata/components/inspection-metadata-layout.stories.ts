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
import { EstimatedCountPreset } from 'src/app/common/schema/metadata-types';
import { InspectionMetadataLayoutComponent } from './inspection-metadata-layout.component';

const meta: Meta<InspectionMetadataLayoutComponent> = {
  title: 'Dialogs/InspectionMetadata/InspectionMetadataLayout',
  component: InspectionMetadataLayoutComponent,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<InspectionMetadataLayoutComponent>;

export const Default: Story = {
  args: {
    data: {
      overview: {
        inspectionType: 'gcp-gke',
        inspectionName: 'Production GKE Cluster - Node Pool Failure',
        inspectionTypeIconPath: '',
        formattedStartTime: '2023/11/14 22:13:20',
        formattedEndTime: '2023/11/15 00:13:20',
        durationText: '2h',
        suggestedFilename: 'prod-gke-cluster-node-pool-failure.khi',
        fileSizeText: '14.8 MB',
      },
      queries: [
        {
          id: 'k8s-audit',
          name: 'GKE Kubernetes Audit Logs',
          query:
            'resource.type="k8s_cluster"\nlogName="projects/test-project/logs/cloudaudit.googleapis.com%2Factivity"',
          estimatedCount: 24500,
          pending: false,
          incomplete: false,
        },
        {
          id: 'node-systemd',
          name: 'Node systemd journals',
          query: 'resource.type="k8s_node"\nlogName=~"systemd"',
          estimatedCountPreset: EstimatedCountPreset.Few,
          pending: false,
          incomplete: false,
        },
      ],
      logs: [
        {
          id: 'fetch-audit',
          name: 'AuditLogFetcherTask',
          log: '[INFO] Initializing Google Cloud Logging client\n[INFO] Filter configured: resource.type="k8s_cluster"\n[INFO] Stream completed: 24500 entries received\n[INFO] Processed in 1.42s',
        },
        {
          id: 'parse-k8s',
          name: 'KubernetesEventParserTask',
          log: '[INFO] Starting event parsing\n[INFO] Identified 12 namespaces, 84 pods, 16 nodes\n[INFO] Timeline graph constructed with 1042 revisions',
        },
      ],
      plan: {
        taskGraph:
          'digraph G {\n  rankdir=LR;\n  node [shape=box];\n  "AuditLogFetcherTask" -> "KubernetesEventParserTask";\n  "KubernetesEventParserTask" -> "TimelineBuilderTask";\n}',
      },
      errors: [],
    },
  },
};

export const WithErrors: Story = {
  args: {
    data: {
      overview: {
        inspectionType: 'gcp-gke',
        inspectionName: 'Cluster Beta Partial Inspection',
        inspectionTypeIconPath: '',
        formattedStartTime: '2023/11/14 22:13:20',
        formattedEndTime: '2023/11/14 23:13:20',
        durationText: '1h',
        suggestedFilename: 'cluster-beta-partial.khi',
        fileSizeText: '2.1 MB',
      },
      queries: [
        {
          id: 'k8s-audit',
          name: 'GKE Kubernetes Audit Logs',
          query: 'resource.type="k8s_cluster"',
          incomplete: true,
        },
      ],
      logs: [
        {
          id: 'fetch-audit',
          name: 'AuditLogFetcherTask',
          log: '[ERROR] Request deadline exceeded while querying Cloud Logging API\n[WARN] Partial result returned',
        },
      ],
      plan: {
        taskGraph: 'digraph G { "Fetch" -> "Parse"; }',
      },
      errors: [
        {
          errorId: 'DEADLINE_EXCEEDED',
          message:
            'The query took too long and timed out before all entries were fetched. Try narrowing the inspection time range.',
          link: 'https://cloud.google.com/logging/docs/reference/v2/rpc/google.logging.v2',
        },
        {
          errorId: 'RATE_LIMIT_WARNING',
          message:
            'Logging API quota limit was approached during inspection execution.',
          link: '',
        },
      ],
    },
  },
};

export const Minimal: Story = {
  args: {
    data: {
      overview: {
        inspectionType: 'local-file',
        inspectionName: 'Uploaded Archive',
        inspectionTypeIconPath: '',
        formattedStartTime: '2023/11/14 22:13:20',
        formattedEndTime: '2023/11/14 22:23:20',
        durationText: '10m',
        suggestedFilename: 'uploaded-archive.khi',
        fileSizeText: '840 KB',
      },
      queries: [],
      logs: [],
      plan: {
        taskGraph: '',
      },
      errors: [],
    },
  },
};
