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
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { CopiableKeyValueComponent } from 'src/app/shared/components/copiable-key-value/copiable-key-value.component';
import {
  ResourceReferenceListComponent,
  ResourceRefAnnotationViewModel,
} from './resource-reference-list.component';
import { Timeline } from 'src/app/store/domain/timeline';
import { LongTimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';
import { TypeSeverityComponent } from './type-severity.component';

import { LogType, Severity } from 'src/app/store/domain/style';

/**
 * Aggregates all the extracted view models required to render the log header,
 * including severity, type, timestamp, and related resource references.
 */
export interface LogContentHeaderViewModel {
  logType: LogType | null;
  severity: Severity | null;
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
    CopiableKeyValueComponent,
    ResourceReferenceListComponent,
  ],
})
export class LogContentHeaderComponent {
  /**
   * The active `LogEntry` to display in the header.
   */
  log = input<ReadonlyDomainElement<Log> | null>(null);

  /**
   * The timezone shift to apply to the timestamp.
   */
  timezoneShift = input<number>(0);

  /**
   * Output emitted when a resource timeline is clicked from the reference list.
   */
  timelineSelected = output<number>();

  /**
   * Output emitted when a resource timeline is hovered from the reference list.
   */
  timelineHighlighted = output<number>();

  /**
   * Input tracking the currently selected timeline to visually indicate selection state
   * in the resource reference list.
   */
  selectedTimeline = input<ReadonlyDomainElement<Timeline> | null>(null);

  /**
   * The pre-resolved resource references associated with this log.
   */
  resourceRefs = input<ResourceRefAnnotationViewModel[]>([]);

  /**
   * Computes the unified `LogContentHeaderViewModel` based on the current `log` input.
   * Extracts log type, severity, formatting timestamp, and merges it with
   * the resolved `resourceRefs`.
   */
  readonly viewModel = computed<LogContentHeaderViewModel>(() => {
    const l = this.log();
    if (!l || l.logIndex < 0) {
      return {
        logType: null,
        severity: null,
        timestamp: '',
        resourceRefs: [],
      };
    }

    return {
      logType: l.logType,
      severity: l.severity,
      timestamp: LongTimestampFormatPipe.toLongDisplayTimestamp(
        l.legacyTimestampMs,
        this.timezoneShift(),
      ),
      resourceRefs: this.resourceRefs(),
    };
  });
}
