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

import { Component, input, output, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { SetInputItem } from 'src/app/shared/components/set-input/set-input.component';

@Component({
  selector: 'khi-new-inspection-alias-chip',
  templateUrl: './alias-chip.component.html',
  styleUrls: ['./alias-chip.component.scss'],
  imports: [CommonModule, MatChipsModule, MatTooltipModule, MatIconModule],
})
export class SetInputAliasChipComponent {
  item = input.required<SetInputItem>();
  remove = output<void>();

  isAlias = computed(
    () => this.item().id.startsWith('@') || this.item().id.startsWith('-@'),
  );
  isExclusion = computed(() => this.item().id.startsWith('-'));

  displayLabel = computed(() => {
    const id = this.item().id;
    if (this.isAlias() && this.isExclusion()) {
      // For the case like `-@foo`, returns `foo`
      return id.substring(2);
    }
    if (this.isAlias() || this.isExclusion()) {
      // For the case like `@foo` or `-foo`, returns `foo`
      return id.substring(1);
    }
    return id;
  });
}
