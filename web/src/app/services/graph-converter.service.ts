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
  GraphData,
  GraphNode,
  PodGraphData,
  GraphPodOwner,
  GraphResourceData,
  ServiceGraphData,
  PodConnectionGraphData,
  GraphPodOwnerOwner,
} from '../common/schema/graph-schema';
import * as k8s from '../store/k8s-types';
import { isConditionPositive } from '../store/condition-positive-map';
import { LongTimestampFormatPipe } from '../common/timestamp-format.pipe';
import { ViewStateService } from '../services/view-state.service';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { toSignal } from '@angular/core/rxjs-interop';
import { TaskYielder } from 'src/app/utils/task-yielder';

interface PodGraphDataGroupedByNode {
  [nodeName: string]: PodGraphData[];
}

interface ContainerGraphDataMap {
  [containerName: string]: ContainerGraphData;
}

@Injectable({
  providedIn: 'root',
})
export class GraphDataConverterService {
  private readonly _viewStateService = inject(ViewStateService);

  private readonly timezoneShift = toSignal(
    this._viewStateService.timezoneShift,
  );

  /**
   * Generates graph data at the specified timestamp asynchronously, yielding execution to prevent UI freezing.
   *
   * @param timelines - Array of timelines to inspect.
   * @param t - Timestamp in nanoseconds.
   * @param abortSignal - Optional signal to cancel the generation.
   * @param maxProcessingTimeMs - Maximum execution time per chunk in milliseconds.
   * @returns A promise resolving to the converted GraphData.
   */
  public async getGraphDataAt(
    timelines: ReadonlyDomainElement<Timeline>[],
    t: bigint,
    abortSignal?: AbortSignal,
    maxProcessingTimeMs = 16,
  ): Promise<GraphData> {
    const taskYielder = new TaskYielder(maxProcessingTimeMs, abortSignal);

    await taskYielder.yield();

    const nodes = this.getNodes(timelines);
    const podNames = await this.getPodGraphData(timelines, t, taskYielder);
    this.sortPods(podNames);

    await taskYielder.yield();

    const nodeData: GraphNode[] = [];
    for (const n of nodes) {
      await taskYielder.yield();
      const data = this.getNodeGraphData(podNames, n, t);
      if (data != null) {
        nodeData.push(data);
      }
    }
    const foundNodeNames = new Set(nodeData.map((n) => n.name));

    // Add nodes not observed in node audit logs but observed in pod manifest
    for (const key in podNames) {
      if (foundNodeNames.has(key)) continue;
      nodeData.push({
        name: key,
        podCIDR: '-',
        taints: [],
        pods: podNames[key] ?? [],
        labels: {},
        conditions: [],
        internalIP: '-',
        externalIP: '-',
      });
    }

    const daemonsetOwners = await this._parsePodOwnerGraphObjects(
      'daemonset',
      nodeData,
      timelines,
      t,
      taskYielder,
    );
    const jobOwners = await this._parsePodOwnerGraphObjects(
      'job',
      nodeData,
      timelines,
      t,
      taskYielder,
    );
    const replicasetOwners = await this._parsePodOwnerGraphObjects(
      'replicaset',
      nodeData,
      timelines,
      t,
      taskYielder,
    );
    const podOwners = {
      daemonset: daemonsetOwners,
      job: jobOwners,
      replicaset: replicasetOwners,
    };

    const services = await this.getServiceGraphData(
      nodeData,
      timelines,
      t,
      taskYielder,
    );

    const cronjobOwnerOwners = await this._parsePodOwnerOwnerGraphObjects(
      'cronjob',
      podOwners.job,
      timelines,
      t,
      taskYielder,
    );
    const deploymentOwnerOwners = await this._parsePodOwnerOwnerGraphObjects(
      'deployment',
      podOwners.replicaset,
      timelines,
      t,
      taskYielder,
    );

    await taskYielder.yield();

    return {
      nodes: nodeData,
      services,
      graphTime: LongTimestampFormatPipe.toLongDisplayTimestamp(
        Number(t / 1_000_000n),
        this.timezoneShift() ?? 0,
      ),
      podOwners,
      podOwnerOwners: {
        cronjob: cronjobOwnerOwners,
        deployment: deploymentOwnerOwners,
      },
    };
  }

