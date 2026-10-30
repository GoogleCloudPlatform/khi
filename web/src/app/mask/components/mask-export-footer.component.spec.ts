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
import { MaskExportFooterComponent } from 'src/app/mask/components/mask-export-footer.component';
import { MaskExportPhase } from 'src/app/mask/types/mask-viewmodel';

describe('MaskExportFooterComponent', () => {
  let component: MaskExportFooterComponent;
  let fixture: ComponentFixture<MaskExportFooterComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MaskExportFooterComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(MaskExportFooterComponent);
    component = fixture.componentInstance;

    fixture.componentRef.setInput('ruleCount', 2);
    fixture.componentRef.setInput('enabledRuleCount', 2);
    fixture.componentRef.setInput('isExporting', false);
    fixture.detectChanges();
  });

  it('should render footer export progress banner and percentage button label while exporting', () => {
    expect(component).toBeTruthy();
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
