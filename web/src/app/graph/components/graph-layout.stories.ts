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
import { GraphLayoutComponent } from './graph-layout.component';
import { emptyGraphData } from 'src/app/common/schema/graph-schema';

const meta: Meta<GraphLayoutComponent> = {
  title: 'Graph/GraphLayout',
  component: GraphLayoutComponent,
  tags: ['autodocs'],
  args: {
    graphData: emptyGraphData(),
  },
};

export default meta;
type Story = StoryObj<GraphLayoutComponent>;

export const Default: Story = {
  render: (args) => ({
    props: {
      ...args,
    },
    template: `
      <div style="height: 500px; width: 100%;">
        <khi-graph-layout [graphData]="graphData"></khi-graph-layout>
      </div>
    `,
  }),
};

export const WithNodeAndPod: Story = {
  ...Default,
  args: {
    graphData: {
      graphTime: '2026-06-16 12:00:00',
      nodes: [
        {
          name: 'k8s-node-1',
          podCIDR: '10.244.0.0/24',
          taints: [],
          conditions: [],
          internalIP: '192.168.1.1',
          externalIP: '-',
          labels: {},
          pods: [
            {
              uid: 'pod-uid-1',
              name: 'nginx-pod',
              namespace: 'default',
              labels: { app: 'nginx' },
              podIP: '10.244.0.5',
              phase: 'Running',
              isPhaseHealthy: true,
              conditions: [],
              containers: [
                {
                  name: 'nginx',
                  status: 'Running',
                  isStatusHealthy: true,
                  isInitContainer: false,
                  ready: true,
                  code: 0,
                  reason: '',
                  statusReadFromManifest: true,
                },
              ],
              ownerUids: new Set(['owner-rs-1']),
            },
          ],
        },
      ],
      services: [],
      podOwners: {
        replicaset: [
          {
            uid: 'owner-rs-1',
            name: 'nginx-rs',
            namespace: 'default',
            labels: { app: 'nginx' },
            status: {},
            ownerUids: new Set(),
            connectedPods: [],
          },
        ],
        daemonset: [],
        job: [],
      },
      podOwnerOwners: {
        deployment: [],
        cronjob: [],
      },
    },
  },
};
