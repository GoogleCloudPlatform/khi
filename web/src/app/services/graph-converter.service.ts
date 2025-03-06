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

import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';
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
import { asBehaviorSubject } from '../utils/observable-util';
import { ResourceTimeline, TimelineLayer } from '../store/timeline';
import { ResourceRevision } from '../store/revision';

interface PodGraphDataGroupedByNode {
  [nodeName: string]: PodGraphData[];
}

interface ContainerGraphDataMap {
  [containerName: string]: ContainerGraphData;
}

interface MappedTimelineEntry {
  [label: string]: ResourceTimeline;
}

@Injectable({
  providedIn: 'root',
})
export class GraphDataConverterService {
  constructor(private _viewStateService: ViewStateService) {}

  private $timeZoneShift: BehaviorSubject<number> = asBehaviorSubject(
    this._viewStateService.timezoneShift,
    0,
  );

  public getGraphDataAt(timelines: ResourceTimeline[], t: number): GraphData {
    const mappedTimeline: MappedTimelineEntry = {};
    for (const timeline of timelines) {
      mappedTimeline[timeline.resourcePath] = timeline;
    }
    const nodes = this.getNodes(timelines);
    const podNames = this.getPodGraphData(mappedTimeline, t);
    this.sortPods(podNames);
    const nodeData = nodes
      .map((n) => this.getNodeGraphData(mappedTimeline, podNames, n, t))
      .filter((a) => a != null) as GraphNode[];
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
    const podOwners = {
      daemonset: this._parsePodOwnerGraphObjects(
        'daemonset',
        nodeData,
        mappedTimeline,
        t,
      ),
      job: this._parsePodOwnerGraphObjects('job', nodeData, mappedTimeline, t),
      replicaset: this._parsePodOwnerGraphObjects(
        'replicaset',
        nodeData,
        mappedTimeline,
        t,
      ),
    };
    return {
      nodes: nodeData,
      services: this.getServiceGraphData(nodeData, mappedTimeline, t),
      graphTime: LongTimestampFormatPipe.toLongDisplayTimestamp(
        t,
        this.$timeZoneShift.value,
      ),
      podOwners,
      podOwnerOwners: {
        cronjob: this._parsePodOwnerOwnerGraphObjects(
          'cronjob',
          podOwners.job,
          mappedTimeline,
          t,
        ),
        deployment: this._parsePodOwnerOwnerGraphObjects(
          'deployment',
          podOwners.replicaset,
          mappedTimeline,
          t,
        ),
      },
    };
  }

