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

import { Injectable, inject } from '@angular/core';
import {
  ContainerGraphData,
  GraphData,
  GraphNode,
  PodGraphData,
  GraphPodOwner,
  GraphResourceData,
  ServiceGraphData,
  GraphPodOwnerOwner,
  PodOwnerKinds,
  PodOwnerOwnerKinds,
} from 'src/app/common/schema/graph-schema';
import { LongTimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';
import { ViewStateService } from 'src/app/services/view-state.service';
import { toSignal } from '@angular/core/rxjs-interop';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import {
  GetArchitectureGraphResponse,
  GraphCondition,
  GraphContainer,
  GraphEdge_EdgeType,
} from 'src/app/generated/api/v1/architecture_graph_pb';
import { SparseBitset } from 'src/app/generated/api/v1/sparse_bitset_pb';

interface PodConnectionMapping {
  readonly node: GraphNode;
  readonly pod: PodGraphData;
}

interface PodOwnersByKind {
  readonly daemonset: GraphPodOwner[];
  readonly job: GraphPodOwner[];
  readonly replicaset: GraphPodOwner[];
}

interface PodOwnerOwnersByKind {
  readonly cronjob: GraphPodOwnerOwner[];
  readonly deployment: GraphPodOwnerOwner[];
}

/**
 * Converts architecture graph RPC responses from the backend Workbench session into cytoscape/dagre GraphData.
 */
@Injectable({
  providedIn: 'root',
})
export class GraphDataConverterService {
  private readonly viewStateService = inject(ViewStateService);
  private readonly workbenchClient = inject(WorkbenchClientService);

  private readonly timezoneShift = toSignal(
    this.viewStateService.timezoneShift,
  );

  /**
   * Fetches architecture graph data at the specified timestamp from the backend Workbench session.
   *
   * @param timestampNs - Target timestamp in nanoseconds.
   * @param timelineBitset - Optional sparse bitset to filter timelines.
   * @param abortSignal - Optional signal to cancel the generation.
   * @param deletionThresholdSeconds - Optional deletion threshold in seconds (defaults to 180).
   * @returns A promise resolving to the converted GraphData.
   */
  public async getGraphDataAt(
    timestampNs: bigint,
    timelineBitset?: SparseBitset,
    abortSignal?: AbortSignal,
    deletionThresholdSeconds = 180,
  ): Promise<GraphData> {
    const res = await this.workbenchClient.getArchitectureGraph(
      timestampNs,
      timelineBitset,
      deletionThresholdSeconds,
      abortSignal,
    );

    return this.convertToGraphData(res, timestampNs);
  }

  /**
   * Converts a GetArchitectureGraphResponse Protobuf message into the frontend GraphData format.
   *
   * @param res - Protobuf response message from GetArchitectureGraph RPC.
   * @param timestampNs - The target timestamp in nanoseconds.
   * @returns Converted GraphData.
   */
  public convertToGraphData(
    res: GetArchitectureGraphResponse,
    timestampNs: bigint,
  ): GraphData {
    const nodeByName = new Map<string, GraphNode>();

    for (const n of res.nodes) {
      const timestamps = this.formatResourceTimestamps(
        timestampNs,
        n.updatedAtNs,
        n.deletedAtNs,
      );
      const node: GraphNode = {
        name: n.name,
        podCIDR: n.podCidr,
        taints: n.taints,
        internalIP: n.internalIp,
        externalIP: n.externalIp,
        labels: n.labels,
        pods: [],
        conditions: n.conditions.map((c: GraphCondition) => ({
          type: c.type,
          message: c.message,
          status: c.status,
          is_positive_status: c.isPositive,
        })),
        ...timestamps,
      };
      nodeByName.set(n.name, node);
    }

    const podMapById = new Map<string, PodConnectionMapping>();

    for (const p of res.pods) {
      let parentNode = nodeByName.get(p.nodeName);
      if (!parentNode) {
        parentNode = {
          name: p.nodeName,
          podCIDR: '-',
          taints: [],
          pods: [],
          labels: {},
          conditions: [],
          internalIP: '-',
          externalIP: '-',
        };
        nodeByName.set(p.nodeName, parentNode);
      }

      const containers: ContainerGraphData[] = p.containers.map(
        (c: GraphContainer) => ({
          name: c.name,
          isInitContainer: c.isInitContainer,
          isStatusHealthy: c.isStatusHealthy,
          status: c.status,
          reason: c.reason,
          code: c.code,
          ready: c.ready,
          statusReadFromManifest: c.statusReadFromManifest,
        }),
      );

      const timestamps = this.formatResourceTimestamps(
        timestampNs,
        p.updatedAtNs,
        p.deletedAtNs,
      );
      const pod: PodGraphData = {
        uid: p.uid,
        name: p.name,
        namespace: p.namespace,
        labels: p.labels,
        containers,
        podIP: p.podIp,
        phase: p.phase,
        isPhaseHealthy: p.isPhaseHealthy,
        conditions: p.conditions.map((c: GraphCondition) => ({
          type: c.type,
          message: c.message,
          status: c.status,
          is_positive_status: c.isPositive,
        })),
        ownerUids: new Set(p.ownerUids),
        ...timestamps,
      };

      parentNode.pods.push(pod);
      podMapById.set(p.id, { node: parentNode, pod });
    }

    // Sort pods on each node
    for (const node of nodeByName.values()) {
      this.sortPods(node.pods);
    }

    const serviceMapById = new Map<string, ServiceGraphData>();
    const services: ServiceGraphData[] = [];

    for (const s of res.services) {
      const timestamps = this.formatResourceTimestamps(
        timestampNs,
        s.updatedAtNs,
        s.deletedAtNs,
      );
      const svc: ServiceGraphData = {
        uid: s.uid,
        name: s.name,
        namespace: s.namespace,
        labels: s.labels,
        clusterIp: s.clusterIp,
        type: s.type,
        connectedPods: [],
        ...timestamps,
      };
      serviceMapById.set(s.id, svc);
      services.push(svc);
    }

    const podOwnerMapById = new Map<string, GraphPodOwner>();
    const podOwners: PodOwnersByKind = {
      daemonset: [],
      job: [],
      replicaset: [],
    };

    for (const po of res.podOwners) {
      const timestamps = this.formatResourceTimestamps(
        timestampNs,
        po.updatedAtNs,
        po.deletedAtNs,
      );
      const owner: GraphPodOwner = {
        uid: po.uid,
        name: po.name,
        namespace: po.namespace,
        labels: po.labels,
        ownerUids: new Set(po.ownerUids),
        status: {},
        connectedPods: [],
        ...timestamps,
      };
      podOwnerMapById.set(po.id, owner);
      const kind = po.kind.toLowerCase() as PodOwnerKinds;
      if (kind === 'daemonset' || kind === 'job' || kind === 'replicaset') {
        podOwners[kind].push(owner);
      }
    }

    const podOwnerOwnerMapById = new Map<string, GraphPodOwnerOwner>();
    const podOwnerOwners: PodOwnerOwnersByKind = {
      cronjob: [],
      deployment: [],
    };

    for (const poo of res.podOwnerOwners) {
      const timestamps = this.formatResourceTimestamps(
        timestampNs,
        poo.updatedAtNs,
        poo.deletedAtNs,
      );
      const ownerOwner: GraphPodOwnerOwner = {
        uid: poo.uid,
        name: poo.name,
        namespace: poo.namespace,
        labels: poo.labels,
        status: {},
        connectedPodOwners: [],
        ...timestamps,
      };
      podOwnerOwnerMapById.set(poo.id, ownerOwner);
      const kind = poo.kind.toLowerCase() as PodOwnerOwnerKinds;
      if (kind === 'cronjob' || kind === 'deployment') {
        podOwnerOwners[kind].push(ownerOwner);
      }
    }

    // Connect edges
    for (const edge of res.edges) {
      switch (edge.type) {
        case GraphEdge_EdgeType.SERVICE_TO_POD: {
          const svc = serviceMapById.get(edge.sourceId);
          const podConn = podMapById.get(edge.targetId);
          if (svc && podConn) {
            svc.connectedPods.push(podConn);
          }
          break;
        }
        case GraphEdge_EdgeType.POD_OWNER_TO_POD: {
          const po = podOwnerMapById.get(edge.sourceId);
          const podConn = podMapById.get(edge.targetId);
          if (po && podConn) {
            po.connectedPods.push(podConn);
          }
          break;
        }
        case GraphEdge_EdgeType.POD_OWNER_OWNER_TO_POD_OWNER: {
          const poo = podOwnerOwnerMapById.get(edge.sourceId);
          const po = podOwnerMapById.get(edge.targetId);
          if (poo && po) {
            poo.connectedPodOwners.push({ podOwner: po });
          }
          break;
        }
      }
    }

    const graphTime = LongTimestampFormatPipe.toLongDisplayTimestamp(
      Number(timestampNs / 1_000_000n),
      this.timezoneShift() ?? 0,
    );

    return {
      nodes: Array.from(nodeByName.values()),
      services,
      graphTime,
      podOwners,
      podOwnerOwners,
    };
  }

  private formatResourceTimestamps(
    timestampNs: bigint,
    updatedAtNs: bigint,
    deletedAtNs: bigint,
  ): GraphResourceData {
    if (deletedAtNs > 0n) {
      const diffSeconds =
        Number((timestampNs - deletedAtNs) / 1_000_000n) / 1000;
      return {
        deletedAt: `${diffSeconds.toFixed(2)}s ago`,
      };
    }
    if (updatedAtNs > 0n) {
      const diffSeconds =
        Number((timestampNs - updatedAtNs) / 1_000_000n) / 1000;
      return {
        updatedAt: `${diffSeconds.toFixed(2)}s ago`,
      };
    }
    return {};
  }

  private sortPods(pods: PodGraphData[]): void {
    const deletionToScore = (p: PodGraphData): number => {
      return p.deletedAt ? 1 : 0;
    };
    const phaseToScore = (p: PodGraphData): number => {
      if (p.phase === 'Pending') return 0;
      if (p.phase === 'Completed') return 2;
      return 1;
    };
    pods.sort(
      (a, b) =>
        deletionToScore(a) - deletionToScore(b) ||
        phaseToScore(a) - phaseToScore(b),
    );
  }
}
