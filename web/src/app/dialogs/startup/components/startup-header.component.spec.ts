/**
 * Copyright 2025 Google LLC
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
import { StartupHeaderComponent } from './startup-header.component';
import { By } from '@angular/platform-browser';
import { MatTooltipModule } from '@angular/material/tooltip';
// Use a mock or a simplified version if IconRegistryModule has side effects,
// but for now strict unit tests might just mock the underlying registry if needed.
// Since IconRegistryModule is simple, we might just include it or mock the registry.
import { MatIconTestingModule } from '@angular/material/icon/testing';

describe('StartupHeaderComponent', () => {
  let component: StartupHeaderComponent;
  let fixture: ComponentFixture<StartupHeaderComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StartupHeaderComponent, MatIconTestingModule, MatTooltipModule],
    }).compileComponents();

    fixture = TestBed.createComponent(StartupHeaderComponent);
    component = fixture.componentInstance;

    // Set required inputs
    fixture.componentRef.setInput('version', 'v1.0.0');
    fixture.componentRef.setInput('isViewerMode', false);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display version', () => {
    const versionEl = fixture.debugElement.query(By.css('.version'));
    expect(versionEl.nativeElement.textContent).toContain('v1.0.0');
  });

  it('should emit open event when open button is clicked', () => {
    spyOn(component.open, 'emit');
    const button = fixture.debugElement.query(By.css('.open-button'));
    button.nativeElement.click();
    expect(component.open.emit).toHaveBeenCalled();
  });

  it('should emit newInspection event when new inspection button is clicked', () => {
    spyOn(component.newInspection, 'emit');
    const button = fixture.debugElement.query(By.css('.new-inspection-button'));
    button.nativeElement.click();
    expect(component.newInspection.emit).toHaveBeenCalled();
  });

  it('should disable new inspection button in viewer mode', () => {
    fixture.componentRef.setInput('isViewerMode', true);
    fixture.detectChanges();
    const button = fixture.debugElement.query(By.css('.new-inspection-button'));
    expect(button.nativeElement.disabled).toBeTrue();
  });
});