  private getServiceGraphData(
    nodes: GraphNode[],
    timeline: MappedTimelineEntry,
    t: number,
  ): ServiceGraphData[] {
    const services = Object.values(timeline).filter(
      (t) =>
        t.layer == TimelineLayer.Name &&
        t.getNameOfLayer(TimelineLayer.Kind) == 'service',
    );
    const result: ServiceGraphData[] = [];
    for (const serviceTimeline of services) {
      const manifest: k8s.K8sServiceResource = this._getManifest(
        timeline,
        serviceTimeline,
        t,
      ) as k8s.K8sServiceResource;
      if (!manifest) continue;

      const serviceName = serviceTimeline.getNameOfLayer(TimelineLayer.Name);
      const serviceNamespace = serviceTimeline.getNameOfLayer(
        TimelineLayer.Namespace,
      );

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
          timeline,
          serviceTimeline,
          graphServiceData,
        )
      ) {
        result.push(graphServiceData);
      }
    }
    return result;
  }

  private getNodes(timeline: ResourceTimeline[]): ResourceTimeline[] {
    return timeline.filter(
      (t) =>
        t.layer == TimelineLayer.Name &&
        t.getNameOfLayer(TimelineLayer.Kind) == 'node' &&
        t.getNameOfLayer(TimelineLayer.Namespace) == 'cluster-scope',
    );
  }

  private getNodeGraphData(
    timeline: MappedTimelineEntry,
    podNames: PodGraphDataGroupedByNode,
    nodeTimeline: ResourceTimeline,
    t: number,
  ): GraphNode | null {
    const nodeManifest: k8s.K8sNodeResource = this._getManifest(
      timeline,
      nodeTimeline,
      t,
    ) as k8s.K8sNodeResource;
    if (!nodeManifest) return null;

    const nodeName = nodeTimeline.getNameOfLayer(TimelineLayer.Name);
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
      this._checkDeletionThresholdAndUpdateTimestamp(
        t,
        timeline,
        nodeTimeline,
        result,
      )
    ) {
      return result;
    }
    return null;
  }

  private getPodGraphData(
    timeline: MappedTimelineEntry,
    t: number,
  ): PodGraphDataGroupedByNode {
    const result: PodGraphDataGroupedByNode = {};
    const podTimelines = Object.values(timeline).filter(
      (t) =>
        t.layer == TimelineLayer.Name &&
        t.getNameOfLayer(TimelineLayer.Kind) == 'pod',
    );
    for (const pd of podTimelines) {
      const manifest = this._getManifest(timeline, pd, t) as k8s.K8sPodResource;
      if (manifest != null) {
        this._parsePodInfo(timeline, t, pd, manifest, result);
      }
    }
    return result;
  }

  private _parsePodInfo(
    timelines: MappedTimelineEntry,
    t: number,
    podTimeline: ResourceTimeline,
    podManifest: k8s.K8sPodResource,
    dest: PodGraphDataGroupedByNode,
  ) {
    const podName = podTimeline.getNameOfLayer(TimelineLayer.Name);
    const podNamespace = podTimeline.getNameOfLayer(TimelineLayer.Namespace);

    const podSpec = podManifest.spec;
    if (!podSpec) return;

    let nodeName = podSpec.nodeName;
    if (!nodeName) {
      const bindingResource = timelines[`${podTimeline.resourcePath}#binding`];
      if (bindingResource) {
        const bindingManifest = bindingResource.getLatestRevisionOfTime(t);
        const bindingResourceManifest = bindingManifest?.parsedManifest;
        if (bindingResourceManifest) {
          nodeName = (
            bindingResourceManifest as unknown as k8s.K8sPodBindingResource
          ).target?.name;
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
        timelines,
        podTimeline,
        podGraphResource,
      )
    ) {
      dest[nodeName].push(podGraphResource);
    }
  }

  private _parsePodOwnerGraphObjects(
    kind: string,
    nodes: GraphNode[],
    timeline: MappedTimelineEntry,
    t: number,
  ): GraphPodOwner[] {
    const owners = Object.values(timeline).filter(
      (t) =>
        t.layer == TimelineLayer.Name &&
        t.getNameOfLayer(TimelineLayer.Kind) == kind,
    );
    const result: GraphPodOwner[] = [];
    for (const owner of owners) {
      const manifest = this._getManifest(
        timeline,
        owner,
        t,
      ) as k8s.K8sControlledResource;
      if (!manifest) continue;
      const uid = manifest.metadata?.uid;
      if (uid) {
        const ownerUids = new Set<string>();
        if (manifest.metadata?.ownerReferences) {
          for (const ownerReference of manifest.metadata.ownerReferences) {
            ownerUids.add(ownerReference.uid);
          }
        }
        const podOwnerGraphData: GraphPodOwner = {
          uid: uid,
          name: owner.getNameOfLayer(TimelineLayer.Name),
          namespace: owner.getNameOfLayer(TimelineLayer.Namespace),
          labels: manifest.metadata?.labels ?? {},
          connectedPods: this._getConnectedPodListFromOwnerUid(uid, nodes),
          status: manifest.status ?? {},
          ownerUids,
        };
        if (
          this._checkDeletionThresholdAndUpdateTimestamp(
            t,
            timeline,
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

  private _parsePodOwnerOwnerGraphObjects(
    kind: string,
    childGraphData: GraphPodOwner[],
    timeline: MappedTimelineEntry,
    t: number,
  ): GraphPodOwnerOwner[] {
    const ownerOwners = Object.values(timeline).filter(
      (t) =>
        t.layer == TimelineLayer.Name &&
        t.getNameOfLayer(TimelineLayer.Kind) == kind,
    );
    const result: GraphPodOwnerOwner[] = [];
    for (const owner of ownerOwners) {
      const manifest = this._getManifest(
        timeline,
        owner,
        t,
      ) as k8s.K8sControlledResource;
      if (!manifest) continue;
      const uid = manifest.metadata?.uid;
      if (uid) {
        const podOwner = childGraphData.filter((c) => c.ownerUids.has(uid));
        const podOwnerOwnerGraphData: GraphPodOwnerOwner = {
          uid: uid,
          name: owner.getNameOfLayer(TimelineLayer.Name),
          namespace: owner.getNameOfLayer(TimelineLayer.Namespace),
          labels: manifest.metadata?.labels ?? {},
          connectedPodOwners: podOwner.map((connectedPod) => ({
            podOwner: connectedPod,
          })),
          status: manifest.status ?? {},
        };
        if (
          this._checkDeletionThresholdAndUpdateTimestamp(
            t,
            timeline,
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
    status?: k8s.K8sStatus,
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

  private _getManifest(
    timelines: MappedTimelineEntry,
    targetEntry: ResourceTimeline,
    t: number,
  ): unknown {
    const resourceLevelRevision = targetEntry.getLatestRevisionOfTime(t);
    let statusLevelRevision: ResourceRevision | null = null;
    const statusLevelEntry = timelines[`${targetEntry.resourcePath}#status`];
    if (statusLevelEntry) {
      statusLevelRevision = statusLevelEntry.getLatestRevisionOfTime(t);
    }
    if (!resourceLevelRevision) {
      return statusLevelRevision?.parsedManifest ?? null;
    }
    let manifest: k8s.K8sControlledResource = {
      ...resourceLevelRevision.parsedManifest!,
    };
    // Override status field in the manifest when newer status subresource update found
    if (
      statusLevelRevision &&
      resourceLevelRevision.startAt < statusLevelRevision.startAt
    ) {
      const statusManifest =
        statusLevelRevision.parsedManifest as k8s.K8sControlledResource;
      if (statusManifest && statusManifest.status) {
        manifest = {
          ...manifest,
          status: statusManifest.status,
        };
      }
    }
    return manifest;
  }

  private _checkDeletionThresholdAndUpdateTimestamp(
    t: number,
    timelines: MappedTimelineEntry,
    timeline: ResourceTimeline,
    result: GraphResourceData,
  ): boolean {
    const deletionThreshold = 180;
    const revision = this._getRevisionLatestWithStatus(timelines, timeline, t);
    if (revision) {
      const diff = (t - revision.startAt) / 1000;
      if (revision.isDeletion) {
        if (diff <= deletionThreshold) {
          result.deletedAt = `${diff.toFixed(2)}s ago`;
        } else {
          return false;
        }
      }
      if (!revision.isDeletion) {
        if (revision.isInferred) {
          result.updatedAt = `more than ${diff.toFixed(2)}s ago`;
        } else {
          result.updatedAt = `${diff.toFixed(2)}s ago`;
        }
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

  private _getRevisionLatestWithStatus(
    timelines: MappedTimelineEntry,
    targetEntry: ResourceTimeline,
    t: number,
  ): ResourceRevision | null {
    const statusLevelEntry = timelines[`${targetEntry.resourcePath}#status`];
    const resourceLevelRevision = targetEntry.getLatestRevisionOfTime(t);
    if (statusLevelEntry) {
      const statusLevelRevision = statusLevelEntry.getLatestRevisionOfTime(t);
      if (!resourceLevelRevision) return statusLevelRevision;
      if (!statusLevelRevision) return resourceLevelRevision;
      return statusLevelRevision.startAt > resourceLevelRevision.startAt
        ? statusLevelRevision
        : resourceLevelRevision;
    }
    return resourceLevelRevision;
  }
}
