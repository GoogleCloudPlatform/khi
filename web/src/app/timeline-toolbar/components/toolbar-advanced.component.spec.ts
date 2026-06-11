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
    fixture.componentRef.setInput('timelineCelError', '');
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

  it('should emit drawDiagram when draw button is clicked', () => {
    let emitted = false;
    component.drawDiagram.subscribe(() => (emitted = true));

    fixture.componentRef.setInput('logOrTimelineNotSelected', false);
    fixture.detectChanges();

    const button = fixture.debugElement.query(
      By.css('button[mat-raised-button]'),
    );
    expect(button.nativeElement.disabled).toBeFalse();

    button.nativeElement.click();

    expect(emitted).toBeTrue();
  });

  it('should bind timeline CEL filter value correctly', () => {
    component.timelineCelFilter.set('timeline.name == "test"');
    fixture.detectChanges();

    const celInputs = fixture.debugElement.queryAll(
      By.css('khi-timeline-cel-input'),
    );
    expect(celInputs.length).toBe(2);
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
});
