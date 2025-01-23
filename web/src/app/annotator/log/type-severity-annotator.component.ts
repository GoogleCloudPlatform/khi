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

import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { AnnotationDecider } from '../annotator';
import { LogTypeMetadata } from 'src/app/generated';
import { LogEntry } from 'src/app/store/log';

@Component({
  standalone: true,
  templateUrl: './type-severity-annotator.component.html',
  styleUrls: ['./type-severity-annotator.component.sass'],
  imports: [CommonModule, MatIconModule],
})
export class TypeSeverityAnnotatorComponent {
  @Input()
  logType = 'N/A';

  @Input()
  severity = 'N/A';

  public static inputMapper: AnnotationDecider<LogEntry> = (
    l?: LogEntry | null,
  ) => {
    let logType = 'N/A';
    if (l !== null && l !== undefined) {
      logType = LogTypeMetadata[l.logType].label;
    }
    return {
      inputs: {
        logType: logType,
        severity: l?.logSeverityLabel ?? 'N/A',
      },
    };
  };
}
