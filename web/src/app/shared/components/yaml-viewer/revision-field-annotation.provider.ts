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

import { Type } from '@angular/core';
import { Revision } from 'src/app/store/domain/timeline';
import {
  AnnotationSeverity,
  YamlAnnotationProvider,
  YamlFieldAnnotation,
} from './yaml-annotation';
import { MutatingWebhookTooltipComponent } from 'src/app/shared/components/yaml-viewer/components/mutating-webhook-tooltip.component';
import { convertJsonPatchPathToArray } from './diff-util';

import { ReadonlyDomainElement } from 'src/app/store/domain/types';

/**
 * Provides field annotations defined within a Revision.
 */
export class RevisionFieldAnnotationProvider implements YamlAnnotationProvider {
  /**
   * @param _revision The revision from which field annotations are extracted.
   */
  constructor(private readonly _revision: ReadonlyDomainElement<Revision>) {}

  /**
   * Generates YamlFieldAnnotations based on the FieldAnnotations in the given revision.
   */
  public getAnnotations(): YamlFieldAnnotation[] {
    const annotations: YamlFieldAnnotation[] = [];
    if (!this._revision.fieldAnnotations) {
      return annotations;
    }

    for (const fa of this._revision.fieldAnnotations) {
      if (!fa.fieldPath) continue;

      const path = convertJsonPatchPathToArray(fa.fieldPath);
      if (fa.mutatingWebhook) {
        annotations.push({
          path,
          component: MutatingWebhookTooltipComponent as Type<unknown>,
          inputs: {
            data: {
              configuration: fa.mutatingWebhook.configuration,
              webhook: fa.mutatingWebhook.webhook,
              round: fa.mutatingWebhook.round,
              index: fa.mutatingWebhook.index,
            },
          },
          severity: AnnotationSeverity.Medium,
        });
      }
    }

    return annotations;
  }
}
