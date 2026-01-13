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

import { CommonModule } from '@angular/common';
import { Component, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * The left top corner indicator of the timeline chart.
 * It shows the scroll mode of the timeline chart.
 */
@Component({
  selector: 'khi-timeline-corner-indicator',
  templateUrl: './timeline-corner-indicator.component.html',
  styleUrls: ['./timeline-corner-indicator.component.scss'],
  imports: [CommonModule, MatIconModule, KHIIconRegistrationModule],
})
export class TimelineCornerIndicatorComponent {
  /**
   * Whether the current scroll mode is the scaling mode.
   */
  readonly isCurrentScalingMode = input(false);
}
