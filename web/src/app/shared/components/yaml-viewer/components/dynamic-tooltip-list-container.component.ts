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
import { YamlFieldAnnotation } from 'src/app/shared/components/yaml-viewer/yaml-annotation';
import { NgComponentOutlet } from '@angular/common';

/**
 * A container component that renders a list of dynamic tooltips.
 */
@Component({
  selector: 'khi-dynamic-tooltip-list-container',
  templateUrl: './dynamic-tooltip-list-container.component.html',
  styleUrls: ['./dynamic-tooltip-list-container.component.scss'],
  imports: [NgComponentOutlet],
})
export class DynamicTooltipListContainerComponent {
  /** The annotations containing the components to render and their inputs. */
  readonly annotations = input.required<YamlFieldAnnotation[]>();
}
