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
import { TimeRangeFilterBuilderComponent } from './time-range-filter-builder.component';
import { TimeRangeFilter } from 'src/app/services/view-state.service';

describe('TimeRangeFilterBuilderComponent', () => {
  let component: TimeRangeFilterBuilderComponent;
  let fixture: ComponentFixture<TimeRangeFilterBuilderComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimeRangeFilterBuilderComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(TimeRangeFilterBuilderComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  it('should initialize input texts from default timestamps if startTime and endTime are null', () => {
    // 2023-11-14T22:00:00Z -> 1699999200000000000n
    fixture.componentRef.setInput('defaultStartTime', 1699999200000000000n);
    // 2023-11-15T00:00:00Z -> 1700006400000000000n
    fixture.componentRef.setInput('defaultEndTime', 1700006400000000000n);
    fixture.componentRef.setInput('timezoneShift', 0);
    fixture.detectChanges();

    const startInput = fixture.nativeElement.querySelectorAll('input')[0];
    const endInput = fixture.nativeElement.querySelectorAll('input')[1];

    expect(startInput.value).toBe('2023-11-14T22:00:00.000+00:00');
    expect(endInput.value).toBe('2023-11-15T00:00:00.000+00:00');
  });

  it('should initialize input texts from startTime and endTime when provided', () => {
    fixture.componentRef.setInput('startTime', 1699999200000000000n);
    fixture.componentRef.setInput('endTime', 1700006400000000000n);
    fixture.componentRef.setInput('timezoneShift', 9);
    fixture.detectChanges();

    const startInput = fixture.nativeElement.querySelectorAll('input')[0];
    const endInput = fixture.nativeElement.querySelectorAll('input')[1];

    expect(startInput.value).toBe('2023-11-15T07:00:00.000+09:00');
    expect(endInput.value).toBe('2023-11-15T09:00:00.000+09:00');
  });

  it('should show error and disable apply button when start time is invalid format', () => {
    fixture.componentRef.setInput('startTime', 1699999200000000000n);
    fixture.componentRef.setInput('endTime', 1700006400000000000n);
    fixture.detectChanges();

    const startInput = fixture.nativeElement.querySelectorAll('input')[0];
    startInput.value = 'invalid-date';
    startInput.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    const errorEl = fixture.nativeElement.querySelector('.error-message');
    expect(errorEl).not.toBeNull();
    expect(errorEl.textContent).toContain('Invalid start time format.');

    const applyButton = fixture.nativeElement.querySelector(
      'button[type="submit"]',
    );
    expect(applyButton.disabled).toBeTrue();
  });

  it('should show error and disable apply button when start time is cleared', () => {
    fixture.componentRef.setInput('startTime', 1699999200000000000n);
    fixture.componentRef.setInput('endTime', 1700006400000000000n);
    fixture.detectChanges();

    const startInput = fixture.nativeElement.querySelectorAll('input')[0];
    startInput.value = '';
    startInput.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    const errorEl = fixture.nativeElement.querySelector('.error-message');
    expect(errorEl).not.toBeNull();
    expect(errorEl.textContent).toContain('Start and end time are required.');

    const applyButton = fixture.nativeElement.querySelector(
      'button[type="submit"]',
    );
    expect(applyButton.disabled).toBeTrue();
  });

  it('should show error when start time is after end time', () => {
    fixture.componentRef.setInput('startTime', 1700006400000000000n);
    fixture.componentRef.setInput('endTime', 1699999200000000000n);
    fixture.detectChanges();

    const errorEl = fixture.nativeElement.querySelector('.error-message');
    expect(errorEl).not.toBeNull();
    expect(errorEl.textContent).toContain(
      'Start time must be before or equal to end time.',
    );

    const applyButton = fixture.nativeElement.querySelector(
      'button[type="submit"]',
    );
    expect(applyButton.disabled).toBeTrue();
  });

  it('should emit confirm with parsed nanoseconds when Apply is clicked', () => {
    fixture.componentRef.setInput('startTime', 1699999200000000000n);
    fixture.componentRef.setInput('endTime', 1700006400000000000n);
    fixture.componentRef.setInput('timezoneShift', 0);
    fixture.detectChanges();

    let emittedRange: TimeRangeFilter | undefined;
    component.confirm.subscribe((val) => {
      emittedRange = val;
    });

    const form = fixture.nativeElement.querySelector('form');
    form.dispatchEvent(new Event('submit'));

    expect(emittedRange).toEqual({
      startTime: 1699999200000000000n,
      endTime: 1700006400000000000n,
    });
  });

  it('should emit deleteButtonClicked when delete button is clicked', () => {
    fixture.componentRef.setInput('showDeleteButton', true);
    fixture.detectChanges();

    let deleteClicked = false;
    component.deleteButtonClicked.subscribe(() => {
      deleteClicked = true;
    });

    const deleteBtn = fixture.nativeElement.querySelector('.delete-btn');
    expect(deleteBtn).not.toBeNull();
    deleteBtn.click();

    expect(deleteClicked).toBeTrue();
  });

  it('should emit closeButtonClicked when cancel button is clicked', () => {
    fixture.detectChanges();

    let cancelClicked = false;
    component.closeButtonClicked.subscribe(() => {
      cancelClicked = true;
    });

    const buttons = fixture.nativeElement.querySelectorAll('.actions button');
    const cancelBtn = buttons[0];
    cancelBtn.click();

    expect(cancelClicked).toBeTrue();
  });
});
