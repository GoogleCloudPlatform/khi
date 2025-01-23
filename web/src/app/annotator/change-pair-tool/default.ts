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
import {
  CommonToolbarButtonComponent,
  CommonToolbarButtonInput,
} from '../common-toolbar-button.component';
import { inject } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';
import { ChangePairAnnotatorResolver } from './resolver';
import { SelectionManagerService } from 'src/app/services/selection-manager.service';
import { map, of, withLatestFrom } from 'rxjs';
import {
  ResourceRevisionChangePair,
  TimelineLayer,
} from 'src/app/store/timeline';

function copyRevisionManifestContentAnnotationDecider(): AnnotationDecider<
  ResourceRevisionChangePair,
  CommonToolbarButtonInput
> {
  const tooltip = 'Copy the manifest to clipboard';
  const icon = 'content_paste';
  return (l) => {
    if (!l)
      return CommonToolbarButtonComponent.disabledAnnotationDecision(
        icon,
        tooltip,
      );
    const snackBar = inject(MatSnackBar);
    const clipboard = inject(Clipboard);
    return {
      inputs: {
        icon,
        tooltip,
        disabled: false,
        onClick: () => {
          let snackbarMessage = 'Copy failed';
          if (clipboard.copy(l.current.resourceContent)) {
            snackbarMessage = 'Copied!';
          }
          snackBar.open(snackbarMessage, undefined, { duration: 1000 });
        },
      },
    };
  };
}

function floatDiffViewButtonAnnotationDecider(): AnnotationDecider<
  ResourceRevisionChangePair,
  CommonToolbarButtonInput
> {
  const tooltip = 'Show the change on another tab';
  const icon = 'open_in_new';
  return (changePair) => {
    if (!changePair) {
      return CommonToolbarButtonComponent.disabledAnnotationDecision(
        icon,
        tooltip,
      );
    }
    const selectionManager = inject(SelectionManagerService);
    return {
      inputs: {
        icon,
        tooltip,
        disabled: false,
        onClick: () => {
          of(changePair.current)
            .pipe(
              withLatestFrom(selectionManager.selectedTimeline),
              map(([current, timeline]) => {
                if (!timeline) {
                  return '';
                }
                const kind = timeline.getNameOfLayer(TimelineLayer.Kind);
                const namespace = timeline.getNameOfLayer(
                  TimelineLayer.Namespace,
                );
                const name = timeline.getNameOfLayer(TimelineLayer.Name);
                let subresource = timeline.getNameOfLayer(
                  TimelineLayer.Subresource,
                );
                if (subresource == '') subresource = '-';
                return `/diff/${kind}/${namespace}/${name}/${subresource}?logIndex=${current.logIndex}`;
              }),
            )
            .subscribe((path) => {
              if (path !== '') {
                window.open(window.location.pathname + path, '_blank');
              }
            });
        },
      },
    };
  };
}

//Tool bar button set for non-floating diff page
export function getDefaultChangePairToolAnnotatorResolver(): ChangePairAnnotatorResolver {
  return new ChangePairAnnotatorResolver([
    new Annotator(
      CommonToolbarButtonComponent,
      copyRevisionManifestContentAnnotationDecider(),
    ),
    new Annotator(
      CommonToolbarButtonComponent,
      floatDiffViewButtonAnnotationDecider(),
    ),
  ]);
}

//Tool bar button set for floating diff page
export function getDefaultChangePairToolAnnotatorResolverForFloatingPage(): ChangePairAnnotatorResolver {
  return new ChangePairAnnotatorResolver([
    new Annotator(
      CommonToolbarButtonComponent,
      copyRevisionManifestContentAnnotationDecider(),
    ),
  ]);
}
