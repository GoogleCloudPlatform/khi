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
  ContainerGraphData,
  GraphNode,
  PodGraphData,
} from '../../../../../common/schema/graph-schema';
import {
  AnchorPoints,
  Direction,
  ElementStyle,
  GraphObject,
} from '../base/base-containers';
import {
  $alignedBox,
  $alignedGroup,
  $boxed_label,
  $empty,
  $hpair,
  $label,
  $pathPipe,
  $rect,
} from '../builder-alias';
import {
  CONTAINER_METADATA_LABEL,
  CONTAINER_STYLE,
  CONTAINER_TYPE_LABEL,
  GRAPH_COLORS,
  INIT_CONTAINER_LABEL_BACKGROUND,
  METADATA_BOX_ERROR,
  METADATA_BOX_WARNING,
  NAMESPACE_LABEL_BACKGROUND,
  NODE_KIND_LABEL,
  NODE_METADATA_BOX,
  NODE_METADATA_LABEL_NORMAL,
  NODE_NAME_LABEL,
  NODE_STYLE,
  POD_KIND_LABEL,
  POD_METADATA_BOX,
  POD_METADATA_LABEL_NORMAL,
  POD_STYLE,
  TRANSPARENT_BOX,
} from '../styles';
import * as common from './common';
export function generateNodeBox(nodeData: GraphNode): GraphObject {
  return $alignedGroup(Direction.Horizontal)
    .withGap(10)
    .withChildren([
      $pathPipe(Direction.Vertical, `node_${nodeData.name}_l`),
      $rect()
        .withStyle(NODE_STYLE)
        .withStyle(common.generateDeletedResourceStyle(nodeData))
        .withChildren([
          $alignedGroup(Direction.Horizontal).withChildren([
            $empty().withChildren([
              common.generateKindLabel(
                'Node',
                GRAPH_COLORS.NODE,
                NODE_KIND_LABEL,
              ),
              $alignedGroup(Direction.Vertical)
                .withMargin(35, 20, 5, 20)
                .withChildren([
                  $empty()
                    .withAnchor(AnchorPoints.TOP)
                    .withPivot(AnchorPoints.TOP)
                    .withChildren([
                      $label(nodeData.name).withStyle(NODE_NAME_LABEL),
                    ]),
                  generatePodElements(nodeData),
                ]),
            ]),
            $pathPipe(Direction.Vertical, `node_${nodeData.name}_r`),
            generateNodeMetadataBox(nodeData),
          ]),
        ]),
    ]);
}

export function generateNodeMetadataBox(node: GraphNode): GraphObject {
  const generateTaintItems = () => {
    if (node.taints.length == 0) {
      return [$label('No taints').withStyle(NODE_METADATA_LABEL_NORMAL)];
    }
    return [
      $label('Node taints:').withStyle(NODE_METADATA_LABEL_NORMAL),
      ...node.taints.map((t) =>
        $rect()
          .withStyle(METADATA_BOX_WARNING)
          .withChildren([
            $label(t)
              .withStyle(NODE_METADATA_LABEL_NORMAL)
              .withMargin(2, 5, 2, 5),
          ]),
      ),
    ];
  };

  const generateConditionItems = () => {
    return [
      $label('Node conditions:').withStyle(NODE_METADATA_LABEL_NORMAL),
      ...node.conditions.map((c) =>
        $rect()
          .withStyle(
            c.is_positive_status ? TRANSPARENT_BOX : METADATA_BOX_ERROR,
          )
          .withChildren([
            $label(`${c.type} : ${c.status}`)
              .withStyle(NODE_METADATA_LABEL_NORMAL)
              .withMargin(2, 5, 2, 5),
          ]),
      ),
    ];
  };

  return $empty()
    .withMinSize(0, 350)
    .withChildren([
      ...common.generateDeletionOrUpdateLabel(node, GRAPH_COLORS.NODE, 20),
      $alignedBox(
        Direction.Vertical,
        TRANSPARENT_BOX,
        5,
        [0, 0, 0, 0],
        [
          $alignedBox(
            Direction.Vertical,
            NODE_METADATA_BOX,
            0,
            [3, 5, 3, 5],
            [
              $label(`PodCIDR:${node.podCIDR}`).withStyle(
                NODE_METADATA_LABEL_NORMAL,
              ),
              $label(`InternalIP:${node.internalIP}`).withStyle(
                NODE_METADATA_LABEL_NORMAL,
              ),
              $label(`ExternalIP:${node.externalIP}`).withStyle(
                NODE_METADATA_LABEL_NORMAL,
              ),
            ],
          ),
          $alignedBox(
            Direction.Vertical,
            NODE_METADATA_BOX,
            3,
            [3, 5, 3, 5],
            [...generateTaintItems()],
          ),
          $alignedBox(
            Direction.Vertical,
            NODE_METADATA_BOX,
            3,
            [3, 5, 3, 5],
            [...generateConditionItems()],
          ),
        ],
      ),
    ]);
}

