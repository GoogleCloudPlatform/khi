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

/**
 * Represents the types of elements that can be visualized in the diagram
 * Each type corresponds to a specific Kubernetes resource kind
 */
export enum DiagramElementType {
  Invalid = '',
  Node = 'node',
  Pod = 'pod',
  Container = 'container',
  // Upper-tier resources
  Deployment = 'deployment',
  ReplicaSet = 'replicaset',
  StatefulSet = 'statefulset',
  DaemonSet = 'daemonset',
  Job = 'job',
  CronJob = 'cronjob',
  // Lower-tier resources
  Service = 'service',
  PersistentVolumeClaim = 'pvc',
  ConfigMap = 'configmap',
  Secret = 'secret',
  Ingress = 'ingress',
}

/**
 * Defines the relationship types between diagram elements
 * Each type has distinct visual representation and semantic meaning
 */
export enum DiagramPathType {
  /**
   * No specific relationship
   */
  Invalid = '',

  /**
   * Ownership relationship where one resource controls another
   * Example: Deployment → ReplicaSet → Pod
   */
  Owner = 'owner',

  /**
   * Usage relationship where one resource consumes another
   * Example: Pod → PersistentVolumeClaim
   */
  Use = 'use',

  /**
   * Network traffic flow between resources
   * Example: Service → Pod, Ingress → Service
   */
  Traffic = 'traffic',
}

/**
 * Mapping between DiagramElementType and their display names
 * Used for consistent labeling across the application
 */
export const DiagramElementKindMap: Record<DiagramElementType, string> = {
  [DiagramElementType.Invalid]: '',
  [DiagramElementType.Node]: 'Node',
  [DiagramElementType.Pod]: 'Pod',
  [DiagramElementType.Container]: 'Container',
  // Upper-tier resources
  [DiagramElementType.Deployment]: 'Deployment',
  [DiagramElementType.ReplicaSet]: 'ReplicaSet',
  [DiagramElementType.StatefulSet]: 'StatefulSet',
  [DiagramElementType.DaemonSet]: 'DaemonSet',
  [DiagramElementType.Job]: 'Job',
  [DiagramElementType.CronJob]: 'CronJob',
  // Lower-tier resources
  [DiagramElementType.Service]: 'Service',
  [DiagramElementType.PersistentVolumeClaim]: 'PVC',
  [DiagramElementType.ConfigMap]: 'ConfigMap',
  [DiagramElementType.Secret]: 'Secret',
  [DiagramElementType.Ingress]: 'Ingress',
};

export interface DiagramModel {
  frames: DiagramFrame[];
}

/**
 * Main data model for the Kubernetes diagram representing a single frame on the diagram.
 * Contains the complete structure of resources and their relationships
 */
export interface DiagramFrame {
  /**
   * The timestamp of this frame.
   */
  ts: number;
  /**
   * Upper general kubernetes elements shown over node list.
   * Lists with lower index placed closer to the list of nodes.
   */
  upper: BasicDiagramElement[][];

  /**
   * Diagram paths connecting elements in differnt layers.
   * index=0 connects upper[0] and node layer. index=i(i>0) connects upper[i] and upper[i-1]
   */
  upperPaths: PathElement[][];
  /**
   * Lower genral kubernetes elements shown over node list.
   * Lists with lower index placed closer to the list of nodes.
   */
  lower: BasicDiagramElement[][];
  /**
   * Diagram paths connecting elements in differnt layers.
   * index=0 connects lower[0] and node layer. index=i(i>0) connects lower[i] and lower[i-1]
   */
  lowerPaths: PathElement[][];
  nodes: NodeDiagramElement[];
}

/**
 * Defines a connection path between diagram elements
 * Represents relationships between Kubernetes resources
 */
export interface PathElement {
  /**
   * ID of the outer element (further from the node layer)
   * Typically a controller or a higher-level resource
   */
  outerID: string;

  /**
   * ID of the inner element (closer to the node layer)
   * Typically a managed resource or a pod
   */
  innerID: string;

  /**
   * Type of relationship between the elements
   * Determines the visual style and semantic meaning of the connection
   */
  type: DiagramPathType;
}

/**
 * Base interface for all diagram elements
 * Provides common properties required for any displayed resource
 */
export interface BasicDiagramElement {
  /**
   * Unique identifier for the element
   * Used for referencing in paths and DOM elements
   */
  id: string;

  /**
   * Type of the Kubernetes resource
   * Determines rendering style and behavior
   */
  type: DiagramElementType;

  /**
   * Display name of the resource
   * Shown in the diagram UI
   */
  name: string;
}

/**
 * Extension of basic diagram element with namespace information
 * Used for Kubernetes resources that exist within a namespace
 */
export interface BasicDiagramNamespacedElement extends BasicDiagramElement {
  /**
   * Kubernetes namespace containing the resource
   * Provides isolation and categorization context
   */
  namespace: string;
}

/**
 * Represents a Kubernetes Node in the diagram
 * Contains the list of pods scheduled to run on this node
 */
export interface NodeDiagramElement extends BasicDiagramElement {
  type: DiagramElementType.Node;
  /**
   * Pods running on this node
   * Displayed within the node's boundaries in the diagram
   */
  pods: PodDiagramElement[];
}

/**
 * Represents a Kubernetes Pod in the diagram
 * A pod is the smallest deployable unit in Kubernetes
 */
export interface PodDiagramElement extends BasicDiagramNamespacedElement {
  type: DiagramElementType.Pod;
  /**
   * Containers within this pod
   * A pod may contain one or more containers sharing network and storage
   */
  containers: ContainerDiagramElement[];
}

/**
 * Represents a Container within a Pod
 * Containers are the actual runtime instances of applications
 */
export interface ContainerDiagramElement extends BasicDiagramElement {
  type: DiagramElementType.Container;
  /**
   * Container image name
   * Includes repository, name and tag (e.g., nginx:1.21)
   */
  image: string;
}
