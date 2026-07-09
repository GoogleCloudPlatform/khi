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

import { Revision } from 'src/app/store/domain/timeline';
import { RevisionFieldAnnotationProvider } from './revision-field-annotation.provider';
import { MutatingWebhookTooltipComponent } from 'src/app/shared/components/yaml-viewer/components/mutating-webhook-tooltip.component';

describe('RevisionFieldAnnotationProvider', () => {
  it('should return empty annotations if revision has no field annotations', () => {
    const revision = { fieldAnnotations: [] } as unknown as Revision;
    const provider = new RevisionFieldAnnotationProvider(revision);
    expect(provider.getAnnotations()).toEqual([]);
  });

  it('should generate annotations from revision field annotations', () => {
    const revision = {
      fieldAnnotations: [
        {
          fieldPath: '/metadata/annotations/cloud.google.com~1neg',
          mutatingWebhook: {
            configuration:
              'neg-annotation.config.common-webhooks.networking.gke.io',
            webhook: 'neg-annotation.common-webhooks.networking.gke.io',
            round: 0,
            index: 0,
          },
        },
      ],
    } as unknown as Revision;

    const provider = new RevisionFieldAnnotationProvider(revision);
    const annotations = provider.getAnnotations();

    expect(annotations.length).toBe(1);
    expect(annotations[0].path).toEqual([
      'metadata',
      'annotations',
      'cloud.google.com/neg',
    ]);
    expect(annotations[0].component).toBe(MutatingWebhookTooltipComponent);
    expect(annotations[0].inputs).toEqual({
      data: {
        configuration:
          'neg-annotation.config.common-webhooks.networking.gke.io',
        webhook: 'neg-annotation.common-webhooks.networking.gke.io',
        round: 0,
        index: 0,
      },
    });
  });
});
