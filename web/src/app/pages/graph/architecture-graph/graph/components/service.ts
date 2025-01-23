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

import { ServiceGraphData } from '../../../../../common/schema/graph-schema';
import { Direction, GraphObject } from '../base/base-containers';
import {
  $alignedBox,
  $alignedGroup,
  $empty,
  $hpair,
  $label,
  $rect,
} from '../builder-alias';
import {
  GRAPH_COLORS,
  SERVICE_KIND_LABEL,
  SERVICE_METADATA_BOX,
  SERVICE_METADATA_LABEL_NORMAL,
  SERVICE_STYLE,
  TRANSPARENT_BOX,
} from '../styles';
import * as common from './common';
export function generateServiceBox(service: ServiceGraphData): GraphObject {
  return $rect()
    .withStyle(SERVICE_STYLE)
    .withStyle(common.generateDeletedResourceStyle(service))
    .withId(`service_${service.namespace}_${service.name}`)
    .withMinSize(0, 200)
    .withChildren([
      common.generateKindLabel(
        'Service',
        GRAPH_COLORS.SERVICE,
        SERVICE_KIND_LABEL,
        4,
      ),
      ...common.generateDeletionOrUpdateLabel(
        service,
        GRAPH_COLORS.SERVICE,
        20,
      ),
      $empty().withChildren([
        $hpair(
          $empty().withChildren([
            $alignedGroup(Direction.Vertical)
              .withMargin(30, 20, 20, 20)
              .withChildren([common.generateNameAndNamespaceBox(service, 20)]),
          ]),
          generateServiceMetadatabox(service),
        ),
      ]),
    ]);
}

export function generateServiceMetadatabox(
  service: ServiceGraphData,
): GraphObject {
  return $empty().withChildren([
    $alignedBox(
      Direction.Vertical,
      TRANSPARENT_BOX,
      5,
      [0, 0, 0, 0],
      [
        $alignedBox(
          Direction.Vertical,
          SERVICE_METADATA_BOX,
          0,
          [3, 5, 3, 5],
          [
            $label(`Type:${service.type}`).withStyle(
              SERVICE_METADATA_LABEL_NORMAL,
            ),
            $label(`Cluster IP:${service.clusterIp}`).withStyle(
              SERVICE_METADATA_LABEL_NORMAL,
            ),
          ],
        ),
      ],
    ),
  ]);
}
