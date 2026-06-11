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
import { LogTypeOverrideListComponent } from 'src/app/dialogs/style-override/components/log-type-override-list.component';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { LogTypeOverrideEvent } from 'src/app/dialogs/style-override/types/style-override-viewmodel';

describe('LogTypeOverrideListComponent', () => {
  let component: LogTypeOverrideListComponent;
  let fixture: ComponentFixture<LogTypeOverrideListComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LogTypeOverrideListComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(LogTypeOverrideListComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.componentRef.setInput('logTypes', []);
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should list log types', () => {
    fixture.componentRef.setInput('logTypes', [
      {
        id: 1,
        label: 'Audit',
        description: 'Audit log events',
        hexColor: '#3f51b5',
        hexForegroundColor: '#ffffff',
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    const valueEl = fixture.debugElement.query(
      By.css('.log-type-preview .value'),
    );
    expect(valueEl.nativeElement.innerText).toContain('Audit');
  });

  it('should emit propertyChange on color input', () => {
    fixture.componentRef.setInput('logTypes', [
      {
        id: 1,
        label: 'Audit',
        description: 'Audit log events',
        hexColor: '#3f51b5',
        hexForegroundColor: '#ffffff',
        isOverridden: false,
        goColorCode: 'style.Color{R: 0.247, G: 0.318, B: 0.710, A: 1.0}',
      },
    ]);
    fixture.detectChanges();

    let emitted: LogTypeOverrideEvent | undefined;
    component.propertyChange.subscribe((event) => (emitted = event));

    const colorInput = fixture.debugElement.query(
      By.css('.color-picker-input'),
    );
    colorInput.nativeElement.value = '#ff0000';
    colorInput.nativeElement.dispatchEvent(new Event('input'));

    expect(emitted).toEqual({ id: 1, backgroundColor: '#ff0000' });
  });
});
