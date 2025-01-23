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

import { TimelineLayer } from 'src/app/store/timeline';
import { Annotator } from '../annotator';
import { CommonFieldAnnotatorComponent } from '../common-field-annotator.component';
import { TimelineAnnotatorResolver } from './resolver';

export function getDefaultTimelineAnnotatorResolver(): TimelineAnnotatorResolver {
  return new TimelineAnnotatorResolver([
    new Annotator(
      CommonFieldAnnotatorComponent,
      CommonFieldAnnotatorComponent.inputMapperForTimelineEntry(
        'workspaces',
        'Kind',
        (tl) => tl.getNameOfLayer(TimelineLayer.Kind),
      ),
    ),
    new Annotator(
      CommonFieldAnnotatorComponent,
      CommonFieldAnnotatorComponent.inputMapperForTimelineEntry(
        'folder',
        'Namespace',
        (tl) => tl.getNameOfLayer(TimelineLayer.Namespace),
      ),
    ),
    new Annotator(
      CommonFieldAnnotatorComponent,
      CommonFieldAnnotatorComponent.inputMapperForTimelineEntry(
        'description',
        'Name',
        (tl) => tl.getNameOfLayer(TimelineLayer.Name),
      ),
    ),
    new Annotator(
      CommonFieldAnnotatorComponent,
      CommonFieldAnnotatorComponent.inputMapperForTimelineEntry(
        'page_info',
        'Subresource',
        (tl) => tl.getNameOfLayer(TimelineLayer.Subresource),
      ),
    ),
  ]);
}
