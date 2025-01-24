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

import {
  GraphData,
  PodOwnerKinds,
  PodOwnerOwnerKinds,
} from '../../../../common/schema/graph-schema';
import { Direction, AnchorPoints, GraphObject } from './base/base-containers';
import {
  $alignedGroup,
  $circle,
  $empty,
  $label,
  $pathPipe,
  $pattern,
  $rect,
} from './builder-alias';
import { GraphPattern } from './base/defs-child-elements';
import { GraphRoot } from './graph-root';
import { generateNodeBox } from './components/workload';
import { generateServiceBox } from './components/service';
import {
  GRAPH_DESCRIPTION_BACKGROUND,
  GRAPH_DESCRIPTION_LABEL,
  GRAPH_WARNING_BACKGROUND,
  GRAPH_WARNING_LABEL,
  TIMESTAMP_LABEL,
} from './styles';
import { PathPipe } from './base/path';
import { generatePodOwnerRow } from './components/pod-owner';

export class GraphRenderer {
  private static TITLE_DESCRIPTION = `Provided by Kubernetes History Inspector(${process.env['NG_APP_VERSION']}).`;

  private static TITLE_WARNING = `This graph only contains the resource observed in the query duration specified. The other resources may exist during the time when the resource were not touched at all.`;

  private _rootElement: GraphRoot;

  public get root(): GraphRoot {
    return this._rootElement;
  }

  constructor(public readonly containerHTMLElement: HTMLElement) {
    this._rootElement = new GraphRoot();
    this._rootElement.attach(containerHTMLElement);
  }

  public updateGraphData(graphData: GraphData): void {
    this._rootElement.clearChildren();
    this._rootElement
      .withDefs({
        ...this.generatePatternDefs(),
      })
      .withChildren([
        this.generateBackground(),
        $alignedGroup(Direction.Vertical)
          .withId('element-root')
          .withGap(20)
          .withChildren([
            ...this.generateTopMessages(graphData),
            generatePodOwnerRow(graphData, graphData.podOwnerOwners),
            $pathPipe(Direction.Horizontal, `pod-owner-owner-pipe`),
            generatePodOwnerRow(graphData, graphData.podOwners),
            $pathPipe(Direction.Horizontal, `pod-owner-pipe`),
            $alignedGroup(Direction.Horizontal)
              .withGap(5)
              .withChildren(graphData.nodes.map((n) => generateNodeBox(n))),
            $pathPipe(Direction.Horizontal, `node-service-pipe`),
            $empty().withChildren([
              $empty()
                .withAnchor(AnchorPoints.CENTER)
                .withPivot(AnchorPoints.CENTER)
                .withChildren([
                  $alignedGroup(Direction.Horizontal)
                    .withMargin(50, 0, 0, 0)
                    .withGap(5)
                    .withChildren(
                      graphData.services.map((s) => generateServiceBox(s)),
                    ),
                ]),
            ]),
          ]),
      ])
      .registerLayoutStep(0, () => {
        const nodeServicePipe = this.root.find(`node-service-pipe`) as PathPipe;
        const podOwnerPipe = this.root.find(`pod-owner-pipe`) as PathPipe;
        const podOwnerOwnerPipe = this.root.find(
          `pod-owner-owner-pipe`,
        ) as PathPipe;
        for (const node of graphData.nodes) {
          const nodeLeftPipe = this.root.find(
            `node_${node.name}_l`,
          )! as PathPipe;
          const nodeRightPipe = this.root.find(
            `node_${node.name}_r`,
          )! as PathPipe;
          nodeServicePipe.connectPipe(nodeLeftPipe);
          podOwnerPipe.connectPipe(nodeRightPipe);
          for (const pod of node.pods) {
            const podRect = this.root.find(`pod_${pod.namespace}_${pod.name}`)!;
            nodeLeftPipe.connectPoint(podRect, AnchorPoints.CENTER_LEFT);
            nodeRightPipe.connectPoint(podRect, AnchorPoints.TOP_RIGHT);
          }
        }
        for (const service of graphData.services) {
          const serviceRect = this.root.find(
            `service_${service.namespace}_${service.name}`,
          )!;
          nodeServicePipe.connectPoint(serviceRect, AnchorPoints.TOP);
        }
        for (const kind in graphData.podOwners) {
          for (const podOwner of graphData.podOwners[kind as PodOwnerKinds]) {
            const podOwnerId = `${kind}_${podOwner.namespace}_${podOwner.name}`;
            const podOwnerGraphObject = this.root.find(podOwnerId)!;
            podOwnerOwnerPipe.connectPoint(
              podOwnerGraphObject,
              AnchorPoints.TOP,
            );
          }
        }
        for (const kind in graphData.podOwnerOwners) {
          for (const podOwnerOwner of graphData.podOwnerOwners[
            kind as PodOwnerOwnerKinds
          ]) {
            const podOwnerOwnerId = `${kind}_${podOwnerOwner.namespace}_${podOwnerOwner.name}`;
            const podOwnerOwnerGraphObject = this.root.find(podOwnerOwnerId)!;
            podOwnerOwnerPipe.connectPoint(
              podOwnerOwnerGraphObject,
              AnchorPoints.BOTTOM,
            );
          }
        }
        for (const service of graphData.services) {
          for (const connection of service.connectedPods) {
            const serviceId = `service_${service.namespace}_${service.name}`;
            const nodeId = `node_${connection.node.name}_l`;
            const podId = `pod_${connection.pod.namespace}_${connection.pod.name}`;
            const path = `${serviceId}/node-service-pipe/${nodeId}/${podId}`;
            nodeServicePipe.registerPath(path, 'arrow', 15, 90);
          }
        }

        for (const kind in graphData.podOwners) {
          for (const owner of graphData.podOwners[kind as PodOwnerKinds]) {
            const ownerName = `${kind}_${owner.namespace}_${owner.name}`;
            const ownerBox = this.root.find(ownerName)!;
            podOwnerPipe.connectPoint(ownerBox, AnchorPoints.BOTTOM);
          }
        }

        for (const kind in graphData.podOwners) {
          for (const owner of graphData.podOwners[kind as PodOwnerKinds]) {
            const ownerId = `${kind}_${owner.namespace}_${owner.name}`;
            for (const connection of owner.connectedPods) {
              const nodeId = `node_${connection.node.name}_r`;
              const podId = `pod_${connection.pod.namespace}_${connection.pod.name}`;
              const path = `${ownerId}/pod-owner-pipe/${nodeId}/${podId}`;
              podOwnerPipe.registerPath(path, 'circle', 8, 90, {
                'stroke-dasharray': '3 3',
              });
            }
          }
        }

        const expectedConnectedKind: {
          [kind in PodOwnerOwnerKinds]: PodOwnerKinds;
        } = {
          cronjob: 'job',
          deployment: 'replicaset',
        };
        for (const kind in graphData.podOwnerOwners) {
          for (const podOwnerOwner of graphData.podOwnerOwners[
            kind as PodOwnerOwnerKinds
          ]) {
            const podOwnerOwnerId = `${kind}_${podOwnerOwner.namespace}_${podOwnerOwner.name}`;
            for (const podOwner of podOwnerOwner.connectedPodOwners) {
              const podOwnerId = `${
                expectedConnectedKind[kind as PodOwnerOwnerKinds]
              }_${podOwner.podOwner.namespace}_${podOwner.podOwner.name}`;
              const path = `${podOwnerOwnerId}/pod-owner-owner-pipe/${podOwnerId}`;
              podOwnerOwnerPipe.registerPath(path, 'circle', 8, 0, {
                'stroke-dasharray': '3 3',
              });
            }
          }
        }
      })
      .render();
  }

