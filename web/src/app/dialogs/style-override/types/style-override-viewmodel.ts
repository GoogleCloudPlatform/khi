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

import { RevisionStateStyle } from 'src/app/store/domain/style';

/**
 * ViewModel for a revision state style override row.
 */
export interface RevisionStateStyleOverrideViewModel {
  readonly id: number;
  readonly label: string;
  readonly icon: string;
  readonly description: string;
  readonly hexColor: string;
  readonly isOverridden: boolean;
  readonly style: RevisionStateStyle;
  readonly goColorCode: string;
}

/**
 * ViewModel for a timeline type style override row.
 */
export interface TimelineTypeStyleOverrideViewModel {
  readonly id: number;
  readonly label: string;
  readonly icon: string;
  readonly description: string;
  readonly hexColor: string;
  readonly hexForegroundColor: string;
  readonly hexChipBackgroundColor: string;
  readonly hexChipForegroundColor: string;
  readonly height: number;
  readonly isOverridden: boolean;
  readonly goColorCode: string;
}

/**
 * Event payload structure when overriding a timeline type.
 */
export interface TimelineTypeOverrideEvent {
  readonly id: number;
  readonly backgroundColor?: string;
  readonly foregroundColor?: string;
  readonly typeChipBackgroundColor?: string;
  readonly typeChipForegroundColor?: string;
  readonly height?: number;
}

/**
 * ViewModel for a log type style override card.
 */
export interface LogTypeStyleOverrideViewModel {
  readonly id: number;
  readonly label: string;
  readonly description: string;
  readonly hexColor: string;
  readonly hexForegroundColor: string;
  readonly isOverridden: boolean;
  readonly goColorCode: string;
}

/**
 * Event payload structure when overriding a log type.
 */
export interface LogTypeOverrideEvent {
  readonly id: number;
  readonly backgroundColor?: string;
  readonly foregroundColor?: string;
}
