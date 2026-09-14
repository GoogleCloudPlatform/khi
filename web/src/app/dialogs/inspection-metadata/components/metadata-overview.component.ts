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
import { MatIconModule } from '@angular/material/icon';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MetadataOverviewViewModel } from '../types/inspection-metadata.model';

/**
 * Dumb component displaying the overview details of an inspection.
 */
@Component({
  selector: 'khi-metadata-overview',
  imports: [MatIconModule, KHIIconRegistrationModule],
  templateUrl: './metadata-overview.component.html',
  styleUrls: ['./metadata-overview.component.scss'],
})
export class MetadataOverviewComponent {
  /** The overview data to display. */
  readonly overview = input.required<MetadataOverviewViewModel>();
}
