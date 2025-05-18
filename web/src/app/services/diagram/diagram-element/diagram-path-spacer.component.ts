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

import { CommonModule } from '@angular/common';
import { Component, input } from '@angular/core';
import { DiagramWaypointAreaDirective } from '../waypoint-area.directive';

/**
 * Defines the possible orientations for a path spacer
 * The orientation affects how the spacer is positioned and styled
 */
type SpacerDirection = 'horizontal' | 'vertical';

/**
 * Component that provides placement areas for arrow waypoints
 * Serves as an invisible connector between diagram elements, allowing path routing
 */
@Component({
  selector: 'diagram-path-spacer',
  templateUrl: './diagram-path-spacer.component.html',
  imports: [CommonModule, DiagramWaypointAreaDirective],
  styleUrl: './diagram-path-spacer.component.sass',
})
export class DiagramPathSpacerComponent {
  /**
   * Unique identifier for the spacer
   * Used to reference this spacer when creating waypoints for arrows
   */
  id = input.required<string>();

  /**
   * Orientation of the spacer element
   * - 'horizontal': For spacers between vertical layers (e.g. between upper tiers)
   * - 'vertical': For spacers at the sides of nodes
   */
  direction = input.required<SpacerDirection>();
}