  private generatePatternDefs(): { [key: string]: GraphPattern } {
    return {
      background: $pattern(20, 20).withChildren([
        $rect()
          .withMinSize(20, 20)
          .withStyle({ fill: 'white' })
          .withChildren([
            $circle()
              .withPivot(AnchorPoints.CENTER)
              .withAnchor(AnchorPoints.CENTER)
              .withStyle({
                r: 1.5,
                fill: 'black',
                opacity: 0.3,
              }),
          ]),
      ]),
    };
  }

  private generateBackground(): GraphObject {
    const MAX_SIZE = 1000000;
    return $rect()
      .withStyle({
        fill: 'url(#background)',
      })
      .withMinSize(MAX_SIZE, MAX_SIZE)
      .withMargin(-MAX_SIZE / 2, 0, 0, -MAX_SIZE / 2)
      .withIgnoredFromParentSizing();
  }

  private generateTopMessages(graphData: GraphData): GraphObject[] {
    if (graphData.graphTime == '-')
      return [this.generateNoDataSelectedMessage()];
    return [
      $empty().withChildren([
        $alignedGroup(Direction.Horizontal)
          .withGap(50)
          .withChildren([
            $label('@' + graphData.graphTime).withStyle(TIMESTAMP_LABEL),
            $empty().withChildren([
              $rect()
                .withChildren([
                  $label(GraphRenderer.TITLE_DESCRIPTION)
                    .withStyle(GRAPH_DESCRIPTION_LABEL)
                    .withMargin(3, 10, 3, 10),
                ])
                .withStyle(GRAPH_DESCRIPTION_BACKGROUND)
                .withAnchor(AnchorPoints.BOTTOM)
                .withPivot(AnchorPoints.BOTTOM),
            ]),
            $empty().withChildren([
              $rect()
                .withChildren([
                  $label(GraphRenderer.TITLE_WARNING)
                    .withStyle(GRAPH_WARNING_LABEL)
                    .withMargin(3, 10, 3, 10),
                ])
                .withStyle(GRAPH_WARNING_BACKGROUND)
                .withAnchor(AnchorPoints.BOTTOM)
                .withPivot(AnchorPoints.BOTTOM),
            ]),
          ]),
      ]),
    ];
  }

  public getSVGForDownload(): SVGElement | null {
    const elementRoot = this.root.find('element-root');
    if (!elementRoot) return null;
    const margin = 20;
    const graphSize = elementRoot.transform.contentSize;
    const copiedNode = this.root.element?.cloneNode(true) as SVGElement;
    const svgWidth = margin * 2 + graphSize.width;
    const svgHeight = margin * 2 + graphSize.height;
    copiedNode.setAttribute('width', '' + svgWidth);
    copiedNode.setAttribute('height', '' + svgHeight);
    copiedNode.setAttribute(
      'viewBox',
      `${-margin},${-margin},${svgWidth},${svgHeight}`,
    );
    return copiedNode;
  }

  public generateNoDataSelectedMessage(): GraphObject {
    const labelStyle = { fill: 'white', 'font-weight': 500, 'font-size': 40 };
    return $rect()
      .withStyle({
        fill: 'orange',
        'stroke-width': '3',
      })
      .withChildren([
        $alignedGroup(Direction.Vertical)
          .withChildren([
            $label('Rendering...').withStyle(labelStyle),
            $label('This step could take few seconds').withStyle(labelStyle),
          ])
          .withMargin(10, 10, 10, 10),
      ]);
  }
}
