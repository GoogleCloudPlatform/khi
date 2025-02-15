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

import { Annotator } from '../annotator';
import { CommonFieldAnnotatorComponent } from '../common-field-annotator.component';
import { ResourceReferenceListAnnotatorComponent } from './resource-reference-list.component';
import { LogAnnotatorResolver } from './resolver';
import { TypeSeverityAnnotatorComponent } from './type-severity-annotator.component';

export function getDefaultLogAnnotatorResolver(): LogAnnotatorResolver {
  return new LogAnnotatorResolver([
    new Annotator(
      TypeSeverityAnnotatorComponent,
      TypeSeverityAnnotatorComponent.inputMapper,
    ),
    new Annotator(
      CommonFieldAnnotatorComponent,
      CommonFieldAnnotatorComponent.inputMapperForTimestamp(
        'schedule',
        'Timestamp',
      ),
    ),
    new Annotator(
      CommonFieldAnnotatorComponent,
      CommonFieldAnnotatorComponent.annotationDeciderForLogBodyField(
        'fingerprint',
        'InsertId',
        (l) => l['insertId'],
      ),
    ),
    new Annotator(
      CommonFieldAnnotatorComponent,
      CommonFieldAnnotatorComponent.inputMapperForSummary(
        'summarize',
        'Summary',
      ),
    ),
    new Annotator(
      ResourceReferenceListAnnotatorComponent,
      ResourceReferenceListAnnotatorComponent.inputMapper,
    ),
  ]);
}
