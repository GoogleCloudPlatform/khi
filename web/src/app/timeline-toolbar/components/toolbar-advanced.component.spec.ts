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
import { By } from '@angular/platform-browser';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ToolbarAdvancedComponent } from 'src/app/timeline-toolbar/components/toolbar-advanced.component';

describe('ToolbarAdvancedComponent', () => {
  let component: ToolbarAdvancedComponent;
  let fixture: ComponentFixture<ToolbarAdvancedComponent>;
  let snackBarSpy: jasmine.SpyObj<MatSnackBar>;

  beforeEach(async () => {
    snackBarSpy = jasmine.createSpyObj('MatSnackBar', ['open']);

    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      providers: [{ provide: MatSnackBar, useValue: snackBarSpy }],
    }).compileComponents();

    fixture = TestBed.createComponent(ToolbarAdvancedComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('timelineIncludeCelError', '');
    fixture.componentRef.setInput('timelineExcludeCelError', '');
    fixture.componentRef.setInput('logCelError', '');
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have default input values', () => {
    expect(component.logOrTimelineNotSelected()).toBeTrue();
    expect(component.timezoneShift()).toBe(0);
  });

  it('should bind timeline CEL filter value correctly', () => {
    component.timelineIncludeCelFilter.set('timeline.name == "test"');
    component.timelineExcludeCelFilter.set(
      'timeline.namespace == "kube-system"',
    );
    fixture.detectChanges();

    const celInputs = fixture.debugElement.queryAll(
      By.css('khi-timeline-cel-input'),
    );
    expect(celInputs.length).toBe(3);
  });

  it('should emit switchToStandard when standard button is clicked', () => {
    let emitted = false;
    component.switchToStandard.subscribe(() => (emitted = true));

    const buttons = fixture.debugElement.queryAll(
      By.css('button[mat-icon-button]'),
    );
    const standardButton = buttons.find(
      (btn) =>
        btn.nativeElement.getAttribute('matTooltip') ===
        'Switch to Standard mode',
    );
    expect(standardButton).toBeTruthy();

    standardButton!.nativeElement.click();
    expect(emitted).toBeTrue();
  });

  it('should update timezoneShift when valid number committed', () => {
    const input = document.createElement('input');
    input.value = '5';
    const event = { target: input } as unknown as Event;

    component.onTimezoneshiftCommit(event);
    expect(component.timezoneShift()).toBe(5);
  });

  it('should default timezoneShift to 0 when invalid value committed', () => {
    const input = document.createElement('input');
    input.value = 'invalid';
    const event = { target: input } as unknown as Event;

    component.onTimezoneshiftCommit(event);
    expect(component.timezoneShift()).toBe(0);
  });

  it('should render time range button when timeRangeFilter is null', () => {
    fixture.componentRef.setInput('timeRangeFilter', null);
    fixture.detectChanges();

    const addTimeBtn = fixture.debugElement.query(
      By.css('.add-time-filter-btn'),
    );
    expect(addTimeBtn).toBeTruthy();
    expect(addTimeBtn.nativeElement.textContent).toContain('Time range');
  });

  it('should render time range badge when timeRangeFilter is set', () => {
    fixture.componentRef.setInput('timezoneShift', 9);
    fixture.componentRef.setInput('timeRangeFilter', {
      startTime: 1699999200000000000n,
      endTime: 1700008200000000000n,
    });
    fixture.detectChanges();

    const timeBadge = fixture.debugElement.query(By.css('.time-range-badge'));
    expect(timeBadge).toBeTruthy();
    expect(timeBadge.nativeElement.textContent).toContain(
      '2023-11-15 07:00:00 ~ 09:30:00',
    );
  });

  it('should clear timeRangeFilter when delete icon is clicked on time range badge', () => {
    fixture.componentRef.setInput('timeRangeFilter', {
      startTime: 1699999200000000000n,
      endTime: 1700008200000000000n,
    });
    fixture.detectChanges();

    const deleteIcon = fixture.debugElement.query(
      By.css('.time-range-badge .delete-icon'),
    );
    expect(deleteIcon).toBeTruthy();
    deleteIcon.nativeElement.click();
    fixture.detectChanges();

    expect(component.timeRangeFilter()).toBeNull();
  });
});
