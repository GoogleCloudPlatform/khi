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

import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
} from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { BreaklinePipe } from 'src/app/common/breakline.pipe';
import {
  CalloutParameterFormField,
  ParameterHintType,
} from 'src/app/common/schema/form-types';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Dumb component that renders a read-only callout banner inside the inspection parameter form.
 */
@Component({
  selector: 'khi-new-inspection-callout-parameter',
  templateUrl: './callout-parameter.component.html',
  styleUrl: './callout-parameter.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatIconModule,
    MatProgressSpinnerModule,
    KHIIconRegistrationModule,
    BreaklinePipe,
  ],
})
export class CalloutParameterComponent {
  /**
   * Exposes ParameterHintType enum to the template.
   */
  protected readonly ParameterHintType = ParameterHintType;

  /**
   * Configuration of the read-only callout banner field.
   */
  readonly parameter = input.required<CalloutParameterFormField>();

  /**
   * Resolves the effective severity of the callout, treating None or undefined as Info.
   */
  readonly severity = computed<ParameterHintType>(() => {
    const hintType = this.parameter().hintType;
    if (
      hintType === ParameterHintType.Error ||
      hintType === ParameterHintType.Warning
    ) {
      return hintType;
    }
    return ParameterHintType.Info;
  });

  /**
   * Resolves the Material icon name matching the callout severity.
   */
  readonly iconName = computed<string>(() => {
    switch (this.severity()) {
      case ParameterHintType.Error:
        return 'error';
      case ParameterHintType.Warning:
        return 'warning';
      default:
        return 'info';
    }
  });
}
