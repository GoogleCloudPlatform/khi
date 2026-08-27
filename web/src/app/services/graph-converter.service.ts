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
  ArchGraphCondition,
  ContainerGraphData,
  DEFAULT_DELETION_THRESHOLD_SECONDS,
  GraphData,
  GraphNode,
  PodGraphData,
  GraphPodOwner,
  GraphResourceData,
  ServiceGraphData,
  GraphPodOwnerOwner,
  PodConnectionGraphData,
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
export class GraphConverterService {
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
   * @param deletionThresholdSeconds - Optional deletion threshold in seconds (defaults to 180).
   * @param abortSignal - Optional signal to cancel the generation.
   * @returns A promise resolving to the converted GraphData.
   */
  public async getGraphDataAt(
    timestampNs: bigint,
    timelineBitset?: SparseBitset,
    deletionThresholdSeconds = DEFAULT_DELETION_THRESHOLD_SECONDS,
    abortSignal?: AbortSignal,
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
      const timestamps = this.formatResourceTimestamps(timestampNs, n);
      const node: GraphNode = {
        name: n.name,
        podCIDR: n.podCidr,
        taints: n.taints,
        internalIP: n.internalIp,
        externalIP: n.externalIp,
        labels: n.labels,
        pods: [],
        conditions: this.convertConditions(n.conditions),
        ...timestamps,
      };
      nodeByName.set(n.name, node);
    }

    const podMapById = new Map<string, PodConnectionGraphData>();

    for (const p of res.pods) {
      const parentNode = nodeByName.get(p.nodeName);
      if (!parentNode) {
        continue;
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

      const timestamps = this.formatResourceTimestamps(timestampNs, p);
      const pod: PodGraphData = {
        uid: p.uid,
        name: p.name,
        namespace: p.namespace,
        labels: p.labels,
        containers,
        podIP: p.podIp,
        phase: p.phase,
        isPhaseHealthy: p.isPhaseHealthy,
        conditions: this.convertConditions(p.conditions),
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
      const timestamps = this.formatResourceTimestamps(timestampNs, s);
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
      const timestamps = this.formatResourceTimestamps(timestampNs, po);
      const owner: GraphPodOwner = {
        uid: po.uid,
        name: po.name,
        namespace: po.namespace,
        labels: po.labels,
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

    for (const ownerOwner of res.podOwnerOwners) {
      const timestamps = this.formatResourceTimestamps(timestampNs, ownerOwner);
      const owner: GraphPodOwnerOwner = {
        uid: ownerOwner.uid,
        name: ownerOwner.name,
        namespace: ownerOwner.namespace,
        labels: ownerOwner.labels,
        connectedPodOwners: [],
        ...timestamps,
      };
      podOwnerOwnerMapById.set(ownerOwner.id, owner);
      const kind = ownerOwner.kind.toLowerCase() as PodOwnerOwnerKinds;
      if (kind === 'cronjob' || kind === 'deployment') {
        podOwnerOwners[kind].push(owner);
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
          const topOwner = podOwnerOwnerMapById.get(edge.sourceId);
          const po = podOwnerMapById.get(edge.targetId);
          if (topOwner && po) {
            topOwner.connectedPodOwners.push({ podOwner: po });
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

  private convertConditions(
    conditions: readonly GraphCondition[],
  ): ArchGraphCondition[] {
    return conditions.map((c) => ({
      type: c.type,
      message: c.message,
      status: c.status,
      is_positive_status: c.isPositive,
    }));
  }

  private formatResourceTimestamps(
    timestampNs: bigint,
    resource: { readonly updatedAtNs: bigint; readonly deletedAtNs: bigint },
  ): GraphResourceData {
    if (resource.deletedAtNs > 0n) {
      const diffSeconds =
        Number((timestampNs - resource.deletedAtNs) / 1_000_000n) / 1000;
      return {
        deletedAt: `${diffSeconds.toFixed(2)}s ago`,
      };
    }
    if (resource.updatedAtNs > 0n) {
      const diffSeconds =
        Number((timestampNs - resource.updatedAtNs) / 1_000_000n) / 1000;
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
