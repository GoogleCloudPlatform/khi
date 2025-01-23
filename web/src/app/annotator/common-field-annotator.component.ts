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

import { ClipboardModule, Clipboard } from '@angular/cdk/clipboard';
import { CommonModule } from '@angular/common';
import { Component, Input, inject } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import * as jsyaml from 'js-yaml';
import {
  NEVER,
  Observable,
  filter,
  map,
  of,
  switchMap,
  withLatestFrom,
} from 'rxjs';
import { LongTimestampFormatPipe } from 'src/app/common/timestamp-format.pipe';
import { InspectionDataStoreService } from 'src/app/services/inspection-data-store.service';
import { ViewStateService } from 'src/app/services/view-state.service';
import { AnnotationDecider, DECISION_HIDDEN } from './annotator';
import { LogEntry } from '../store/log';
import { ResourceRevision } from '../store/revision';
import { TimelineEntry } from '../store/timeline';

@Component({
  standalone: true,
  imports: [CommonModule, MatIconModule, MatTooltipModule, ClipboardModule],
  templateUrl: './common-field-annotator.component.html',
  styleUrl: './common-field-annotator.component.sass',
})
export class CommonFieldAnnotatorComponent {
  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);
  @Input()
  icon = '';

  @Input()
  label = '';

  @Input()
  value: Observable<string> = NEVER;

  onValueClick(value: string) {
    let snackbarMessage: string;
    if (this.clipboard.copy(value)) {
      snackbarMessage = 'Copied!';
    } else {
      snackbarMessage = 'Copy failed.';
    }
    this.snackBar.open(snackbarMessage, undefined, { duration: 1000 });
  }

  /**
   * Functions used for CommonFieldAnnotator with LogEntry
   */
  /**
   * Get mapper function from actual log body object
   * @param icon
   * @param label
   * @param fieldMapper receives decoded strucure log data and maps to a string
   * @returns
   */
  public static annotationDeciderForLogBodyField(
    icon: string,
    label: string,
    //  eslint-disable-next-line @typescript-eslint/no-explicit-any
    fieldMapper: (logRoot: any) => string,
  ): AnnotationDecider<LogEntry> {
    return (l?: LogEntry | null) => {
      if (!l) return DECISION_HIDDEN;
      const dataStore = inject(InspectionDataStoreService);
      return {
        inputs: {
          icon,
          label,
          value: of(l.body).pipe(
            withLatestFrom(
              dataStore.referenceResolver.pipe(filter((tb) => !!tb)),
            ),
            switchMap(([tr, loader]) => loader!.getText(tr)),
            map((yamlStr) => fieldMapper(jsyaml.load(yamlStr))),
          ),
        },
      };
    };
  }

  public static inputMapperForTimestamp(
    icon: string,
    label: string,
  ): AnnotationDecider<LogEntry> {
    return (l?: LogEntry | null) => {
      if (!l) return DECISION_HIDDEN;
      const viewState = inject(ViewStateService);
      const tsflongPipe = new LongTimestampFormatPipe(viewState);
      return {
        inputs: {
          icon,
          label,
          value: tsflongPipe.transform(l.time),
        },
      };
    };
  }

  public static inputMapperForSummary(
    icon: string,
    label: string,
  ): AnnotationDecider<LogEntry> {
    return (l?: LogEntry | null) => {
      if (!l) return DECISION_HIDDEN;
      return {
        inputs: {
          icon,
          label,
          value: of([l.summary]),
        },
      };
    };
  }

  /**
   * Functions used for CommonFieldAnnotator with TimelineEntry
   */
  public static inputMapperForTimelineEntry(
    icon: string,
    label: string,
    fieldMapper: (tl: TimelineEntry) => string,
  ): AnnotationDecider<TimelineEntry> {
    return (tl?: TimelineEntry | null) => {
      if (!tl) return DECISION_HIDDEN;
      const result = fieldMapper(tl);
      if (!result) return DECISION_HIDDEN;
      return {
        inputs: {
          icon,
          label,
          value: of(result),
        },
      };
    };
  }

  public static inputMapperForTimelineOfRevisions(
    icon: string,
    label: string,
    fieldMapper: (tl: ResourceRevision) => string | undefined,
  ): AnnotationDecider<TimelineEntry> {
    return (tl?: TimelineEntry | null) => {
      if (!tl) return DECISION_HIDDEN;
      const values = new Set<string>();
      for (const rev of tl.revisions) {
        const value = fieldMapper(rev);
        if (value !== undefined) {
          values.add(value);
        }
      }
      if (values.size == 0) return DECISION_HIDDEN;
      return {
        inputs: {
          icon,
          label,
          value: of([...values].join(',')),
        },
      };
    };
  }
}
