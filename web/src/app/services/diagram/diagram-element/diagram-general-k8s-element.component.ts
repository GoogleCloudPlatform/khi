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

import { Component, computed, input } from '@angular/core';
import { DiagramElementDirective } from '../diagram-element.directive';
import { LOD } from '../lod.service';
import { DiagramWaypointAreaDirective } from '../waypoint-area.directive';
import {
  BasicDiagramElement,
  BasicDiagramNamespacedElement,
  DiagramElementKindMap,
} from '../diagram-model-types';

/**
 * Component for displaying a general Kubernetes resource element
 * Handles rendering based on resource type, name, and namespace
 */
@Component({
  selector: 'diagram-general-k8s-element',
  templateUrl: './diagram-general-k8s-element.component.html',
  styleUrl: './diagram-general-k8s-element.component.sass',
  imports: [DiagramElementDirective, DiagramWaypointAreaDirective],
  standalone: true,
})
export class DiagramGeneralK8sElementComponent {
  /** Level Of Detail reference */
  LOD = LOD;

  /** Reference to diagram element kind mapping */
  DiagramElementKindMap = DiagramElementKindMap;

  /**
   * The resource data to be displayed
   * Contains information about the Kubernetes resource
   */
  resource = input.required<BasicDiagramElement>();

  /**
   * Gets the display name for the resource kind
   * @returns The human-readable kind name based on the resource type
   */
  getKindName(): string {
    return DiagramElementKindMap[this.resource().type] || 'Resource';
  }

  /**
   * Computed property that checks if the resource has a namespace property
   */
  hasNamespace = computed(() => {
    return 'namespace' in this.resource();
  });

  /**
   * Computed property that gets the namespace of the resource if it exists
   */
  namespace = computed(() => {
    return this.hasNamespace()
      ? (this.resource() as BasicDiagramNamespacedElement).namespace
      : '';
  });
}