  private async getServiceGraphData(
    nodes: GraphNode[],
    timelines: ReadonlyDomainElement<Timeline>[],
    t: bigint,
    taskYielder: TaskYielder,
  ): Promise<ServiceGraphData[]> {
    const services = timelines.filter((serviceTimeline) => {
      const path = serviceTimeline.path;
      return path.length === 5 && path[2].label === 'service';
    });
    const result: ServiceGraphData[] = [];
    for (const serviceTimeline of services) {
      await taskYielder.yield();
      const rev = serviceTimeline.lookupRevisionAtNs(t, false);
      if (!rev) continue;
      const manifest =
        rev.body as ReadonlyDomainElement<k8s.K8sServiceResource>;
      if (!manifest) continue;

      const path = serviceTimeline.path;
      const serviceName = path[4].label;
      const serviceNamespace = path[3].label;

      const selector = manifest.spec?.selector ?? {};
      const connectedPods: PodConnectionGraphData[] = [];
      if (selector) {
        for (const node of nodes) {
          for (const pod of node.pods) {
            let match = Object.keys(selector).length != 0;
            for (const key in selector) {
              if (
                !(key in pod.labels) ||
                (key in pod.labels && pod.labels[key] != selector[key])
              ) {
                match = false;
              }
            }
            if (match) {
              connectedPods.push({
                node: node,
                pod: pod,
              });
            }
          }
        }
      }

      const graphServiceData: ServiceGraphData = {
        uid: manifest.metadata?.uid,
        name: serviceName,
        namespace: serviceNamespace,
        labels: {},
        clusterIp: manifest.status?.clusterIp ?? '-',
        type: manifest.spec?.type ?? 'Unknown',
        connectedPods,
      };

      if (
        this._checkDeletionThresholdAndUpdateTimestamp(
          t,
          serviceTimeline,
          graphServiceData,
        )
      ) {
        result.push(graphServiceData);
      }
    }
    return result;
  }

  private getNodes(
    timeline: ReadonlyDomainElement<Timeline>[],
  ): ReadonlyDomainElement<Timeline>[] {
    return timeline.filter((t) => {
      const path = t.path;
      // Cluster -> API version -> Kind -> Namespace -> Name
      return (
        path.length === 5 &&
        path[2].label === 'node' &&
        path[3].label === 'cluster-scope'
      );
    });
  }

  private getNodeGraphData(
    podNames: PodGraphDataGroupedByNode,
    nodeTimeline: ReadonlyDomainElement<Timeline>,
    t: bigint,
  ): GraphNode | null {
    const rev = nodeTimeline.lookupRevisionAtNs(t, false);
    if (!rev) return null;
    const nodeManifest = rev.body as ReadonlyDomainElement<k8s.K8sNodeResource>;
    if (!nodeManifest) return null;

    const path = nodeTimeline.path;

    const nodeName = path[4].label;
    const result: GraphNode = {
      name: nodeName,
      labels: {},
      pods: podNames[nodeName] ?? [],
      internalIP: '-',
      externalIP: '-',
      podCIDR: '-',
      taints: [],
      conditions: [],
    };

    result.podCIDR = nodeManifest.spec?.podCIDR ?? '-';
    result.taints =
      nodeManifest.spec?.taints?.map((t) => `${t.key}(${t.effect})`) ?? [];
    result.conditions = this._parseConditions('node', nodeManifest.status);

    if (nodeManifest.status && nodeManifest.status.addresses) {
      for (const addressTuple of nodeManifest.status.addresses) {
        if (addressTuple.type == 'InternalIP') {
          result.internalIP = addressTuple.address;
        }
        if (addressTuple.type == 'ExternalIP') {
          result.externalIP = addressTuple.address;
        }
      }
    }

    if (
      this._checkDeletionThresholdAndUpdateTimestamp(t, nodeTimeline, result)
    ) {
      return result;
    }
    return null;
  }

  private async getPodGraphData(
    timeline: ReadonlyDomainElement<Timeline>[],
    t: bigint,
    taskYielder: TaskYielder,
  ): Promise<PodGraphDataGroupedByNode> {
    const result: PodGraphDataGroupedByNode = {};
    const podTimelines = timeline.filter((t) => {
      const path = t.path;
      // Cluster -> API version -> Kind -> Namespace -> Name
      return path.length === 5 && path[2].label === 'pod';
    });
    for (const pd of podTimelines) {
      await taskYielder.yield();
      const rev = pd.lookupRevisionAtNs(t, false);
      if (!rev) continue;
      const manifest = rev.body as ReadonlyDomainElement<k8s.K8sPodResource>;
      if (manifest != null) {
        this._parsePodInfo(t, pd, manifest, result);
      }
    }
    return result;
  }

