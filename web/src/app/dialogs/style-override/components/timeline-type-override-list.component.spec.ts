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
import { TimelineTypeOverrideListComponent } from './timeline-type-override-list.component';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { TimelineTypeOverrideEvent } from 'src/app/dialogs/style-override/types/style-override-viewmodel';

describe('TimelineTypeOverrideListComponent', () => {
  let component: TimelineTypeOverrideListComponent;
  let fixture: ComponentFixture<TimelineTypeOverrideListComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimelineTypeOverrideListComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(TimelineTypeOverrideListComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.componentRef.setInput('timelineTypes', []);
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should list timeline types', () => {
    fixture.componentRef.setInput('timelineTypes', [
      {
        id: 1,
        label: 'Pod',
        icon: 'pod',
        description: 'Pod type',
        hexColor: '#3f51b5',
        hexForegroundColor: '#ffffff',
        hexChipBackgroundColor: '#1a237e',
        height: 1.2,
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    const nameEl = fixture.debugElement.query(By.css('.main-name'));
    expect(nameEl.nativeElement.innerText).toContain('Pod');
  });

  it('should emit propertyChange on color input', () => {
    fixture.componentRef.setInput('timelineTypes', [
      {
        id: 1,
        label: 'Pod',
        icon: 'pod',
        description: 'Pod type',
        hexColor: '#3f51b5',
        hexForegroundColor: '#ffffff',
        hexChipBackgroundColor: '#1a237e',
        height: 1.2,
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    let emitted: TimelineTypeOverrideEvent | undefined;
    component.propertyChange.subscribe((event) => (emitted = event));

    const colorInput = fixture.debugElement.query(
      By.css('.color-picker-input'),
    );
    colorInput.nativeElement.value = '#ff0000';
    colorInput.nativeElement.dispatchEvent(new Event('input'));

    expect(emitted).toEqual({ id: 1, backgroundColor: '#ff0000' });
  });

  it('should emit propertyChange on height input change', () => {
    fixture.componentRef.setInput('timelineTypes', [
      {
        id: 1,
        label: 'Pod',
        icon: 'pod',
        description: 'Pod type',
        hexColor: '#3f51b5',
        hexForegroundColor: '#ffffff',
        hexChipBackgroundColor: '#1a237e',
        height: 1.2,
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    let emitted: TimelineTypeOverrideEvent | undefined;
    component.propertyChange.subscribe((event) => (emitted = event));

    const heightInput = fixture.debugElement.query(By.css('.height-input'));
    heightInput.nativeElement.value = '1.5';
    heightInput.nativeElement.dispatchEvent(new Event('change'));

    expect(emitted).toEqual({ id: 1, height: 1.5 });
  });
});
