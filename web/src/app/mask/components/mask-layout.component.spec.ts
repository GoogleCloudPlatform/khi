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
import { MaskLayoutComponent } from 'src/app/mask/components/mask-layout.component';
import {
  MaskExportPhase,
  StringPoolIndexPhase,
} from 'src/app/mask/types/mask-viewmodel';

describe('MaskLayoutComponent', () => {
  let component: MaskLayoutComponent;
  let fixture: ComponentFixture<MaskLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MaskLayoutComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(MaskLayoutComponent);
    component = fixture.componentInstance;

    fixture.componentRef.setInput('isWorkbenchActive', true);
    fixture.componentRef.setInput('indexStatus', {
      phase: StringPoolIndexPhase.Building,
      progressPercentage: 62.4,
      message: 'Building string pool trigram index...',
    });
    fixture.componentRef.setInput('livePreviewEnabled', true);
    fixture.componentRef.setInput('searchQuery', '');
    fixture.componentRef.setInput('searchRegexError', '');
    fixture.componentRef.setInput('candidates', [
      { value: 'prod-secret-ns', selected: false, inRuleList: false },
    ]);
    fixture.componentRef.setInput('totalCandidateCount', 1);
    fixture.componentRef.setInput('isLoadingCandidates', false);
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
    fixture.componentRef.setInput('enabledRuleCount', 1);
    fixture.componentRef.setInput('isExporting', false);

    fixture.detectChanges();
  });

  it('should render building index progress banner', () => {
    expect(component).toBeTruthy();
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Building string search index...');
    expect(text).toContain('62%');
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

  it('should render footer export progress banner and percentage button label while exporting', () => {
    fixture.componentRef.setInput('isExporting', true);
    fixture.componentRef.setInput('exportProgress', {
      phase: MaskExportPhase.GeneratingArchive,
      phaseLabel: 'Generating masked .khi archive...',
      progressPercentage: 57.6,
      message: 'Masking string pool (16,384 / 28,400 strings)...',
    });
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const banner = root.querySelector('.export-progress-banner');
    expect(banner).toBeTruthy();
    expect(banner?.textContent).toContain('Generating masked .khi archive...');
    expect(banner?.textContent).toContain('58%');
    expect(banner?.textContent).toContain(
      'Masking string pool (16,384 / 28,400 strings)...',
    );
    expect(banner?.querySelector('mat-progress-bar')).toBeTruthy();

    const exportBtn = root.querySelector('.export-btn') as HTMLButtonElement;
    expect(exportBtn.disabled).toBeTrue();
    expect(exportBtn.textContent).toContain('Exporting... (58%)');
  });
});
