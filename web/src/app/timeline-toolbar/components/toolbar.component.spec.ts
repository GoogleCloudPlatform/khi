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
});
