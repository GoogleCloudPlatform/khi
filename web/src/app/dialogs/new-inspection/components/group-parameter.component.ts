/**
 * Copyright 2025 Google LLC
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

import { Component, computed, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import {
  animate,
  state,
  style,
  transition,
  trigger,
} from '@angular/animations';
import {
  GroupParameterFormField,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { TextParameterComponent } from 'src/app/dialogs/new-inspection/components/text-parameter.component';
import { FileParameterComponent } from 'src/app/dialogs/new-inspection/components/file-parameter.component';
import { SetParameterComponent } from 'src/app/dialogs/new-inspection/components/set-parameter.component';
import { CheckboxParameterComponent } from 'src/app/dialogs/new-inspection/components/checkbox-parameter.component';
import { ParameterHeaderComponent } from 'src/app/dialogs/new-inspection/components/parameter-header.component';
import { ParameterHintComponent } from 'src/app/dialogs/new-inspection/components/parameter-hint.component';

/**
 * A collection of form fields.
 */
@Component({
  selector: 'khi-new-inspection-group-parameter',
  templateUrl: './group-parameter.component.html',
  styleUrls: ['./group-parameter.component.scss'],
  imports: [
    CommonModule,
    MatIconModule,
    KHIIconRegistrationModule,
    MatButtonModule,
    TextParameterComponent,
    FileParameterComponent,
    SetParameterComponent,
    CheckboxParameterComponent,
    ParameterHeaderComponent,
    ParameterHintComponent,
  ],
  animations: [
    trigger('children-animation', [
      state(
        'expanded',
        style({
          height: '*',
        }),
      ),
      state(
        'collapsed',
        style({
          height: '0',
        }),
      ),
      transition('expanded => collapsed', animate('150ms ease-in')),
      transition('collapsed => expanded', animate('150ms ease-out')),
    ]),
    trigger('expander-animation', [
      state(
        'expanded',
        style({
          transform: 'rotate(0deg)',
        }),
      ),
      state(
        'collapsed',
        style({
          transform: 'rotate(-90deg)',
        }),
      ),
      transition('expanded => collapsed', animate('150ms ease-in')),
      transition('collapsed => expanded', animate('150ms ease-out')),
    ]),
  ],
})
export class GroupParameterComponent {
  /**
   * Exposes ParameterInputType enum to the template.
   */
  protected readonly ParameterInputType = ParameterInputType;

  /**
   * The setting of this group type form field.
   */
  readonly parameter = input.required<GroupParameterFormField>();

  /**
   * If the children is collapsed or not. When it is null, user didn't click the expander to toggle yet.
   * When this is true, then the children is collapsed and hidden.
   */
  private readonly collapsedFromUserInput = signal<boolean | null>(null);

  /**
   * Computes the current expansion state of the group's children.
   */
  readonly childrenStatus = computed(() => {
    const fromUserInput = this.collapsedFromUserInput();
    const fromDefaultValue = this.parameter().collapsedByDefault;
    return (fromUserInput ?? fromDefaultValue) ? 'collapsed' : 'expanded';
  });

  /**
   * Toggles the collapsed status for children.
   */
  toggle(): void {
    this.collapsedFromUserInput.set(this.childrenStatus() !== 'collapsed');
  }
}
