/**
 * Copyright 2024 Google LLC
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
import { ToolbarComponent } from 'src/app/timeline-toolbar/components/toolbar.component';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTooltip } from '@angular/material/tooltip';
import { By } from '@angular/platform-browser';

describe('ToolbarComponent', () => {
  let component: ToolbarComponent;
  let fixture: ComponentFixture<ToolbarComponent>;
  let snackBarSpy: jasmine.SpyObj<MatSnackBar>;

  beforeEach(async () => {
    snackBarSpy = jasmine.createSpyObj('MatSnackBar', ['open']);

    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule, ToolbarComponent],
      providers: [{ provide: MatSnackBar, useValue: snackBarSpy }],
    }).compileComponents();

    fixture = TestBed.createComponent(ToolbarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have default input values', () => {
    expect(component.showButtonLabel()).toBeFalse();
    expect(component.logOrTimelineNotSelected()).toBeTrue();
    expect(component.timezoneShift()).toBe(0);
  });

  it('should emit switchToAdvanced when advanced button is clicked', () => {
    let emitted = false;
    component.switchToAdvanced.subscribe(() => (emitted = true));

    const buttons = fixture.debugElement.queryAll(
      By.css('button[mat-icon-button]'),
    );
    const advancedButton = buttons.find(
      (btn) =>
        btn.nativeElement.getAttribute('matTooltip') ===
        'Switch to advanced filter mode',
    );
    expect(advancedButton).toBeTruthy();

    advancedButton!.nativeElement.click();
    expect(emitted).toBeTrue();
  });

  it('should render the filter badges for provided filters', () => {
    fixture.componentRef.setInput('timelineFilters', [
      {
        id: '1',
        timelineType: 'K8sResource',
        mode: 'regex',
        value: 'Pod',
        action: 'include',
      },
      {
        id: '2',
        timelineType: '*',
        mode: 'regex',
        value: 'test-pattern',
        action: 'include',
      },
    ]);
    fixture.detectChanges();

    const badges = fixture.debugElement.queryAll(By.css('.filter-badge'));
    expect(badges.length).toBe(2);
    expect(badges[0].nativeElement.textContent).toContain('K8sResource: Pod');
    expect(badges[1].nativeElement.textContent).toContain('*: test-pattern');
  });

  it('should toggle filter builder popover when add button is clicked', () => {
    const addButton = fixture.debugElement.query(By.css('.add-filter-btn'));
    expect(addButton).toBeTruthy();
    expect(component['isFilterBuilderOpen']()).toBeFalse();

    addButton.nativeElement.click();
    fixture.detectChanges();

    expect(component['isFilterBuilderOpen']()).toBeTrue();
  });

  it('should delete the filter when delete icon in a badge is clicked', () => {
    fixture.componentRef.setInput('timelineFilters', [
      {
        id: '1',
        timelineType: 'K8sResource',
        mode: 'selection',
        value: 'Pod',
        action: 'include',
      },
    ]);
    fixture.detectChanges();

    const deleteIcon = fixture.debugElement.query(
      By.css('.filter-badge .delete-icon'),
    );
    expect(deleteIcon).toBeTruthy();

    deleteIcon.nativeElement.click();
    fixture.detectChanges();

    expect(component.timelineFilters().length).toBe(0);
  });

  it('should bind logSearchTerms to khi-chip-search-bar', () => {
    fixture.componentRef.setInput('logSearchTerms', ['test-query']);
    fixture.detectChanges();

    const chipSearchBar = fixture.debugElement.query(
      By.css('khi-chip-search-bar'),
    );
    expect(chipSearchBar).toBeTruthy();
    expect(component.logSearchTerms()).toEqual(['test-query']);
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

  it('should render time range badge with two lines when timeRangeFilter is set', () => {
    // 2023-11-15 07:00:00 JST (+9) -> 1699999200000000000n
    // 2023-11-15 09:30:00 JST (+9) -> 1700008200000000000n
    fixture.componentRef.setInput('timezoneShift', 9);
    fixture.componentRef.setInput('timeRangeFilter', {
      startTime: 1699999200000000000n,
      endTime: 1700008200000000000n,
    });
    fixture.detectChanges();

    const timeBadge = fixture.debugElement.query(By.css('.time-range-badge'));
    expect(timeBadge).toBeTruthy();
    const lines = timeBadge.queryAll(By.css('.time-range-line'));
    expect(lines.length).toBe(2);
    expect(lines[0].nativeElement.textContent.trim()).toBe(
      '2023-11-15 07:00:00',
    );
    expect(lines[1].nativeElement.textContent.trim()).toBe('~ 09:30:00');
  });

  it('should set detailed tooltip on time range badge when timeRangeFilter is set', () => {
    fixture.componentRef.setInput('timezoneShift', 9);
    fixture.componentRef.setInput('timeRangeFilter', {
      startTime: 1699999200000000000n,
      endTime: 1700008200000000000n,
    });
    fixture.detectChanges();

    const timeBadge = fixture.debugElement.query(By.css('.time-range-badge'));
    expect(timeBadge).toBeTruthy();
    const tooltip = timeBadge.injector.get(MatTooltip);
    expect(tooltip.message).toBe(
      '2023-11-15T07:00:00.000+09:00 ~ 2023-11-15T09:30:00.000+09:00 (2h30m)',
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