  private _parsePodInfo(
    t: bigint,
    podTimeline: ReadonlyDomainElement<Timeline>,
    podManifest: ReadonlyDomainElement<k8s.K8sPodResource>,
    dest: PodGraphDataGroupedByNode,
  ) {
    const podPath = podTimeline.path;
    const podName = podPath[4].label;
    const podNamespace = podPath[3].label;

    const podSpec = podManifest.spec;
    if (!podSpec) return;

    let nodeName = podSpec.nodeName;
    if (!nodeName) {
      for (const podChild of podTimeline.children()) {
        if (podChild.name == 'binding') {
          const bindingRev = podChild.lookupRevisionAtNs(t, false);
          if (bindingRev) {
            const bindingManifest =
              bindingRev.body as ReadonlyDomainElement<k8s.K8sPodBindingResource>;
            if (bindingManifest) {
              nodeName = bindingManifest.target?.name;
            }
          }
        }
      }
      if (!nodeName) {
        return;
      }
    }
    if (!(nodeName in dest)) dest[nodeName] = [];

    const containerGraphData: ContainerGraphDataMap = {};

    if (podSpec.initContainers) {
      for (const container of podSpec.initContainers) {
        containerGraphData[container.name] = {
          name: container.name,
          status: 'Unknown',
          isInitContainer: true,
          isStatusHealthy: false,
          ready: false,
          code: 0,
          reason: 'Unknown',
          statusReadFromManifest: false,
        };
      }
    }
    if (podSpec.containers) {
      for (const container of podSpec.containers) {
        containerGraphData[container.name] = {
          name: container.name,
          status: 'Unknown',
          isInitContainer: false,
          isStatusHealthy: false,
          ready: false,
          code: 0,
          reason: 'Unknown',
          statusReadFromManifest: false,
        };
      }
    }

    const podStatus = podManifest.status;
    if (podStatus) {
      if (podStatus.initContainerStatuses) {
        for (const containerStatus of podStatus.initContainerStatuses) {
          containerGraphData[containerStatus.name].statusReadFromManifest =
            true;
          containerGraphData[containerStatus.name].ready =
            containerStatus.ready;
          this._convertContainerStatusStateToString(
            containerStatus.state,
            containerGraphData[containerStatus.name],
          );
        }
      }

      if (podStatus.containerStatuses) {
        for (const containerStatus of podStatus.containerStatuses) {
          containerGraphData[containerStatus.name].statusReadFromManifest =
            true;
          containerGraphData[containerStatus.name].ready =
            containerStatus.ready;
          this._convertContainerStatusStateToString(
            containerStatus.state,
            containerGraphData[containerStatus.name],
          );
        }
      }
    }

    const podPhase = podManifest.status?.phase ?? 'Unknown';
    const ownerUids = new Set<string>();
    if (podManifest.metadata?.ownerReferences) {
      for (const owner of podManifest.metadata.ownerReferences) {
        ownerUids.add(owner.uid);
      }
    }

    const podGraphResource: PodGraphData = {
      uid: podManifest.metadata?.uid,
      name: podName,
      namespace: podNamespace,
      labels: podManifest.metadata?.labels ?? {},
      containers: Object.values(containerGraphData),
      podIP: podManifest.status?.podIP ?? '-',
      phase: podPhase,
      isPhaseHealthy: podPhase == 'Running' || podPhase == 'Completed',
      conditions: this._parseConditions('pod', podManifest.status),
      ownerUids,
    };

    if (
      this._checkDeletionThresholdAndUpdateTimestamp(
        t,
        podTimeline,
        podGraphResource,
      )
    ) {
      dest[nodeName].push(podGraphResource);
    }
  }

  private async _parsePodOwnerGraphObjects(
    kind: string,
    nodes: GraphNode[],
    timelines: ReadonlyDomainElement<Timeline>[],
    t: bigint,
    taskYielder: TaskYielder,
  ): Promise<GraphPodOwner[]> {
    const owners = timelines.filter((t) => {
      const path = t.path;
      return path.length === 5 && path[2].label === kind;
    });
    const result: GraphPodOwner[] = [];
    for (const owner of owners) {
      await taskYielder.yield();
      const rev = owner.lookupRevisionAtNs(t, false);
      if (!rev) continue;
      const manifest =
        rev.body as ReadonlyDomainElement<k8s.K8sControlledResource>;
      if (!manifest) continue;
      const uid = manifest.metadata?.uid;
      if (uid) {
        const ownerUids = new Set<string>();
        if (manifest.metadata?.ownerReferences) {
          for (const ownerReference of manifest.metadata.ownerReferences) {
            ownerUids.add(ownerReference.uid);
          }
        }
        const path = owner.path;

        const podOwnerGraphData: GraphPodOwner = {
          uid: uid,
          name: path[4].label,
          namespace: path[3].label,
          labels: manifest.metadata?.labels ?? {},
          connectedPods: this._getConnectedPodListFromOwnerUid(uid, nodes),
          status: manifest.status ?? {},
          ownerUids,
        };
        if (
          this._checkDeletionThresholdAndUpdateTimestamp(
            t,
            owner,
            podOwnerGraphData,
          )
        ) {
          result.push(podOwnerGraphData);
        }
      }
    }
    return result;
  }

