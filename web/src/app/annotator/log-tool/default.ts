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

import { Annotator, AnnotationDecider } from '../annotator';
import { CommonToolbarButtonComponent } from '../common-toolbar-button.component';
import { LogAnnotatorResolver } from '../log/resolver';
import { inject } from '@angular/core';
import { InspectionDataStoreService } from 'src/app/services/inspection-data-store.service';
import { filter, map, of, switchMap, withLatestFrom } from 'rxjs';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';
import * as jsyaml from 'js-yaml';
import { LogEntry } from 'src/app/store/log';

function copyLogEntryContentMapper(
  toolTip: string,
): AnnotationDecider<LogEntry> {
  return (l) => {
    if (!l) {
      return {
        inputs: {
          icon: 'content_paste',
          tooltip: toolTip,
          disabled: true,
          onClick: () => ({}),
        },
      };
    }
    const snackBar = inject(MatSnackBar);
    const clipboard = inject(Clipboard);
    const dataStore = inject(InspectionDataStoreService);
    return {
      inputs: {
        icon: 'content_paste',
        tooltip: toolTip,
        disabled: l && l.logIndex < 0,
        onClick: () => {
          of(l.body)
            .pipe(
              withLatestFrom(
                dataStore.referenceResolver.pipe(filter((tb) => !!tb)),
              ),
              switchMap(([lr, tbs]) => tbs!.getText(lr)),
              map((text) => {
                if (clipboard.copy(text)) {
                  return 'Copied!';
                } else {
                  return 'Copy failed';
                }
              }),
            )
            .subscribe((copiedMessage) => {
              snackBar.open(copiedMessage, undefined, { duration: 1000 });
            });
        },
      },
    };
  };
}

function copyLogQueryContentMapper(
  toolTip: string,
): AnnotationDecider<LogEntry> {
  return (l) => {
    if (!l) {
      return {
        // TODO: think better icon later
        inputs: {
          icon: 'markdown_paste',
          tooltip: toolTip,
          disabled: true,
          onClick: () => ({}),
        },
      };
    }
    const snackBar = inject(MatSnackBar);
    const clipboard = inject(Clipboard);
    const dataStore = inject(InspectionDataStoreService);
    return {
      inputs: {
        icon: 'markdown_paste',
        tooltip: toolTip,
        disabled: l && l.logIndex < 0,
        onClick: () => {
          of(l)
            .pipe(
              withLatestFrom(
                dataStore.referenceResolver.pipe(filter((tb) => !!tb)),
              ),
              switchMap(([l, source]) => source!.getText(l.body)),
              map((logBody) => {
                const parsedLog = jsyaml.load(logBody) as {
                  [key: string]: string;
                };
                const timestamp = parsedLog['timestamp'];
                return `(
-- Log query for "${l.summary}"
insertId="${l.insertId}"
timestamp="${timestamp}"
)`;
              }),
            )
            .subscribe((query) => {
              let snackbarMessage = 'Copy failed';
              if (clipboard.copy(query)) {
                snackbarMessage = 'Copied!';
              }
              snackBar.open(snackbarMessage, undefined, { duration: 1000 });
            });
        },
      },
    };
  };
}
export function getDefaultLogToolAnnotatorResolver(): LogAnnotatorResolver {
  return new LogAnnotatorResolver([
    new Annotator(
      CommonToolbarButtonComponent,
      copyLogEntryContentMapper('Copy the log to clipboard'),
    ),
    new Annotator(
      CommonToolbarButtonComponent,
      copyLogQueryContentMapper('Copy query to clipboard'),
    ),
  ]);
}
