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
import { DiagramModel } from '../diagram-model-types';
import { DiagramNodeElementComponent } from '../diagram-element/diagram-node-element.component';
import { DiagramGeneralK8sElementComponent } from '../diagram-element/diagram-general-k8s-element.component';
import { DiagramPathSpacerComponent } from '../diagram-element/diagram-path-spacer.component';

/**
 * Root component for rendering Kubernetes resource diagrams
 * Defines the overall structure of the diagram with upper, node, and lower tiers
 */
@Component({
  selector: 'diagram-k8s-root',
  templateUrl: './diagram-k8s-root.component.html',
  styleUrls: [
    './diagram-k8s-root.component.sass',
    './diagram-k8s-root-colors.sass',
  ],
  imports: [
    DiagramNodeElementComponent,
    DiagramGeneralK8sElementComponent,
    DiagramPathSpacerComponent,
  ],
})
export class DiagramK8sRootComponent {
  /**
   * The diagram data model to render
   * Contains Kubernetes resources organized in hierarchical tiers
   */
  model = input.required<DiagramModel>();
}
