/**
 * Copyright 2025 Google LLC
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
  BasicDiagramNamespacedElement,
  ContainerDiagramElement,
  DiagramElementType,
  DiagramModel,
  DiagramPathType,
  NodeDiagramElement,
  PathElement,
  PodDiagramElement,
} from './diagram-model-types';

// Sample Container elements
const nginxContainer: ContainerDiagramElement = {
  id: 'container-nginx-1',
  type: DiagramElementType.Container,
  name: 'nginx',
  image: 'nginx:1.21',
};

const redisContainer: ContainerDiagramElement = {
  id: 'container-redis-1',
  type: DiagramElementType.Container,
  name: 'redis',
  image: 'redis:6.2-alpine',
};

const appContainer: ContainerDiagramElement = {
  id: 'container-app-1',
  type: DiagramElementType.Container,
  name: 'app',
  image: 'myapp:latest',
};

const dbContainer: ContainerDiagramElement = {
  id: 'container-db-1',
  type: DiagramElementType.Container,
  name: 'postgres',
  image: 'postgres:13',
};

const promContainer: ContainerDiagramElement = {
  id: 'container-prometheus-1',
  type: DiagramElementType.Container,
  name: 'prometheus',
  image: 'prom/prometheus:v2.32.1',
};

// Sample Pod elements
const webPod: PodDiagramElement = {
  id: 'pod-web-84b87c8f55-2mh5j',
  type: DiagramElementType.Pod,
  name: 'web-84b87c8f55-2mh5j',
  namespace: 'default',
  containers: [nginxContainer, appContainer],
};

const cachePod: PodDiagramElement = {
  id: 'pod-cache-7c4df56f69-z8vx2',
  type: DiagramElementType.Pod,
  name: 'cache-7c4df56f69-z8vx2',
  namespace: 'default',
  containers: [redisContainer],
};

const dbPod: PodDiagramElement = {
  id: 'pod-db-6c64cdb966-lqz9x',
  type: DiagramElementType.Pod,
  name: 'db-6c64cdb966-lqz9x',
  namespace: 'default',
  containers: [dbContainer],
};

const monitoringPod: PodDiagramElement = {
  id: 'pod-monitoring-67d8fb584b-v5jl2',
  type: DiagramElementType.Pod,
  name: 'monitoring-67d8fb584b-v5jl2',
  namespace: 'kube-system',
  containers: [promContainer],
};

// Sample Node elements
const workerNode1: NodeDiagramElement = {
  id: 'node-worker-1',
  type: DiagramElementType.Node,
  name: 'worker-1',
  pods: [webPod, cachePod],
};

const workerNode2: NodeDiagramElement = {
  id: 'node-worker-2',
  type: DiagramElementType.Node,
  name: 'worker-2',
  pods: [dbPod, monitoringPod],
};

// Upper tier resources (controllers and their controllers)
/**
 * Represents a ReplicaSet resource that manages the web pods
 * Part of the upper tier resources, directly controlling pod replicas
 */
const webReplicaSet: BasicDiagramNamespacedElement = {
  id: 'rs-web-84b87c8f55',
  type: DiagramElementType.ReplicaSet,
  name: 'web-84b87c8f55',
  namespace: 'default',
};

/**
 * Represents a ReplicaSet resource that manages the cache pods
 * Part of the upper tier resources, directly controlling pod replicas
 */
const cacheReplicaSet: BasicDiagramNamespacedElement = {
  id: 'rs-cache-7c4df56f69',
  type: DiagramElementType.ReplicaSet,
  name: 'cache-7c4df56f69',
  namespace: 'default',
};

/**
 * Represents a Deployment resource that manages the web ReplicaSet
 * Part of the upper tier resources, providing declarative updates for web application
 */
const webDeployment: BasicDiagramNamespacedElement = {
  id: 'deploy-web',
  type: DiagramElementType.Deployment,
  name: 'web',
  namespace: 'default',
};

/**
 * Represents a Deployment resource that manages the cache ReplicaSet
 * Part of the upper tier resources, providing declarative updates for cache service
 */
const cacheDeployment: BasicDiagramNamespacedElement = {
  id: 'deploy-cache',
  type: DiagramElementType.Deployment,
  name: 'cache',
  namespace: 'default',
};

/**
 * Represents a StatefulSet resource that manages the database pod
 * Provides stable network identities and persistent storage for database
 */
const dbStatefulSet: BasicDiagramNamespacedElement = {
  id: 'sts-db',
  type: DiagramElementType.StatefulSet,
  name: 'db',
  namespace: 'default',
};

// Lower tier resources (services, storage, and ingress)
/**
 * Represents a Service resource that exposes the web pods
 * Part of the lower tier resources, providing network access to pods
 */
