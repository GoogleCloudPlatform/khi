/**
 * Copyright 2026 Google LLC
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

import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';
import { LogStore } from 'src/app/store/domain/log-store';
import { TimelineStore } from 'src/app/store/domain/timeline-store';
import { StyleProvider } from 'src/app/store/domain/style';
import {
  CELTimelineFilterEnvironment,
  CELLogFilterEnvironment,
} from 'src/app/store/domain/filter/cel-env';

export class SearchWorkerState {
  styleStore: StyleProvider | null = null;
  internPoolStore: InternPoolStore | null = null;
  logStore: LogStore | null = null;
  timelineStore: TimelineStore | null = null;

  readonly timelineCelEnv = new CELTimelineFilterEnvironment();
  readonly logCelEnv = new CELLogFilterEnvironment();
}

export const searchWorkerState = new SearchWorkerState();
