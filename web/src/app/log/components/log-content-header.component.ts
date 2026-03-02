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

import { Component, computed, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LogEntry } from 'src/app/store/log';
import { CommonFieldAnnotatorComponent } from 'src/app/annotator/common-field-annotator.component';
import {
  ResourceReferenceListComponent,
  ResourceRefAnnotationViewModel,
} from './resource-reference-list.component';
import { ResourceTimeline } from 'src/app/store/timeline';
import { LogTypeMetadata } from 'src/app/zzz-generated';

import { LongTimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';
import { TypeSeverityComponent } from './type-severity.component';

/**
 * Aggregates all the extracted view models required to render the log header,
 * including severity, type, timestamp, and related resource references.
 */
export interface LogContentHeaderViewModel {
  logType: string;
  severity: string;
  timestamp: string;
  resourceRefs: ResourceRefAnnotationViewModel[];
}

/**
 * The `LogHeaderComponent` provides a comprehensive view of a `LogEntry`'s metadata.
 * It renders the log's type, severity, timestamp, and a list of related resources.
 * By computing a unified `LogContentHeaderViewModel`, it coordinates data extraction across
 * multiple sub-components (like `TypeSeverityAnnotatorComponent` and `ResourceReferenceListComponent`).
 */
@Component({
  selector: 'khi-log-content-header',
  templateUrl: './log-content-header.component.html',
  styleUrls: ['./log-content-header.component.scss'],
  imports: [
    CommonModule,
    TypeSeverityComponent,
    CommonFieldAnnotatorComponent,
    ResourceReferenceListComponent,
  ],
})
export class LogContentHeaderComponent {
  /**
   * The active `LogEntry` to display in the header.
   */
  log = input<LogEntry | null>(null);

  /**
   * The timezone shift to apply to the timestamp.
   */
  timezoneShift = input<number>(0);

  /**
   * Output emitted when a resource timeline is clicked from the reference list.
   */
  resourceSelected = output<string>();

  /**
   * Output emitted when a resource timeline is hovered from the reference list.
   */
  resourceHighlighted = output<string>();

  /**
   * Input tracking the currently selected timeline to visually indicate selection state
   * in the resource reference list.
   */
  selectedTimeline = input<ResourceTimeline | null>(null);

  /**
   * The resolved paths for resource references associated with this log.
   */
  referencedResourcePaths = input<string[]>([]);

  /**
   * Computes the unified `LogContentHeaderViewModel` based on the current `log` input.
   * Extracts log type, severity, formatting timestamp, and merges it with
   * the resolved `resourceRefs`.
   */
  readonly viewModel = computed<LogContentHeaderViewModel>(() => {
    const l = this.log();
    if (!l || l.logIndex < 0) {
      return {
        logType: '',
        severity: '',
        timestamp: '',
        resourceRefs: [],
      };
    }

    let resourceRefs = [] as ResourceRefAnnotationViewModel[];
    const paths = this.referencedResourcePaths();
    if (paths && paths.length > 0) {
      resourceRefs = paths.map((path) => {
        const splittedPath = path.split('#');
        const resourceRefLabel = `${splittedPath[splittedPath.length - 1]} of ${splittedPath[splittedPath.length - 2]}`;
        return {
          label: resourceRefLabel,
          path,
        };
      });
    }

    return {
      logType: LogTypeMetadata[l.logType]?.label ?? 'Unknown',
      severity: l.logSeverityLabel ?? 'N/A',
      timestamp: LongTimestampFormatPipe.toLongDisplayTimestamp(
        l.time,
        this.timezoneShift(),
      ),
      resourceRefs,
    };
  });
}
