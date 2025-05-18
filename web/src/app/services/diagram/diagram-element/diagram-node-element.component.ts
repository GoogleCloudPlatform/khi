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

import { Component, input } from '@angular/core';
import { DiagramElementDirective } from '../diagram-element.directive';
import { LOD } from '../lod.service';
import { DiagramWaypointAreaDirective } from '../waypoint-area.directive';
import { NodeDiagramElement } from '../diagram-model-types';
import { DiagramPodElementComponent } from './diagram-pod-element.component';

/**
 * Component to display a Kubernetes Node element with its pods
 */
@Component({
  selector: 'diagram-node-element',
  templateUrl: './diagram-node-element.component.html',
  styleUrl: './diagram-node-element.component.sass',
  imports: [
    DiagramElementDirective,
    DiagramWaypointAreaDirective,
    DiagramPodElementComponent,
  ],
  standalone: true,
})
export class DiagramNodeElementComponent {
  LOD = LOD;

  /**
   * The node data to be displayed
   */
  node = input.required<NodeDiagramElement>();

  /**
   * Generate a unique ID for a pod based on the node and pod IDs
   * @param podId - The ID of the pod
   * @returns A unique ID string
   */
  getPodElementId(podId: string): string {
    return `${this.node().id}-pod-${podId}`;
  }
}
