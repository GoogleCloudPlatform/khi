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
import { ToolbarComponent } from './toolbar.component';
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

  it('should display the count of included kinds when showButtonLabel is true', () => {
    // Set inputs
    fixture.componentRef.setInput('showButtonLabel', true);
    fixture.componentRef.setInput('kinds', new Set(['pod', 'service', 'node']));
    component.includedKinds.set(new Set(['pod', 'service']));

    fixture.detectChanges();

    const element = fixture.debugElement.nativeElement;
    // The template renders something like "Kinds2/3" (without spacing in the indicator span)
    expect(element.textContent).toContain('Kinds');
    expect(element.textContent).toContain('2/3');
  });

  it('should emit drawDiagram when draw button is clicked', () => {
    let emitted = false;
    component.drawDiagram.subscribe(() => (emitted = true));

    // Set logOrTimelineNotSelected to false to enable the button
    fixture.componentRef.setInput('logOrTimelineNotSelected', false);
    fixture.detectChanges();

    const button = fixture.debugElement.query(
      By.css('button[mat-raised-button]'),
    );
    expect(button.nativeElement.disabled).toBeFalse();

    button.nativeElement.click();

    expect(emitted).toBeTrue();
  });

  it('should toggle hideSubresourcesWithoutMatchingLogs model when toggle in template is clicked', () => {
    const toggles = fixture.debugElement.queryAll(By.css('mat-button-toggle'));
    expect(toggles.length).toBe(2);

    const subresourceToggle = toggles[0];
    subresourceToggle.nativeElement.querySelector('button').click();
    fixture.detectChanges();

    // The state should be toggled
    expect(component.hideSubresourcesWithoutMatchingLogs()).toBeTrue();
  });
});
