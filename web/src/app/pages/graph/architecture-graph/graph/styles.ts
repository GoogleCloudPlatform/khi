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

import { ElementStyle } from './base/base-containers';

export const GRAPH_COLORS = {
  NODE: '#4caf50',
  POD: '#3949AB',
  CONTAINER: '#F4511E',
  SERVICE: '#FB8C00',
  INIT_CONTAINER: '#BF360C',
  SUCCESS: '#00C853',

  // pod owners,
  REPLICA_SET: '#0D47A1',
  DAEMON_SET: '#9C27B0',
  JOB: '#00796B',
  CRONJOB: '#00838F',
  DEPLOYMENT: '#1976D2',

  ERROR: '#D32F2F',
  WARNING: '#F0A100',

  WARNING_LIGHTER: '#FFC170',
  SUCCESS_LIGHTER: '#80F883',
  ERROR_LIGHTER: '#F38F8F',
};

export const NODE_STYLE: ElementStyle = {
  fill: '#FAFAFA',
  'stroke-width': '4',
  stroke: GRAPH_COLORS.NODE,
  rx: 4,
  ry: 4,
};

export const NODE_METADATA_BOX: ElementStyle = {
  fill: GRAPH_COLORS.NODE,
  rx: 4,
  ry: 4,
};

export const METADATA_BOX_ERROR: ElementStyle = {
  fill: GRAPH_COLORS.ERROR,
  rx: 4,
  ry: 4,
};

export const METADATA_BOX_WARNING: ElementStyle = {
  fill: GRAPH_COLORS.WARNING,
  rx: 4,
  ry: 4,
};

export const NODE_METADATA_LABEL_ERROR: ElementStyle = {
  'font-weight': 500,
  fill: GRAPH_COLORS.ERROR,
  'font-size': 15,
};

export const NODE_METADATA_LABEL_NORMAL: ElementStyle = {
  fill: 'white',
  'font-size': 15,
};

export const CONTAINER_METADATA_LABEL: ElementStyle = {
  fill: 'white',
  'font-size': 10,
};

export const SERVICE_METADATA_LABEL_NORMAL: ElementStyle = {
  fill: 'white',
  'font-size': 15,
};

export const POD_STYLE: ElementStyle = {
  fill: '#FAFAFA',
  'stroke-width': '2',
  stroke: GRAPH_COLORS.POD,
  rx: 4,
  ry: 4,
};

export const POD_METADATA_BOX: ElementStyle = {
  fill: GRAPH_COLORS.POD,
  rx: 4,
  ry: 4,
};

export const POD_METADATA_LABEL_NORMAL: ElementStyle = {
  fill: 'white',
  'font-size': 10,
};

export const CONTAINER_STYLE: ElementStyle = {
  fill: '#FAFAFA',
  'stroke-width': '2',
  stroke: GRAPH_COLORS.CONTAINER,
};

export const FINSIHED_CONTAINER_STYLE: ElementStyle = {
  fill: '#BABABA',
  'stroke-width': '2',
  stroke: GRAPH_COLORS.CONTAINER,
};

export const TRANSPARENT_BOX: ElementStyle = {
  fill: 'transparent',
  stroke: 'transparent',
};

export const NAMESPACE_LABEL_BACKGROUND: ElementStyle = {
  fill: '#888',
  rx: 4,
  ry: 4,
};

export const INIT_CONTAINER_LABEL_BACKGROUND: ElementStyle = {
  fill: GRAPH_COLORS.INIT_CONTAINER,
};

export const NODE_NAME_LABEL: ElementStyle = {
  'font-size': 25,
};

export const NODE_KIND_LABEL: ElementStyle = {
  'font-size': 30,
  fill: 'white',
  'font-weight': 500,
};

export const POD_KIND_LABEL: ElementStyle = {
  'font-size': 20,
  fill: 'white',
  'font-weight': 500,
};

export const CONTAINER_TYPE_LABEL: ElementStyle = {
  'font-size': 10,
  fill: 'white',
  'font-weight': 500,
};

export const KIND_LABEL_FONT_STYLE: ElementStyle = {
  fill: 'white',
  'font-weight': 500,
};

export const KIND_LABEL_BACKGROUND_BASE: ElementStyle = {
  rx: 4,
  ry: 4,
};

export const SERVICE_STYLE: ElementStyle = {
  fill: '#FAFAFA',
  'stroke-width': '2',
  stroke: GRAPH_COLORS.SERVICE,
  rx: 4,
  ry: 4,
};

export const SERVICE_KIND_LABEL: ElementStyle = {
  'font-size': 20,
  fill: 'white',
  'font-weight': 500,
};

export const SERVICE_METADATA_BOX: ElementStyle = {
  fill: GRAPH_COLORS.SERVICE,
  rx: 4,
  ry: 4,
};

export const TIMESTAMP_LABEL: ElementStyle = {
  'font-size': 40,
  fill: '#444',
};

export const GRAPH_DESCRIPTION_LABEL: ElementStyle = {
  'font-size': 15,
  fill: '#EEE',
  'font-weight': 500,
};

export const GRAPH_DESCRIPTION_BACKGROUND: ElementStyle = {
  fill: '#111',
  rx: 4,
  ry: 4,
};

export const GRAPH_WARNING_LABEL: ElementStyle = {
  'font-size': 25,
  fill: 'white',
  'font-weight': 500,
};

export const GRAPH_WARNING_BACKGROUND: ElementStyle = {
  fill: 'red',
  stroke: 'red',
  rx: 4,
  ry: 4,
};

export const POD_OWNER_STYLE: ElementStyle = {
  fill: '#FAFAFA',
  'stroke-width': '4',
  rx: 4,
  ry: 4,
};

export const POD_OWNER_METADATA_STYLE: ElementStyle = {
  rx: 4,
  ry: 4,
  fill: 'transparent',
};

export const POD_OWNER_KIND_LABEL: ElementStyle = {
  'font-size': 15,
  'font-weight': 500,
  fill: 'white',
};

export const POD_OWNER_METADATA_LABEL: ElementStyle = {
  'font-size': 15,
  fill: 'white',
};
