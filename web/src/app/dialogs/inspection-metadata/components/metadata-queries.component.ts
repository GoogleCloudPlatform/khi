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
import { CommonModule } from '@angular/common';
import { MetadataCodeViewerComponent } from './metadata-code-viewer.component';
import { MetadataQueryViewModel } from '../types/inspection-metadata.model';

/**
 * Dumb component displaying the list of executed inspection queries.
 */
@Component({
  selector: 'khi-metadata-queries',
  imports: [CommonModule, MetadataCodeViewerComponent],
  templateUrl: './metadata-queries.component.html',
  styleUrls: ['./metadata-queries.component.scss'],
})
export class MetadataQueriesComponent {
  /** List of queries to display. */
  readonly queries = input.required<readonly MetadataQueryViewModel[]>();

  /**
   * Computes the display badge for a given query based on count, preset, or status.
   * @param query The query item.
   * @returns Formatted badge text.
   */
  protected getQueryBadge(query: MetadataQueryViewModel): string {
    if (query.estimatedCount !== undefined) {
      return `Count: ~${query.estimatedCount.toLocaleString()}`;
    }
    if (query.estimatedCountPreset) {
      return `Preset: ${query.estimatedCountPreset}`;
    }
    if (query.pending) {
      return 'Pending';
    }
    if (query.incomplete) {
      return 'Incomplete';
    }
    return '';
  }
}
