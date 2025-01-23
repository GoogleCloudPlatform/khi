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

import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { NEVER, Observable, of } from 'rxjs';
import { AnnotationDecider, DECISION_HIDDEN } from './annotator';
import { ResourceRevisionChangePair } from '../store/timeline';
@Component({
  standalone: true,
  imports: [CommonModule, MatIconModule],
  templateUrl: './common-warning-message.component.html',
  styleUrl: './common-warning-message.component.sass',
})
export class CommonWarningMessageComponent {
  @Input()
  icon = 'warm';

  @Input()
  message: Observable<string> = NEVER;

  public static inputMapperForRevisionPair(
    icon: string,
    messageMapper: (source: ResourceRevisionChangePair) => string,
    predicate: (source: ResourceRevisionChangePair) => boolean,
  ): AnnotationDecider<ResourceRevisionChangePair> {
    return (source) => {
      if (!source || !predicate(source)) return DECISION_HIDDEN;
      return {
        inputs: {
          icon,
          message: of(messageMapper(source!)),
        },
      };
    };
  }
}
