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

import { TestBed } from '@angular/core/testing';
import { create } from '@bufbuild/protobuf';
import { BehaviorSubject } from 'rxjs';
import {
  GetArchitectureGraphResponseSchema,
  GraphEdge_EdgeType,
} from 'src/app/generated/api/v1/architecture_graph_pb';
import { SparseBitsetSchema } from 'src/app/generated/api/v1/sparse_bitset_pb';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { ViewStateService } from 'src/app/services/view-state.service';
import { GraphConverterService } from 'src/app/services/graph-converter.service';

describe('GraphConverterService', () => {
  let service: GraphConverterService;
  let mockWorkbenchClient: jasmine.SpyObj<WorkbenchClientService>;
  let mockViewStateService: { timezoneShift: BehaviorSubject<number> };

  beforeEach(() => {
    mockWorkbenchClient = jasmine.createSpyObj('WorkbenchClientService', [
      'getArchitectureGraph',
    ]);
    mockViewStateService = {
      timezoneShift: new BehaviorSubject<number>(0),
    };

    TestBed.configureTestingModule({
      providers: [
        GraphConverterService,
        { provide: WorkbenchClientService, useValue: mockWorkbenchClient },
        { provide: ViewStateService, useValue: mockViewStateService },
      ],
    });

    service = TestBed.inject(GraphConverterService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('convertToGraphData', () => {
    it('should convert proto response into GraphData with correct node, pod, and edge associations', () => {
      const timestampNs = 10_000_000_000n; // 10s
      const protoResponse = create(GetArchitectureGraphResponseSchema, {
        timestampNs,
        nodes: [
          {
            id: 'node-1',
            name: 'node-a',
            podCidr: '10.244.0.0/24',
            internalIp: '192.168.1.10',
            externalIp: '35.200.0.1',
            taints: ['node-role.kubernetes.io/master:NoSchedule'],
            conditions: [
              {
                type: 'Ready',
                status: 'True',
                message: 'kubelet is posting ready status',
                isPositive: true,
              },
            ],
            labels: { 'kubernetes.io/hostname': 'node-a' },
            updatedAtNs: 9_000_000_000n, // 1s ago
          },
        ],
        pods: [
          {
            id: 'pod-1',
            name: 'frontend-pod',
            namespace: 'default',
            uid: 'uid-pod-1',
            nodeName: 'node-a',
            podIp: '10.244.0.5',
            phase: 'Running',
            isPhaseHealthy: true,
            conditions: [
              {
                type: 'Ready',
                status: 'True',
                message: '',
                isPositive: true,
              },
            ],
            containers: [
              {
                name: 'nginx',
                isInitContainer: false,
                isStatusHealthy: true,
                status: 'Running',
                reason: '',
                code: 0,
                ready: true,
                statusReadFromManifest: true,
              },
            ],
            ownerUids: ['uid-rs-1'],
            labels: { app: 'frontend' },
            updatedAtNs: 8_000_000_000n, // 2s ago
          },
        ],
        services: [
          {
            id: 'svc-1',
            name: 'frontend-service',
            namespace: 'default',
            uid: 'uid-svc-1',
            type: 'ClusterIP',
            clusterIp: '10.96.0.1',
            labels: { app: 'frontend' },
            updatedAtNs: 7_000_000_000n,
          },
        ],
        podOwners: [
          {
            id: 'po-1',
            name: 'frontend-rs',
            namespace: 'default',
            kind: 'replicaset',
            uid: 'uid-rs-1',
            ownerUids: ['uid-deploy-1'],
            labels: { app: 'frontend' },
            updatedAtNs: 6_000_000_000n,
          },
        ],
        podOwnerOwners: [
          {
            id: 'poo-1',
            name: 'frontend-deploy',
            namespace: 'default',
            kind: 'deployment',
            uid: 'uid-deploy-1',
            labels: { app: 'frontend' },
            updatedAtNs: 5_000_000_000n,
          },
        ],
        edges: [
          {
            type: GraphEdge_EdgeType.SERVICE_TO_POD,
            sourceId: 'svc-1',
            targetId: 'pod-1',
          },
          {
            type: GraphEdge_EdgeType.POD_OWNER_TO_POD,
            sourceId: 'po-1',
            targetId: 'pod-1',
          },
          {
            type: GraphEdge_EdgeType.POD_OWNER_OWNER_TO_POD_OWNER,
            sourceId: 'poo-1',
            targetId: 'po-1',
          },
        ],
      });

      const result = service.convertToGraphData(protoResponse, timestampNs);

      expect(result.nodes.length).toBe(1);
      const node = result.nodes[0];
      expect(node.name).toBe('node-a');
      expect(node.podCIDR).toBe('10.244.0.0/24');
      expect(node.internalIP).toBe('192.168.1.10');
      expect(node.externalIP).toBe('35.200.0.1');
      expect(node.updatedAt).toBe('1.00s ago');
      expect(node.pods.length).toBe(1);

      const pod = node.pods[0];
      expect(pod.name).toBe('frontend-pod');
      expect(pod.namespace).toBe('default');
      expect(pod.podIP).toBe('10.244.0.5');
      expect(pod.phase).toBe('Running');
      expect(pod.isPhaseHealthy).toBeTrue();
      expect(pod.updatedAt).toBe('2.00s ago');

      expect(result.services.length).toBe(1);
      const svc = result.services[0];
      expect(svc.name).toBe('frontend-service');
      expect(svc.connectedPods.length).toBe(1);
      expect(svc.connectedPods[0].pod.name).toBe('frontend-pod');
      expect(svc.connectedPods[0].node.name).toBe('node-a');

      expect(result.podOwners.replicaset.length).toBe(1);
      const rs = result.podOwners.replicaset[0];
      expect(rs.name).toBe('frontend-rs');
      expect(rs.connectedPods.length).toBe(1);
      expect(rs.connectedPods[0].pod.name).toBe('frontend-pod');

      expect(result.podOwnerOwners.deployment.length).toBe(1);
      const deploy = result.podOwnerOwners.deployment[0];
      expect(deploy.name).toBe('frontend-deploy');
      expect(deploy.connectedPodOwners.length).toBe(1);
      expect(deploy.connectedPodOwners[0].podOwner.name).toBe('frontend-rs');
    });

    it('should ignore pod if nodeName does not match any known node', () => {
      const timestampNs = 5_000_000_000n;
      const protoResponse = create(GetArchitectureGraphResponseSchema, {
        timestampNs,
        nodes: [],
        pods: [
          {
            id: 'pod-unobserved',
            name: 'standalone-pod',
            namespace: 'kube-system',
            nodeName: 'unobserved-node',
            phase: 'Running',
          },
        ],
      });

      const result = service.convertToGraphData(protoResponse, timestampNs);
      expect(result.nodes.length).toBe(0);
    });

    it('should format deletedAt timestamp for deleted resources', () => {
      const timestampNs = 10_000_000_000n;
      const protoResponse = create(GetArchitectureGraphResponseSchema, {
        timestampNs,
        nodes: [
          {
            id: 'node-deleted',
            name: 'node-old',
            deletedAtNs: 8_500_000_000n, // 1.5s ago
          },
        ],
      });

      const result = service.convertToGraphData(protoResponse, timestampNs);
      expect(result.nodes[0].deletedAt).toBe('1.50s ago');
      expect(result.nodes[0].updatedAt).toBeUndefined();
    });
  });

  describe('getGraphDataAt', () => {
    it('should call workbenchClient.getArchitectureGraph and return converted data', async () => {
      const timestampNs = 10_000_000_000n;
      const sparseBitset = create(SparseBitsetSchema, {
        indices: [0],
        masks: [1],
      });
      const protoResponse = create(GetArchitectureGraphResponseSchema, {
        timestampNs,
        nodes: [{ id: 'n1', name: 'node-1' }],
      });
      mockWorkbenchClient.getArchitectureGraph.and.resolveTo(protoResponse);

      const result = await service.getGraphDataAt(
        timestampNs,
        sparseBitset,
        180,
        undefined,
      );

      expect(mockWorkbenchClient.getArchitectureGraph).toHaveBeenCalledWith(
        timestampNs,
        sparseBitset,
        180,
        undefined,
      );
      expect(result.nodes.length).toBe(1);
      expect(result.nodes[0].name).toBe('node-1');
    });
  });
});