  private _getConnectedPodListFromOwnerUid(
    uid: string,
    nodes: GraphNode[],
  ): PodConnectionGraphData[] {
    const result = [] as PodConnectionGraphData[];
    for (const node of nodes) {
      for (const pod of node.pods) {
        if (pod.ownerUids.has(uid)) {
          result.push({
            node,
            pod,
          });
        }
      }
    }
    return result;
  }

  private async _parsePodOwnerOwnerGraphObjects(
    kind: string,
    childGraphData: GraphPodOwner[],
    timelines: ReadonlyDomainElement<Timeline>[],
    t: bigint,
    taskYielder: TaskYielder,
  ): Promise<GraphPodOwnerOwner[]> {
    const ownerOwners = timelines.filter((ownerTimeline) => {
      const path = ownerTimeline.path;
      return path.length === 5 && path[2].label === kind;
    });
    const result: GraphPodOwnerOwner[] = [];
    for (const owner of ownerOwners) {
      await taskYielder.yield();
      const rev = owner.lookupRevisionAtNs(t, false);
      if (!rev) continue;
      const manifest =
        rev.body as ReadonlyDomainElement<k8s.K8sControlledResource>;
      if (!manifest) continue;
      const uid = manifest.metadata?.uid;
      if (uid) {
        const path = owner.path;
        const podOwner = childGraphData.filter((c) => c.ownerUids.has(uid));
        const podOwnerOwnerGraphData: GraphPodOwnerOwner = {
          uid: uid,
          name: path[4].label,
          namespace: path[3].label,
          labels: manifest.metadata?.labels ?? {},
          connectedPodOwners: podOwner.map((connectedPod) => ({
            podOwner: connectedPod,
          })),
          status: manifest.status ?? {},
        };
        if (
          this._checkDeletionThresholdAndUpdateTimestamp(
            t,
            owner,
            podOwnerOwnerGraphData,
          )
        ) {
          result.push(podOwnerOwnerGraphData);
        }
      }
    }

    return result;
  }

  private _convertContainerStatusStateToString(
    status: k8s.ContainerStatusState,
    dest: ContainerGraphData,
  ) {
    if (status.running) {
      dest.status = 'Running';
      dest.isStatusHealthy = true;
    }

    if (status.terminated) {
      if (status.terminated.reason == 'Completed') {
        dest.isStatusHealthy = true;
      }
      dest.status = `${status.terminated.reason}`;
    }
  }

  private _parseConditions(
    resourceType: 'pod' | 'node',
    status?: ReadonlyDomainElement<k8s.K8sStatus>,
  ): ArchGraphCondition[] {
    if (!status || !status.conditions) return [];
    return status.conditions.map((condition) => ({
      type: condition.type,
      message: condition.message,
      status: condition.status,
      is_positive_status: isConditionPositive(
        resourceType,
        condition.type,
        condition.status,
      ),
    }));
  }

  private _checkDeletionThresholdAndUpdateTimestamp(
    t: bigint,
    timeline: ReadonlyDomainElement<Timeline>,
    result: GraphResourceData,
  ): boolean {
    const deletionThreshold = 180;
    const revision = timeline.lookupRevisionAtNs(t, false);
    if (revision) {
      const diff = Number((t - revision.changedTime) / 1_000_000n) / 1000;
      if (
        revision.verb.label === 'Delete' ||
        revision.verb.label === 'DeleteCollection'
      ) {
        if (diff <= deletionThreshold) {
          result.deletedAt = `${diff.toFixed(2)}s ago`;
        } else {
          return false;
        }
      } else {
        result.updatedAt = `${diff.toFixed(2)}s ago`;
      }
    }
    return true;
  }

  private sortPods(dest: PodGraphDataGroupedByNode) {
    const deletionToScore: (p: PodGraphData) => number = (p) => {
      return p.deletedAt ? 1 : 0;
    };
    const phaseToScore: (p: PodGraphData) => number = (p) => {
      if (p.phase == 'Pending') return 0;
      if (p.phase == 'Completed') return 2;
      return 1;
    };
    for (const key in dest) {
      const podList = dest[key];
      dest[key] = podList.sort(
        (a, b) =>
          deletionToScore(a) - deletionToScore(b) ||
          phaseToScore(a) - phaseToScore(b),
      );
    }
  }
}
