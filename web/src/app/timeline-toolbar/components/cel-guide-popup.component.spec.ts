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
import {
  CelGuidePopupComponent,
  CelGuideTab,
} from './cel-guide-popup.component';

describe('CelGuidePopupComponent', () => {
  let component: CelGuidePopupComponent;
  let fixture: ComponentFixture<CelGuidePopupComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CelGuidePopupComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(CelGuidePopupComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  it('should default active tab to Overview', () => {
    expect(component.activeTab()).toBe(CelGuideTab.Overview);
  });

  it('should switch active tab when button is clicked', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.tab-button');
    expect(buttons.length).toBe(3);

    // Click Log CEL tab
    buttons[2].click();
    fixture.detectChanges();
    expect(component.activeTab()).toBe(CelGuideTab.LogCel);
  });
});
