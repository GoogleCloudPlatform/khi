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
import { MatProgressBar } from '@angular/material/progress-bar';
import { InspectionType } from 'src/app/common/schema/api-types';
import { InspectionTypeSelectorComponent } from 'src/app/dialogs/new-inspection/components/inspection-type-selector.component';

describe('InspectionTypeSelectorComponent', () => {
  let fixture: ComponentFixture<InspectionTypeSelectorComponent>;

  const mockInspectionTypes: readonly InspectionType[] = [
    {
      id: 'gke',
      name: 'Google Kubernetes Engine',
      description: 'Inspect GKE cluster logs\nand metrics.',
      icon: 'assets/gke.svg',
    },
    {
      id: 'composer',
      name: 'Cloud Composer',
      description: 'Inspect Cloud Composer environment.',
      icon: 'assets/composer.svg',
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule, InspectionTypeSelectorComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(InspectionTypeSelectorComponent);
  });

  it('should render loading indicator when inspectionTypes is null', () => {
    fixture.componentRef.setInput('inspectionTypes', null);
    fixture.detectChanges();

    const loadingText = fixture.debugElement.query(By.css('p'));
    expect(loadingText).toBeTruthy();
    expect(loadingText.nativeElement.textContent).toContain(
      'Loading list of inspection types...',
    );

    const progressBar = fixture.debugElement.query(
      By.directive(MatProgressBar),
    );
    expect(progressBar).toBeTruthy();

    const cards = fixture.debugElement.queryAll(By.css('.inspection-type'));
    expect(cards.length).toBe(0);
  });

  it('should render inspection type cards when inspectionTypes is populated', () => {
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.detectChanges();

    const progressBar = fixture.debugElement.query(
      By.directive(MatProgressBar),
    );
    expect(progressBar).toBeNull();

    const cards = fixture.debugElement.queryAll(By.css('.inspection-type'));
    expect(cards.length).toBe(2);

    const titles = fixture.debugElement.queryAll(By.css('mat-card-title'));
    expect(titles[0].nativeElement.textContent).toContain(
      'Google Kubernetes Engine',
    );
    expect(titles[1].nativeElement.textContent).toContain('Cloud Composer');
  });

  it('should emit selectInspectionType when an inspection type card is clicked', () => {
    fixture.componentRef.setInput('inspectionTypes', mockInspectionTypes);
    fixture.detectChanges();

    let emittedType: InspectionType | undefined;
    fixture.componentInstance.selectInspectionType.subscribe(
      (type: InspectionType) => {
        emittedType = type;
      },
    );

    const cards = fixture.debugElement.queryAll(By.css('.inspection-type'));
    cards[0].nativeElement.click();

    expect(emittedType).toEqual(mockInspectionTypes[0]);
  });
});
