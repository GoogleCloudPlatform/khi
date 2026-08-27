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

import { By } from '@angular/platform-browser';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatSliderThumb } from '@angular/material/slider';
import {
  GraphToolbarComponent,
  MAX_DELETION_THRESHOLD_SECONDS,
  MIN_DELETION_THRESHOLD_SECONDS,
} from './graph-toolbar.component';

describe('GraphToolbarComponent', () => {
  let component: GraphToolbarComponent;
  let fixture: ComponentFixture<GraphToolbarComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [GraphToolbarComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(GraphToolbarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    document
      .querySelectorAll('.cdk-overlay-container')
      .forEach((el) => el.remove());
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should format deletion threshold correctly and handle boundary values', () => {
    fixture.componentRef.setInput(
      'deletionThresholdSeconds',
      MIN_DELETION_THRESHOLD_SECONDS,
    );
    fixture.detectChanges();
    const label = fixture.nativeElement.querySelector('.slider-label');
    expect(label.textContent).toBe('Retention period: 10s');

    fixture.componentRef.setInput(
      'deletionThresholdSeconds',
      MAX_DELETION_THRESHOLD_SECONDS,
    );
    fixture.detectChanges();
    expect(label.textContent).toBe('Retention period: 3600s (60m)');
  });

  it('should update deletionThresholdSeconds when slider thumb emits valueChange', () => {
    const thumbDirective = fixture.debugElement
      .query(By.directive(MatSliderThumb))
      .injector.get(MatSliderThumb);
    thumbDirective.valueChange.emit(600);
    fixture.detectChanges();

    expect(component.deletionThresholdSeconds()).toBe(600);
    expect(
      fixture.nativeElement.querySelector('.slider-label').textContent,
    ).toBe('Retention period: 600s (10m)');
  });

  it('should display tooltip on slider label', () => {
    const label = fixture.nativeElement.querySelector('.slider-label');
    expect(
      label.getAttribute('matTooltip') ||
        label.getAttribute('ng-reflect-message'),
    ).toBe('Duration to keep deleted resources visible on the graph');
  });

  it('should propagate model changes when deletionThresholdSeconds is updated', () => {
    let emittedValue: number | undefined;
    component.deletionThresholdSeconds.subscribe((v) => {
      emittedValue = v;
    });
    component.deletionThresholdSeconds.set(300);
    expect(emittedValue).toBe(300);
  });

  it('should emit fitToView when fit button is clicked', () => {
    const spy = spyOn(component.fitToView, 'emit');
    const button = fixture.nativeElement.querySelector('button.fit-button');
    button.click();
    expect(spy).toHaveBeenCalled();
  });

  it('should disable controls when isLoading is true', () => {
    fixture.componentRef.setInput('isLoading', true);
    fixture.detectChanges();

    const thumbInput = fixture.nativeElement.querySelector(
      'input[matSliderThumb]',
    );
    const fitButton = fixture.nativeElement.querySelector('button.fit-button');
    const downloadButton = fixture.nativeElement.querySelector(
      'button.download-button',
    );

    expect(thumbInput.disabled).toBeTrue();
    expect(fitButton.disabled).toBeTrue();
    expect(downloadButton.disabled).toBeTrue();
  });

  it('should emit downloadSvg and downloadPng when menu items are clicked', () => {
    const svgSpy = spyOn(component.downloadSvg, 'emit');
    const pngSpy = spyOn(component.downloadPng, 'emit');

    const downloadButton = fixture.nativeElement.querySelector(
      'button.download-button',
    );
    downloadButton.click();
    fixture.detectChanges();

    const menuItems = document.querySelectorAll<HTMLButtonElement>(
      '.mat-mdc-menu-content button[mat-menu-item]',
    );
    expect(menuItems.length).toBe(2);

    menuItems[0].click();
    expect(svgSpy).toHaveBeenCalled();

    downloadButton.click();
    fixture.detectChanges();

    const updatedMenuItems = document.querySelectorAll<HTMLButtonElement>(
      '.mat-mdc-menu-content button[mat-menu-item]',
    );
    updatedMenuItems[1].click();
    expect(pngSpy).toHaveBeenCalled();
  });
});
