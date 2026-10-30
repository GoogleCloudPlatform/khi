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
import { MaskIndexBannerComponent } from 'src/app/mask/components/mask-index-banner.component';
import { StringPoolIndexPhase } from 'src/app/mask/types/mask-viewmodel';

describe('MaskIndexBannerComponent', () => {
  let component: MaskIndexBannerComponent;
  let fixture: ComponentFixture<MaskIndexBannerComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MaskIndexBannerComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(MaskIndexBannerComponent);
    component = fixture.componentInstance;
  });

  it('should render building progress percentage and message', () => {
    fixture.componentRef.setInput('indexStatus', {
      phase: StringPoolIndexPhase.Building,
      progressPercentage: 62.4,
      message: 'Building string pool trigram index...',
    });
    fixture.detectChanges();

    expect(component).toBeTruthy();
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Building string search index...');
    expect(text).toContain('62%');
    expect(text).toContain('Building string pool trigram index...');
  });

  it('should render ready state banner', () => {
    fixture.componentRef.setInput('indexStatus', {
      phase: StringPoolIndexPhase.Ready,
      progressPercentage: 100,
      message: 'Ready',
    });
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Trigram string search index ready');
  });

  it('should render fallback message when index build fails', () => {
    fixture.componentRef.setInput('indexStatus', {
      phase: StringPoolIndexPhase.Failed,
      progressPercentage: 0,
      message: 'Failed',
    });
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain(
      'Trigram index unavailable; using linear scan fallback.',
    );
  });
});