export function generatePodElements(node: GraphNode): GraphObject {
  const generatePodBoxStyle: (p: PodGraphData) => ElementStyle = (
    p: PodGraphData,
  ) => {
    if (p.phase == 'Pending' || p.phase == 'Unknown') {
      return {
        fill: GRAPH_COLORS.WARNING_LIGHTER,
      };
    }
    if (p.phase == 'Succeeded') {
      return {
        fill: GRAPH_COLORS.SUCCESS_LIGHTER,
      };
    }
    if (p.phase == 'Failed') {
      return {
        fill: GRAPH_COLORS.ERROR_LIGHTER,
      };
    }
    return {};
  };
  return $empty()
    .withMargin(0, 0, 0, 0)
    .withChildren([
      $alignedGroup(Direction.Vertical)
        .withAnchor(AnchorPoints.TOP)
        .withPivot(AnchorPoints.TOP)
        .withGap(10)
        .withMargin(30, 0, 0, 0)
        .withChildren(
          node.pods.map((podData) => {
            return $rect()
              .withId(`pod_${podData.namespace}_${podData.name}`)
              .withStyle(POD_STYLE)
              .withStyle(common.generateDeletedResourceStyle(podData))
              .withStyle(generatePodBoxStyle(podData))
              .withMinSize(200, 0)
              .withChildren([
                common.generateKindLabel(
                  'Pod',
                  GRAPH_COLORS.POD,
                  POD_KIND_LABEL,
                ),
                ...common.generateDeletionOrUpdateLabel(
                  podData,
                  GRAPH_COLORS.POD,
                  15,
                ),
                $hpair(
                  $empty().withChildren([
                    $empty()
                      .withAnchor(AnchorPoints.TOP)
                      .withPivot(AnchorPoints.TOP)
                      .withChildren([
                        $empty()
                          .withAnchor(AnchorPoints.TOP)
                          .withPivot(AnchorPoints.TOP)
                          .withChildren([
                            $empty()
                              .withAnchor(AnchorPoints.TOP)
                              .withPivot(AnchorPoints.TOP)
                              .withChildren([
                                common
                                  .generateNameAndNamespaceBox(podData, 15)
                                  .withMargin(25, 10, 0, 10),
                              ]),
                            generateContainerElements(podData),
                          ])
                          .withMargin(0, 20, 0, 20),
                      ]),
                  ]),
                  generatePodMetadataBox(podData),
                ).withMinimumGap(30),
              ]);
          }),
        ),
    ]);
}

export function generatePodMetadataBox(pod: PodGraphData): GraphObject {
  const generateConditionItems = () => {
    return [
      $label('conditions:').withStyle(POD_METADATA_LABEL_NORMAL),
      ...pod.conditions.map((c) =>
        $rect()
          .withStyle(
            c.is_positive_status ? TRANSPARENT_BOX : METADATA_BOX_ERROR,
          )
          .withChildren([
            $label(`${c.type} : ${c.status}`)
              .withStyle(POD_METADATA_LABEL_NORMAL)
              .withMargin(2, 5, 2, 5),
          ]),
      ),
    ];
  };

  const podPhaseLabelStyle: () => ElementStyle = () => {
    if (pod.phase == 'Running') {
      return {
        ...POD_METADATA_LABEL_NORMAL,
      };
    }
    return {
      ...POD_METADATA_LABEL_NORMAL,
      'font-weight': 500,
    };
  };

  const podPhaseBoxStyle: () => ElementStyle = () => {
    if (pod.phase == 'Failed') {
      return {
        ...POD_METADATA_BOX,
        fill: GRAPH_COLORS.ERROR,
      };
    }
    if (pod.phase == 'Succeeded' || pod.phase == 'Running') {
      return {
        ...POD_METADATA_BOX,
        fill: GRAPH_COLORS.SUCCESS,
      };
    }
    return {
      ...POD_METADATA_BOX,
      fill: GRAPH_COLORS.WARNING,
    };
  };

  return $empty().withChildren([
    $alignedBox(
      Direction.Vertical,
      TRANSPARENT_BOX,
      5,
      [0, 0, 0, 0],
      [
        $alignedBox(
          Direction.Vertical,
          POD_METADATA_BOX,
          0,
          [3, 10, 3, 10],
          [
            $boxed_label(
              `Pod phase:${pod.phase}`,
              podPhaseBoxStyle(),
              podPhaseLabelStyle(),
              [3, 10, 3, 10],
            ),
            $label(`PodIP:${pod.podIP}`).withStyle(POD_METADATA_LABEL_NORMAL),
          ],
        ),
        $alignedBox(
          Direction.Vertical,
          POD_METADATA_BOX,
          0,
          [3, 10, 3, 10],
          [...generateConditionItems()],
        ),
      ],
    ),
  ]);
}

