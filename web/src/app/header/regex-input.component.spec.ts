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

import { OverlayModule } from '@angular/cdk/overlay';
import { HarnessLoader } from '@angular/cdk/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { MatFormFieldHarness } from '@angular/material/form-field/testing';
import { RegexInputComponent } from './regex-input.component';
import { MatInputHarness } from '@angular/material/input/testing';

describe('RegexFilterFormComponent', () => {
  let component: RegexInputComponent;
  let fixture: ComponentFixture<RegexInputComponent>;
  let loader: HarnessLoader;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [RegexInputComponent],
      imports: [
        NoopAnimationsModule,
        MatInputModule,
        OverlayModule,
        MatFormFieldModule,
        FormsModule,
        ReactiveFormsModule,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RegexInputComponent);
    component = fixture.componentInstance;
    loader = TestbedHarnessEnvironment.loader(fixture);
    fixture.detectChanges();
  });

  it('should show error when regex is invalid', async () => {
    expect(component).toBeTruthy();

    const formField = await loader.getHarness(MatFormFieldHarness);
    const matInput = await loader.getHarness(MatInputHarness);
    await matInput.setValue('a(');
    await matInput.blur();
    fixture.detectChanges();
    expect(await formField.hasErrors()).toBe(true);
  });

  it('should not show any error when regex is valid', async () => {
    expect(component).toBeTruthy();

    const formField = await loader.getHarness(MatFormFieldHarness);
    const matInput = await loader.getHarness(MatInputHarness);
    await matInput.setValue('a()');
    await matInput.blur();
    fixture.detectChanges();
    expect(await formField.hasErrors()).toBe(false);
  });
});
