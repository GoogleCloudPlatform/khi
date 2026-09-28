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
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { InspectionFeature } from 'src/app/common/schema/api-types';
import { FeatureSelectorComponent } from 'src/app/dialogs/new-inspection/components/feature-selector.component';

describe('FeatureSelectorComponent', () => {
  let fixture: ComponentFixture<FeatureSelectorComponent>;

  const mockFeatures: readonly InspectionFeature[] = [
    {
      id: 'feature-audit',
      label: 'Kubernetes Audit Log',
      description: 'Ingest Kubernetes API audit logs.',
      enabled: true,
    },
    {
      id: 'feature-event',
      label: 'Kubernetes Event Log',
      description: 'Ingest Kubernetes events.',
      enabled: false,
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule, FeatureSelectorComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(FeatureSelectorComponent);
  });

  it('should render feature cards and apply selected class to enabled features', () => {
    fixture.componentRef.setInput('features', mockFeatures);
    fixture.detectChanges();

    const cards = fixture.debugElement.queryAll(By.css('.feature-card'));
    expect(cards.length).toBe(2);

    expect(cards[0].nativeElement.classList.contains('selected')).toBeTrue();
    const selectedIcon = cards[0].query(By.css('.selected-icon'));
    expect(selectedIcon).toBeTruthy();

    expect(cards[1].nativeElement.classList.contains('selected')).toBeFalse();
    const unselectedIcon = cards[1].query(By.css('.selected-icon'));
    expect(unselectedIcon).toBeNull();
  });

  it('should emit toggleFeature with feature id when a card is clicked', () => {
    fixture.componentRef.setInput('features', mockFeatures);
    fixture.detectChanges();

    let toggledFeatureId: string | undefined;
    fixture.componentInstance.toggleFeature.subscribe((id: string) => {
      toggledFeatureId = id;
    });

    const cards = fixture.debugElement.queryAll(By.css('.feature-card'));
    cards[1].nativeElement.click();

    expect(toggledFeatureId).toBe('feature-event');
  });

  it('should show error message and disable Next button when hasEnabledFeatures is false', () => {
    fixture.componentRef.setInput(
      'features',
      mockFeatures.map((f) => ({ ...f, enabled: false })),
    );
    fixture.detectChanges();

    const errorMsg = fixture.debugElement.query(
      By.css('.errmsg-no-selected-feature'),
    );
    expect(errorMsg).toBeTruthy();
    expect(errorMsg.nativeElement.textContent).toContain(
      'Select at least a feature',
    );

    const nextButton = fixture.debugElement.query(By.css('.next-button'));
    expect(nextButton.nativeElement.disabled).toBeTrue();
  });

  it('should enable Next button, hide error message, and emit nextStep when clicked while hasEnabledFeatures is true', () => {
    fixture.componentRef.setInput('features', mockFeatures);
    fixture.detectChanges();

    const errorMsg = fixture.debugElement.query(
      By.css('.errmsg-no-selected-feature'),
    );
    expect(errorMsg).toBeNull();

    const nextButton = fixture.debugElement.query(By.css('.next-button'));
    expect(nextButton.nativeElement.disabled).toBeFalse();

    let nextStepEmitted = false;
    fixture.componentInstance.nextStep.subscribe(() => {
      nextStepEmitted = true;
    });

    nextButton.nativeElement.click();
    expect(nextStepEmitted).toBeTrue();
  });
});