export function generateContainerElements(podData: PodGraphData): GraphObject {
  const generateInitContainerLabel = (c: ContainerGraphData) =>
    !c.isInitContainer
      ? []
      : [
          $rect()
            .withAnchor(AnchorPoints.TOP_RIGHT)
            .withPivot(AnchorPoints.TOP_RIGHT)
            .withStyle(INIT_CONTAINER_LABEL_BACKGROUND)
            .withChildren([
              $label('INIT')
                .withMargin(0, 10, 0, 10)
                .withStyle(CONTAINER_TYPE_LABEL),
            ]),
        ];

  const generateContainerBorderStyle: (
    c: ContainerGraphData,
  ) => ElementStyle = (c: ContainerGraphData) => {
    if (c.status != 'Running') {
      return {
        'stroke-dasharray': '10 10',
      };
    }
    return {};
  };

  const generateContainerFillStyle: (c: ContainerGraphData) => ElementStyle = (
    c: ContainerGraphData,
  ) => {
    if (c.status == 'Running' || !c.statusReadFromManifest) return {};
    if (c.status == 'Completed') {
      return {
        fill: GRAPH_COLORS.SUCCESS_LIGHTER,
      };
    }
    return {
      fill: GRAPH_COLORS.ERROR_LIGHTER,
    };
  };
  return $empty()
    .withAnchor(AnchorPoints.TOP)
    .withPivot(AnchorPoints.TOP)
    .withMargin(70, 0, 10, 0)
    .withChildren([
      $alignedGroup(Direction.Horizontal)
        .withGap(5)
        .withChildren(
          podData.containers.map((c) => {
            return $empty().withChildren([
              $rect()
                .withMinSize(150, 0)
                .withStyle(CONTAINER_STYLE)
                .withStyle(generateContainerBorderStyle(c))
                .withStyle(generateContainerFillStyle(c))
                .withChildren([
                  common.generateKindLabel(
                    'Container',
                    GRAPH_COLORS.CONTAINER,
                    CONTAINER_TYPE_LABEL,
                    0,
                  ),
                  ...generateInitContainerLabel(c),
                  $empty()
                    .withAnchor(AnchorPoints.TOP)
                    .withPivot(AnchorPoints.TOP)
                    .withMargin(5, 0, 0, 0)
                    .withChildren([$label(c.name).withMargin(15, 0, 10, 0)]),
                  $empty()
                    .withAnchor(AnchorPoints.CENTER)
                    .withPivot(AnchorPoints.CENTER)
                    .withChildren([generateContainerStatusBox(c)])
                    .withMargin(20, 0, 10, 0)
                    .withMinSize(0, 50), // TODO: bug fix
                ]),
            ]);
          }),
        ),
    ]);
}

export function generateContainerStatusBox(
  container: ContainerGraphData,
): GraphObject {
  if (!container.statusReadFromManifest) {
    return $boxed_label(
      'No info',
      { fill: '#333' },
      CONTAINER_METADATA_LABEL,
      [2, 10, 2, 10],
    );
  }
  const readyStateLabelGenerater: () => GraphObject[] = () => {
    const color = container.ready ? GRAPH_COLORS.SUCCESS : GRAPH_COLORS.ERROR;
    if (container.status != 'Running') {
      return [];
    }
    return [
      $rect()
        .withStyle(NAMESPACE_LABEL_BACKGROUND)
        .withStyle({ fill: color })
        .withChildren([
          $empty()
            .withAnchor(AnchorPoints.CENTER)
            .withPivot(AnchorPoints.CENTER)
            .withChildren([
              $label(container.ready ? 'READY' : 'NOT READY')
                .withMargin(0, 10, 0, 10)
                .withStyle(CONTAINER_METADATA_LABEL),
            ]),
        ]),
    ];
  };

  const statusLabelGenerator: () => GraphObject[] = () => {
    const color = container.isStatusHealthy
      ? GRAPH_COLORS.SUCCESS
      : GRAPH_COLORS.ERROR;
    const labels = [container.status];
    if (container.status != 'Running') {
      labels.push(`reason: ${container.reason}`);
      labels.push(`code: ${container.code}`);
    }
    return [
      $rect()
        .withStyle(NAMESPACE_LABEL_BACKGROUND)
        .withStyle({ fill: color })
        .withChildren([
          $empty()
            .withAnchor(AnchorPoints.CENTER)
            .withPivot(AnchorPoints.CENTER)
            .withChildren([
              $alignedGroup(Direction.Vertical).withChildren(
                labels.map((l) =>
                  $empty()
                    .withPivot(AnchorPoints.TOP)
                    .withAnchor(AnchorPoints.TOP)
                    .withChildren([$label(l).withStyle(CONTAINER_TYPE_LABEL)]),
                ),
              ),
            ]),
        ]),
    ];
  };

  return $empty().withChildren([
    $alignedGroup(Direction.Vertical)
      .withGap(2)
      .withChildren([...readyStateLabelGenerater(), ...statusLabelGenerator()]),
  ]);
}
