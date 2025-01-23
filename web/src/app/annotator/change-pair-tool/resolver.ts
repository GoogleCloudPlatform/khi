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

import { InjectionToken } from '@angular/core';
import { AnnotatorResolver } from '../annotator';
import { ResourceRevisionChangePair } from 'src/app/store/timeline';

export const CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER =
  new InjectionToken<ChangePairAnnotatorResolver>(
    'CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER',
  );

export const CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER =
  new InjectionToken<ChangePairAnnotatorResolver>(
    'CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER',
  );

export class ChangePairAnnotatorResolver extends AnnotatorResolver<ResourceRevisionChangePair> {}
