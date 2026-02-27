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

import { Component, computed, input, inject, resource } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LogEntry } from 'src/app/store/log';
import { TypeSeverityComponent } from './type-severity.component';
import { CommonFieldAnnotatorComponent } from 'src/app/annotator/common-field-annotator.component';
import {
  ResourceReferenceListComponent,
  ResourceRefAnnotationViewModel,
} from './resource-reference-list.component';
import { ViewStateService } from 'src/app/services/view-state.service';
import { InspectionDataStoreService } from 'src/app/services/inspection-data-store.service';
import { LogTypeMetadata } from 'src/app/zzz-generated';
import { LongTimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';
import { Observable, firstValueFrom, map } from 'rxjs';
import {
  KHIFileTextReference,
  LogAnnotationTypeResourceRef,
} from 'src/app/common/schema/khi-file-types';
import { ToTextReferenceFromKHIFileBinary } from 'src/app/common/loader/reference-type';
import { toSignal } from '@angular/core/rxjs-interop';

/**
 * Represents a view model for a generic common field displayed in the log header,
 * such as a timestamp.
 */
export interface CommonFieldViewModel {
  icon: string;
  label: string;
  value: Observable<string>;
}

/**
 * Aggregates all the extracted view models required to render the log header,
 * including severity, type, timestamp, and related resource references.
 */
export interface LogContentHeaderViewModel {
  typeSeverity: { logType: string; severity: string } | null;
  timestamp: CommonFieldViewModel | null;
  resourceRefs: { refs: ResourceRefAnnotationViewModel[] } | null;
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

  private readonly viewState = inject(ViewStateService);
  private readonly dataStore = inject(InspectionDataStoreService);
  private readonly referenceResolver = toSignal(
    this.dataStore.referenceResolver,
    { initialValue: null },
  );
  private readonly inspectionData = toSignal(this.dataStore.inspectionData, {
    initialValue: null,
  });

  /**
   * Computes the unified `LogContentHeaderViewModel` based on the current `log` input.
   * Extracts log type, severity, formatting timestamp, and merges it with
   * the resolved `resourceRefs`.
   */
  readonly viewModel = computed<LogContentHeaderViewModel>(() => {
    const l = this.log();
    if (!l || l.logIndex < 0) {
      return {
        typeSeverity: null,
        timestamp: null,
        resourceRefs: null,
      };
    }

    const typeSeverity = {
      logType: LogTypeMetadata[l.logType].label,
      severity: l.logSeverityLabel ?? 'N/A',
    };

    const timestamp = {
      icon: 'schedule',
      label: 'Timestamp',
      value: this.viewState.timezoneShift.pipe(
        map((t) => LongTimestampFormatPipe.toLongDisplayTimestamp(l.time, t)),
      ),
    };

    let resourceRefs = null;
    const paths = this.referencedResourcePaths.value();
    if (paths && paths.length > 0) {
      resourceRefs = {
        refs: paths.map((path) => {
          const splittedPath = path.split('#');
          const resourceRefLabel = `${splittedPath[splittedPath.length - 1]} of ${splittedPath[splittedPath.length - 2]}`;
          return {
            label: resourceRefLabel,
            path,
          };
        }),
      };
    }

    return {
      typeSeverity,
      timestamp,
      resourceRefs,
    };
  });

  /**
   * An asynchronous `resource` that resolves all `LogAnnotationTypeResourceRef` paths
   * associated with the current `log`. It uses `ReferenceResolver` to load buffer content
   * and queries `InspectionDataStoreService` to find all aliased timelines for the resource.
   */
  protected readonly referencedResourcePaths = resource({
    params: () => ({
      log: this.log(),
      resolver: this.referenceResolver(),
      inspectionData: this.inspectionData(),
    }),
    loader: async ({ params }) => {
      const { log, resolver, inspectionData } = params;
      if (!log || !resolver || !inspectionData) return [];

      const textRefs = log.annotations
        .filter((a) => a.type === LogAnnotationTypeResourceRef)
        .map((a) => a['path'] as KHIFileTextReference);

      if (textRefs.length === 0) return [];

      const refs = await Promise.all(
        textRefs.map((ref) =>
          firstValueFrom(
            resolver.getText(ToTextReferenceFromKHIFileBinary(ref)),
          ),
        ),
      );

      const paths = refs.flatMap((ref) => [
        ref,
        ...inspectionData.getAliasedTimelines(ref).map((t) => t.resourcePath),
      ]);
      return [...new Set(paths)];
    },
  });
}
