/**
 * Copyright 2024 Google LLC
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

import { LongTimestampFormatPipe } from '../../../../common/timestamp-format.pipe';
import {
  ContainerGraphData,
  GraphData,
  GraphNode,
  PodGraphData,
  ServiceGraphData,
} from '../../../../common/schema/graph-schema';
import { rendererIt } from './test/graph-test-utiility';

function $node(name: string, ...pods: PodGraphData[]): GraphNode {
  return {
    name,
    labels: {},
    pods,
    podCIDR: '10.0.0.0/8',
    externalIP: '10.0.0.0',
    internalIP: '11.0.0.0',
    taints: ['sample-taint(NoExecute)'],
    conditions: [],
  };
}

function $pod(
  namespace: string,
  name: string,
  ...containers: ContainerGraphData[]
): PodGraphData {
  return {
    name,
    namespace,
    labels: {},
    containers,
    podIP: '10.0.0.1',
    conditions: [],
    phase: 'Running',
    isPhaseHealthy: true,
    ownerUids: new Set(),
  };
}

function $container(
  name: string,
  isInit: boolean,
  status: string,
  ready: boolean,
): ContainerGraphData {
  return {
    name,
    isInitContainer: isInit,
    status,
    ready,
    isStatusHealthy: true,
    reason: 'Unknown',
    code: 0,
    statusReadFromManifest: true,
  };
}

function $service(name: string, namespace: string): ServiceGraphData {
  return {
    name,
    namespace,
    labels: {},
    clusterIp: '10.0.0.3',
    type: 'ClusterIP',
    connectedPods: [],
  };
}

describe('Graph renderer', () => {
  rendererIt(
    'Updating Graph Data should remove the previous svg node',
    300,
    (renderer) => {
      expect(renderer.containerHTMLElement.querySelectorAll('svg').length).toBe(
        1,
      );
      expect(
        renderer.containerHTMLElement.querySelectorAll('defs').length,
      ).toBe(1);

      renderer.updateGraphData({
        nodes: [$node('node-bar')],
        services: [
          $service('foo-service', 'kube-system'),
          $service('bar-service', 'qux-namespace'),
        ],
        graphTime: LongTimestampFormatPipe.toLongDisplayTimestamp(0, 0),
        podOwnerOwners: {
          cronjob: [],
          deployment: [],
        },
        podOwners: {
          daemonset: [],
          job: [],
          replicaset: [],
        },
      });

      expect(renderer.containerHTMLElement.querySelectorAll('svg').length).toBe(
        1,
      );
      expect(
        renderer.containerHTMLElement.querySelectorAll('defs').length,
      ).toBe(1);
    },
  );

  rendererIt(
    'Simple node data should be displayed correctly',
    1000,
    (renderer) => {
      const mockData: GraphData = {
        nodes: [
          $node(
            'node-foo',
            $pod(
              'kube-system',
              'foo-sytem-very-very-very-very-long-pod-name',
              $container('init-container-1', true, 'Completed', false),
              $container('container-1', false, 'Running', true),
            ),
            $pod(
              'kube-system',
              'bar-sytem',
              $container('init-container-2', true, 'Completed', false),
              $container('container-2', false, 'Running', true),
            ),
          ),
          $node(
            'node-bar',
            $pod(
              'kube-system',
              'foo-sytem',
              $container('init-container-1', true, 'Completed', false),
              $container('container-1', false, 'Running', true),
            ),
            $pod(
              'kube-system',
              'bar-sytem',
              $container('init-container-2', true, 'Completed', false),
              $container('container-2', false, 'Running', true),
            ),
          ),
        ],
        services: [
          $service('foo-service', 'kube-system'),
          $service('bar-service', 'qux-namespace'),
        ],
        graphTime: LongTimestampFormatPipe.toLongDisplayTimestamp(0, 0),
        podOwnerOwners: {
          cronjob: [],
          deployment: [],
        },
        podOwners: {
          daemonset: [],
          job: [],
          replicaset: [],
        },
      };

      renderer.updateGraphData(mockData);
    },
  );
});
