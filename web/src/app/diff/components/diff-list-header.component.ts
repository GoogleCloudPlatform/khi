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
import { CopiableKeyValueComponent } from 'src/app/shared/components/copiable-key-value/copiable-key-value.component';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * View model for a single path node displayed in the diff list header.
 */
export interface HeaderPathNodeViewModel {
  /** The label key or layer type name (e.g., 'Kind', 'Namespace'). */
  readonly label: string;
  /** The actual resource identifier/name. */
  readonly value: string;
  /** The Material Symbol icon name to display. */
  readonly icon: string;
}

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
  readonly timeline = input.required<ReadonlyDomainElement<Timeline> | null>();

  /**
   * Computed signal for path nodes view models to display.
   */
  protected readonly pathNodes = computed<HeaderPathNodeViewModel[]>(() => {
    const t = this.timeline();
    if (!t) return [];
    return t.path.map((node) => ({
      label: node.type.label,
      value: node.label,
      icon: node.type.icon || 'label',
    }));
  });
}
