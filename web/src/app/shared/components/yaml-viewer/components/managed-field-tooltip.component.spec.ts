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

import { ManagedFieldTooltipComponent } from 'src/app/shared/components/yaml-viewer/components/managed-field-tooltip.component';

describe('ManagedFieldTooltipComponent', () => {
  let component: ManagedFieldTooltipComponent;
  let fixture: ComponentFixture<ManagedFieldTooltipComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ManagedFieldTooltipComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ManagedFieldTooltipComponent);
    component = fixture.componentInstance;

    fixture.componentRef.setInput('manager', 'test-manager');
    fixture.componentRef.setInput('time', 1609459200000000000n);
    fixture.componentRef.setInput('timezoneShift', 9);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
