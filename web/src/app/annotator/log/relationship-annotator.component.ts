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

import { CommonModule } from '@angular/common';
import { Component, Input, inject } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import {
  NEVER,
  Observable,
  filter,
  forkJoin,
  map,
  of,
  shareReplay,
  switchMap,
  withLatestFrom,
} from 'rxjs';
import {
  KHIFileTextReference,
  LogAnnotationTypeResourceRef,
} from 'src/app/common/schema/khi-file-types';
import { SelectionManagerService } from 'src/app/services/selection-manager.service';
import { AnnotationDecider, DECISION_HIDDEN } from '../annotator';
import { LogEntry } from 'src/app/store/log';
import { InspectionDataStoreService } from 'src/app/services/inspection-data-store.service';
import { ToTextReferenceFromKHIFileBinary } from 'src/app/common/loader/reference-type';

interface ResourceRefAnnotationViewModel {
  label: string;
  path: string;
}

@Component({
  standalone: true,
  templateUrl: './relationship-annotator.component.html',
  styleUrl: './relationship-annotator.component.sass',
  imports: [CommonModule, MatIconModule],
})
export class RelationshipAnnotatorComponent {
  private readonly selectionManager = inject(SelectionManagerService);

  @Input()
  refs: Observable<ResourceRefAnnotationViewModel[]> = NEVER;

  currentSelectedTimelinePath = this.selectionManager.selectedTimeline.pipe(
    map((t) => t?.resourcePath ?? ''),
    shareReplay(1),
  );

  public selectResource(resourcePath: string) {
    this.selectionManager.onSelectTimeline(resourcePath);
  }

  public highlightResource(resourcePath: string) {
    this.selectionManager.onHighlightTimeline(resourcePath);
  }

  public static inputMapper: AnnotationDecider<LogEntry> = (
    l?: LogEntry | null,
  ) => {
    if (!l) {
      return DECISION_HIDDEN;
    }
    const dataStore = inject(InspectionDataStoreService);
    const pathReferences: KHIFileTextReference[] = [];
    for (const annotation of l.annotations) {
      if (annotation.type == LogAnnotationTypeResourceRef) {
        const pathReference = annotation['path'] as KHIFileTextReference;
        pathReferences.push(pathReference);
      }
    }
    if (pathReferences.length == 0) return DECISION_HIDDEN;
    return {
      inputs: {
        refs: of(pathReferences).pipe(
          withLatestFrom(
            dataStore.referenceResolver.pipe(filter((tb) => !!tb)),
          ),
          map(([refs, bufferLoader]) =>
            refs.map((ref) =>
              bufferLoader!.getText(ToTextReferenceFromKHIFileBinary(ref)),
            ),
          ),
          switchMap((refs) => forkJoin(refs)),
          map((refs) =>
            [...new Set(refs)].map((path) => {
              const splittedPath = path.split('#');
              const resourceRefLabel = `${splittedPath[splittedPath.length - 1]} of ${splittedPath[splittedPath.length - 2]}`;
              return {
                label: resourceRefLabel,
                path,
              } as ResourceRefAnnotationViewModel;
            }),
          ),
        ),
      },
    };
  };
}
