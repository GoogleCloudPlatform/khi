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

import { CommonModule } from '@angular/common';
import { Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import {
  MaskRuleItemViewModel,
  RuleEnabledChangeEvent,
  RuleReplacementChangeEvent,
} from 'src/app/mask/types/mask-viewmodel';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * Displays the list of configured string masking rules with enable toggles, replacement editors, and rule actions.
 */
@Component({
  selector: 'khi-mask-rule-list',
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatCheckboxModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './mask-rule-list.component.html',
  styleUrls: ['./mask-rule-list.component.scss'],
})
export class MaskRuleListComponent {
  /**
   * Configured masking rules displayed in the Selected Masks tab.
   */
  public readonly rules = input.required<readonly MaskRuleItemViewModel[]>();

  /**
   * Emits when a rule's enabled checkbox changes.
   */
  public readonly ruleEnabledChange = output<RuleEnabledChangeEvent>();

  /**
   * Emits when a rule's replacement string changes.
   */
  public readonly ruleReplacementChange = output<RuleReplacementChangeEvent>();

  /**
   * Emits when the user resets a rule's replacement back to its assigned `<masked_N>` placeholder.
   */
  public readonly ruleResetReplacement = output<string>();

  /**
   * Emits when a rule is removed from the rule list.
   */
  public readonly ruleRemove = output<string>();

  protected onRuleEnabledToggle(target: string, enabled: boolean): void {
    this.ruleEnabledChange.emit({ target, enabled });
  }

  protected onRuleReplacementInput(target: string, replacement: string): void {
    this.ruleReplacementChange.emit({ target, replacement });
  }
}
