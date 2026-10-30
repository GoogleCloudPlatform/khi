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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MaskRuleListComponent } from 'src/app/mask/components/mask-rule-list.component';
import {
  RuleEnabledChangeEvent,
  RuleReplacementChangeEvent,
} from 'src/app/mask/types/mask-viewmodel';

describe('MaskRuleListComponent', () => {
  let component: MaskRuleListComponent;
  let fixture: ComponentFixture<MaskRuleListComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MaskRuleListComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(MaskRuleListComponent);
    component = fixture.componentInstance;

    fixture.componentRef.setInput('rules', [
      {
        target: 'secret-pod',
        replacement: '',
        defaultReplacement: '<masked_1>',
        isDeletion: true,
        isDefaultReplacement: false,
        enabled: true,
      },
    ]);
    fixture.detectChanges();
  });

  it('should render rule cards and emit rule actions', () => {
    const enabledEvents: RuleEnabledChangeEvent[] = [];
    const replacementEvents: RuleReplacementChangeEvent[] = [];
    const resetTargets: string[] = [];
    const removedTargets: string[] = [];

    component.ruleEnabledChange.subscribe((e) => enabledEvents.push(e));
    component.ruleReplacementChange.subscribe((e) => replacementEvents.push(e));
    component.ruleResetReplacement.subscribe((t) => resetTargets.push(t));
    component.ruleRemove.subscribe((t) => removedTargets.push(t));

    const root = fixture.nativeElement as HTMLElement;
    expect(root.querySelector('.rule-target')?.textContent).toContain(
      'secret-pod',
    );
    expect(root.querySelector('.delete-badge')?.textContent).toContain(
      '(delete)',
    );

    component['onRuleEnabledToggle']('secret-pod', false);
    component['onRuleReplacementInput']('secret-pod', '<masked_custom>');
    (root.querySelector('.rule-reset-btn') as HTMLButtonElement).click();
    (root.querySelector('.rule-remove-btn') as HTMLButtonElement).click();

    expect(enabledEvents).toEqual([{ target: 'secret-pod', enabled: false }]);
    expect(replacementEvents).toEqual([
      { target: 'secret-pod', replacement: '<masked_custom>' },
    ]);
    expect(resetTargets).toEqual(['secret-pod']);
    expect(removedTargets).toEqual(['secret-pod']);
  });
});
