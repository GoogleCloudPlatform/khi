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
import { ChangePairAnnotatorResolver } from '../change-pair-tool/resolver';
import { CommonWarningMessageComponent } from '../common-warning-message.component';

export function getDefaultChangePairAnnotatorResolver(): ChangePairAnnotatorResolver {
  return new ChangePairAnnotatorResolver([
    new Annotator(
      CommonWarningMessageComponent,
      CommonWarningMessageComponent.inputMapperForRevisionPair(
        'warning',
        () => 'This is a deletion request',
        (p) => p.current && p.current.isDeletion,
      ),
    ),
    new Annotator(
      CommonWarningMessageComponent,
      CommonWarningMessageComponent.inputMapperForRevisionPair(
        'warning',
        () => 'No change made by this request',
        (p) => p.previous?.resourceContent === p.current.resourceContent,
      ),
    ),
  ]);
}
