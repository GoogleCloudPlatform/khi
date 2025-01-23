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

import { K8sCondition } from 'src/app/store/k8s-types';
import {
  GraphData,
  GraphPodOwnerBase,
  PodOwnerKinds,
  PodOwnerOwnerKinds,
} from '../../../../../common/schema/graph-schema';
import {
  AnchorPoints,
  Direction,
  ElementStyle,
  GraphObject,
} from '../base/base-containers';
import { $alignedGroup, $empty, $label, $rect } from '../builder-alias';
import {
  GRAPH_COLORS,
  POD_OWNER_KIND_LABEL,
  POD_OWNER_METADATA_LABEL,
  POD_OWNER_METADATA_STYLE,
  POD_OWNER_STYLE,
} from '../styles';
import * as common from './common';

interface PodOwnerMetadata {
  label: string;
  value: string;
  state: 'none' | 'normal' | 'warning' | 'error' | 'success';
}

type PodOwnerMetadataParser = (data: GraphPodOwnerBase) => PodOwnerMetadata[];

function defaultMetadataParsers(data: GraphPodOwnerBase): PodOwnerMetadata[] {
  const nonConditions = Object.keys(data.status)
    .filter((f) => f != 'conditions')
    .map((k) => ({
      label: k,
      value: data.status[k],
      state: 'normal',
    }));
  const conditions = [] as PodOwnerMetadata[];
  if ('conditions' in data.status) {
    const conditionsData = data.status['conditions'] as K8sCondition[];

    conditions.push({
      label: '* conditions',
      value: '',
      state: 'normal',
    });
    for (const condition of conditionsData) {
      conditions.push({
        label: condition.type,
        value: condition.status,
        state: 'normal',
      });
    }
    conditions.push({
      label: '* others',
      value: '',
      state: 'normal',
    });
  }

  return [...conditions, ...nonConditions] as PodOwnerMetadata[];
}

const POD_OWNER_METADATA_PARSERS: {
  [kind in PodOwnerKinds | PodOwnerOwnerKinds]: PodOwnerMetadataParser;
} = {
  daemonset: defaultMetadataParsers,
  job: defaultMetadataParsers,
  replicaset: defaultMetadataParsers,
  cronjob: defaultMetadataParsers,
  deployment: defaultMetadataParsers,
};

export function generatePodOwnerRow(
  data: GraphData,
  owners: { [kind: string]: GraphPodOwnerBase[] },
): GraphObject {
  return $empty().withChildren([
    $alignedGroup(Direction.Horizontal)
      .withAnchor(AnchorPoints.CENTER)
      .withPivot(AnchorPoints.CENTER)
      .withGap(30)
      .withChildren(
        Object.keys(owners).map((k) =>
          generatePodOwnerRowForKind(
            k as PodOwnerKinds | PodOwnerOwnerKinds,
            owners[k],
          ),
        ),
      ),
  ]);
}

function generatePodOwnerRowForKind(
  kind: PodOwnerKinds | PodOwnerOwnerKinds,
  owners: GraphPodOwnerBase[],
): GraphObject {
  return $alignedGroup(Direction.Horizontal)
    .withGap(10)
    .withChildren(owners.map((owner) => generatePodOwnerBox(kind, owner)));
}

function generatePodOwnerBox(
  kind: PodOwnerKinds | PodOwnerOwnerKinds,
  owner: GraphPodOwnerBase,
): GraphObject {
  return $rect()
    .withStyle(getPodOwnerBoxStyle(kind))
    .withMinSize(0, 300)
    .withId(`${kind}_${owner.namespace}_${owner.name}`)
    .withChildren([
      common.generateKindLabel(
        toUpperOnlyFirstChar(kind),
        getKindColor(kind),
        POD_OWNER_KIND_LABEL,
      ),
      ...common.generateDeletionOrUpdateLabel(owner, getKindColor(kind), 20),
      $alignedGroup(Direction.Horizontal).withChildren([
        $alignedGroup(Direction.Vertical).withChildren([
          $empty().withChildren([
            common
              .generateNameAndNamespaceBox(owner, 15)
              .withMargin(25, 20, 10, 20),
          ]),
        ]),
        generatePodOwnerMetadata(kind, owner),
      ]),
    ]);
}

function generatePodOwnerMetadata(
  kind: PodOwnerKinds | PodOwnerOwnerKinds,
  owner: GraphPodOwnerBase,
): GraphObject {
  const metadata = POD_OWNER_METADATA_PARSERS[kind as PodOwnerKinds](owner);
  return $rect()
    .withStyle(getPodOwnerMetadataBoxStyle(kind))
    .withChildren([
      $alignedGroup(Direction.Vertical).withChildren(
        metadata
          .filter((m) => m.state != 'none')
          .map((m) =>
            $rect()
              .withStyle(stateToMetadataBackgroundStyle(m.state))
              .withChildren([
                $label(`${m.label}:${m.value}`)
                  .withStyle(POD_OWNER_METADATA_LABEL)
                  .withMargin(3, 5, 3, 5),
              ]),
          ),
      ),
    ]);
}

function getPodOwnerBoxStyle(
  kind: PodOwnerKinds | PodOwnerOwnerKinds,
): ElementStyle {
  return {
    ...POD_OWNER_STYLE,
    stroke: getKindColor(kind),
  };
}

function getPodOwnerMetadataBoxStyle(
  kind: PodOwnerKinds | PodOwnerOwnerKinds,
): ElementStyle {
  return {
    ...POD_OWNER_METADATA_STYLE,
    fill: getKindColor(kind),
  };
}

function getKindColor(kind: PodOwnerKinds | PodOwnerOwnerKinds): string {
  switch (kind) {
    case 'daemonset':
      return GRAPH_COLORS.DAEMON_SET;
    case 'job':
      return GRAPH_COLORS.JOB;
    case 'replicaset':
      return GRAPH_COLORS.REPLICA_SET;
    case 'deployment':
      return GRAPH_COLORS.DEPLOYMENT;
    case 'cronjob':
      return GRAPH_COLORS.CRONJOB;
  }
  throw new Error('Unsupported kind');
}

function stateToMetadataBackgroundStyle(
  state: 'normal' | 'warning' | 'error' | 'success' | 'none',
): ElementStyle {
  if (state == 'normal')
    return {
      ...POD_OWNER_METADATA_STYLE,
    };
  let color = '';
  switch (state) {
    case 'error':
      color = GRAPH_COLORS.ERROR;
      break;
    case 'warning':
      color = GRAPH_COLORS.WARNING;
      break;
    case 'success':
      color = GRAPH_COLORS.SUCCESS;
      break;
  }
  return {
    ...POD_OWNER_METADATA_STYLE,
    fill: color,
  };
}

function toUpperOnlyFirstChar(s: string): string {
  return s[0].toUpperCase() + s.substring(1);
}
