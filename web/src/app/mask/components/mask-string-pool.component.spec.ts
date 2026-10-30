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
import { MaskStringPoolComponent } from 'src/app/mask/components/mask-string-pool.component';

describe('MaskStringPoolComponent', () => {
  let component: MaskStringPoolComponent;
  let fixture: ComponentFixture<MaskStringPoolComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MaskStringPoolComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(MaskStringPoolComponent);
    component = fixture.componentInstance;

    fixture.componentRef.setInput('searchQuery', '');
    fixture.componentRef.setInput('searchRegexError', '');
    fixture.componentRef.setInput('candidates', [
      { value: 'test-secret-ns', selected: false, inRuleList: false },
    ]);
    fixture.componentRef.setInput('totalCandidateCount', 1);
    fixture.componentRef.setInput('isLoadingCandidates', false);

    fixture.detectChanges();
  });

  it('should emit addCustomSubstring and clear draft input on submit', () => {
    const emitted: string[] = [];
    component.addCustomSubstring.subscribe((target) => emitted.push(target));

    component['onCustomSubstringInput']('customer-id-42');
    component['onSubmitCustomSubstring']();

    expect(emitted).toEqual(['customer-id-42']);
    expect(component['customSubstringDraft']()).toBe('');
  });

  it('should render inline regex errors and emit selectShownCandidates', () => {
    let selectShownCount = 0;
    component.selectShownCandidates.subscribe(() => selectShownCount++);

    const root = fixture.nativeElement as HTMLElement;
    fixture.componentRef.setInput(
      'searchRegexError',
      'invalid regex query: missing closing ]',
    );
    fixture.detectChanges();

    expect(root.querySelector('.search-regex-error')?.textContent).toContain(
      'missing closing ]',
    );

    const selectShownBtn = root.querySelector(
      '.select-shown-btn',
    ) as HTMLButtonElement;
    expect(selectShownBtn.textContent).toContain('Select shown (1)');
    expect(selectShownBtn.disabled).toBeFalse();
    selectShownBtn.click();
    expect(selectShownCount).toBe(1);
  });
});
