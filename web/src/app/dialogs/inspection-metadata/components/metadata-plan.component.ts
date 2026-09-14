/**
 * Copyright 2026 Google LLC
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
import { MetadataCodeViewerComponent } from './metadata-code-viewer.component';
import { MetadataPlanViewModel } from '../types/inspection-metadata.model';

/**
 * Dumb component displaying the inspection task plan graph.
 */
@Component({
  selector: 'khi-metadata-plan',
  imports: [MetadataCodeViewerComponent],
  templateUrl: './metadata-plan.component.html',
  styleUrls: ['./metadata-plan.component.scss'],
})
export class MetadataPlanComponent {
  /** The plan data to display. */
  readonly plan = input.required<MetadataPlanViewModel>();
}
