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
import { PodDiagramElement } from '../diagram-model-types';

/**
 * Component to display a Kubernetes Pod element with its containers
 */
@Component({
  selector: 'diagram-pod-element',
  templateUrl: './diagram-pod-element.component.html',
  styleUrl: './diagram-pod-element.component.sass',
  imports: [DiagramElementDirective, DiagramWaypointAreaDirective],
  standalone: true,
})
export class DiagramPodElementComponent {
  LOD = LOD;

  /**
   * The pod data to be displayed
   */
  pod = input.required<PodDiagramElement>();
}
