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

@Component({
  selector: 'sample-diagram-element',
  templateUrl: './sample-diagram-element.component.html',
  styleUrl: './sample-diagram-element.component.sass',
  imports: [DiagramElementDirective],
})
export class SampleDiagramElementComponent {
  lod = LOD;
  id = input.required<string>();
}