const webService: BasicDiagramNamespacedElement = {
  id: 'svc-web',
  type: DiagramElementType.Service,
  name: 'web',
  namespace: 'default',
};

/**
 * Represents a Service resource that exposes the cache pods
 * Part of the lower tier resources, providing internal network access to cache
 */
const cacheService: BasicDiagramNamespacedElement = {
  id: 'svc-cache',
  type: DiagramElementType.Service,
  name: 'cache',
  namespace: 'default',
};

/**
 * Represents a Service resource that exposes the database pods
 * Part of the lower tier resources, providing stable network identity for database
 */
const dbService: BasicDiagramNamespacedElement = {
  id: 'svc-db',
  type: DiagramElementType.Service,
  name: 'db',
  namespace: 'default',
};

/**
 * Represents a PersistentVolumeClaim used by the database pod
 * Provides persistent storage for database data
 */
const dbPvc: BasicDiagramNamespacedElement = {
  id: 'pvc-db-data',
  type: DiagramElementType.PersistentVolumeClaim,
  name: 'db-data',
  namespace: 'default',
};

/**
 * Represents an Ingress resource that routes external traffic to the web service
 * Provides HTTP/HTTPS routing to internal services
 */
const webIngress: BasicDiagramNamespacedElement = {
  id: 'ingress-web',
  type: DiagramElementType.Ingress,
  name: 'web-ingress',
  namespace: 'default',
};

/**
 * Represents a ConfigMap containing application configuration
 * Mounted into web pods as configuration
 */
const appConfig: BasicDiagramNamespacedElement = {
  id: 'cm-app-config',
  type: DiagramElementType.ConfigMap,
  name: 'app-config',
  namespace: 'default',
};

/**
 * Path connections between upper tier resources and nodes/lower tiers
 * Each index corresponds to connections from upper[index] to the layer below it
 */
const upperPaths: PathElement[][] = [
  // Level 0: Connections between ReplicaSets/StatefulSets and Pods
  [
    {
      outerID: 'rs-web-84b87c8f55',
      innerID: 'pod-web-84b87c8f55-2mh5j',
      type: DiagramPathType.Owner,
    },
    {
      outerID: 'rs-cache-7c4df56f69',
      innerID: 'pod-cache-7c4df56f69-z8vx2',
      type: DiagramPathType.Owner,
    },
    {
      outerID: 'sts-db',
      innerID: 'pod-db-6c64cdb966-lqz9x',
      type: DiagramPathType.Owner,
    },
  ],
  // Level 1: Connections between Deployments and ReplicaSets
  [
    {
      outerID: 'deploy-web',
      innerID: 'rs-web-84b87c8f55',
      type: DiagramPathType.Owner,
    },
    {
      outerID: 'deploy-cache',
      innerID: 'rs-cache-7c4df56f69',
      type: DiagramPathType.Owner,
    },
  ],
];

/**
 * Path connections between lower tier resources and nodes/other lower tiers
 * Each index corresponds to connections from lower[index] to the layer above it
 */
const lowerPaths: PathElement[][] = [
  // Level 0: Connections between Services/PVCs/ConfigMaps and Pods
  [
    {
      outerID: 'svc-web',
      innerID: 'pod-web-84b87c8f55-2mh5j',
      type: DiagramPathType.Traffic,
    },
    {
      outerID: 'svc-cache',
      innerID: 'pod-cache-7c4df56f69-z8vx2',
      type: DiagramPathType.Traffic,
    },
    {
      outerID: 'svc-db',
      innerID: 'pod-db-6c64cdb966-lqz9x',
      type: DiagramPathType.Traffic,
    },
    {
      outerID: 'pvc-db-data',
      innerID: 'pod-db-6c64cdb966-lqz9x',
      type: DiagramPathType.Use,
    },
    {
      outerID: 'cm-app-config',
      innerID: 'pod-web-84b87c8f55-2mh5j',
      type: DiagramPathType.Use,
    },
  ],
  // Level 1: Connections between Ingress and Services
  [
    {
      outerID: 'ingress-web',
      innerID: 'svc-web',
      type: DiagramPathType.Traffic,
    },
  ],
];

// Sample diagram model with hierarchical structure and connections
export const SAMPLE_DIAGRAM: DiagramModel = {
  // Upper tier (controllers)
  upper: [
    [webReplicaSet, cacheReplicaSet, dbStatefulSet], // Level 0: closest to nodes
    [webDeployment, cacheDeployment], // Level 1: higher level controllers
  ],
  upperPaths: upperPaths,
  // Lower tier (services, storage, etc.)
  lower: [
    [webService, cacheService, dbService, dbPvc, appConfig], // Level 0: closest to nodes
    [webIngress], // Level 1: external access
  ],
  lowerPaths: lowerPaths,
  nodes: [workerNode1, workerNode2],
};
