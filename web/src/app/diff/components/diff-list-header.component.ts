/**
 * Copyright 2024 Google LLC
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
import { CommonModule } from '@angular/common';
import { ResourceTimeline, TimelineLayer } from '../../store/timeline';
import { CopiableKeyValueComponent } from 'src/app/shared/components/copiable-key-value/copiable-key-value.component';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Component for displaying the header of the diff list, which shows annotators for the selected timeline.
 */
@Component({
  selector: 'khi-diff-list-header',
  templateUrl: './diff-list-header.component.html',
  styleUrls: ['./diff-list-header.component.scss'],
  imports: [CommonModule, CopiableKeyValueComponent, KHIIconRegistrationModule],
})
export class DiffListHeaderComponent {
  /**
   * The selected timeline.
   */
  readonly timeline = input.required<ResourceTimeline | null>();

  /**
   * Computed signal for the timeline's kind.
   */
  protected readonly kind = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Kind);
  });

  /**
   * Computed signal for the timeline's namespace.
   */
  protected readonly namespace = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Namespace);
  });

  /**
   * Computed signal for the timeline's name.
   */
  protected readonly name = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Name);
  });

  /**
   * Computed signal for the timeline's subresource.
   */
  protected readonly subresource = computed(() => {
    return this.timeline()?.getNameOfLayer(TimelineLayer.Subresource);
  });
}
