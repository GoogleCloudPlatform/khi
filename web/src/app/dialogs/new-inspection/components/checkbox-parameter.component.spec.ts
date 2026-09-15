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

import { provideZoneChangeDetection, NgModule } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CheckboxParameterComponent } from './checkbox-parameter.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatIconRegistry } from '@angular/material/icon';
import {
  CheckboxParameterFormField,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { MatCheckboxHarness } from '@angular/material/checkbox/testing';
import { HarnessLoader } from '@angular/cdk/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from './service/parameter-store';
import {
  BrowserTestingModule,
  platformBrowserTesting,
} from '@angular/platform-browser/testing';

@NgModule({ providers: [provideZoneChangeDetection()] })
export class ZoneChangeDetectionModule {}

describe('CheckboxParameterComponent', () => {
  let fixture: ComponentFixture<CheckboxParameterComponent>;
  let harnessLoader: HarnessLoader;
  let parameterStore: DefaultParameterStore;

  const defaultParameter: CheckboxParameterFormField = {
    id: 'test-checkbox-id',
    type: ParameterInputType.Checkbox,
    label: 'test-label',
    default: false,
    description: 'A test checkbox parameter description.',
    hintType: ParameterHintType.None,
    hint: '',
    readonly: false,
  };

  beforeAll(() => {
    TestBed.resetTestEnvironment();
    TestBed.initTestEnvironment(
      [ZoneChangeDetectionModule, BrowserTestingModule],
      platformBrowserTesting(),
      { teardown: { destroyAfterEach: false } },
    );
  });

  beforeEach(async () => {
    parameterStore = new DefaultParameterStore();
    await TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule, CheckboxParameterComponent],
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: parameterStore,
        },
      ],
    }).compileComponents();
    const matIconRegistry = TestBed.inject(MatIconRegistry);
    matIconRegistry.setDefaultFontSetClass('material-symbols-outlined');
    fixture = TestBed.createComponent(CheckboxParameterComponent);
    fixture.componentRef.setInput('parameter', defaultParameter);
    parameterStore.setDefaultValues({
      'test-checkbox-id': false,
    });
    harnessLoader = TestbedHarnessEnvironment.loader(fixture);
  });

  afterEach(() => {
    parameterStore.destroy();
  });

  it('should create and show default unchecked state', async () => {
    fixture.detectChanges();

    expect(fixture.componentInstance).toBeTruthy();
    const matCheckbox = await harnessLoader.getHarness(MatCheckboxHarness);

    expect(await matCheckbox.isDisabled()).toBeFalse();
    expect(await matCheckbox.isChecked()).toBeFalse();
    expect(await matCheckbox.getLabelText()).toBe('test-label');
  });

  it('should update store when checkbox is toggled', async () => {
    fixture.detectChanges();

    const matCheckbox = await harnessLoader.getHarness(MatCheckboxHarness);
    await matCheckbox.toggle();

    expect(parameterStore.currentParameters()).toEqual({
      'test-checkbox-id': true,
    });
    expect(await matCheckbox.isChecked()).toBeTrue();
  });

  it('should be disabled and not update store when parameter.readonly is true', async () => {
    fixture.componentRef.setInput('parameter', {
      ...defaultParameter,
      readonly: true,
    });
    fixture.detectChanges();

    const matCheckbox = await harnessLoader.getHarness(MatCheckboxHarness);
    expect(await matCheckbox.isDisabled()).toBeTrue();

    // Directly invoking onToggle when readonly should not modify store.
    fixture.componentInstance.onToggle(true);
    expect(parameterStore.currentParameters()).toEqual({
      'test-checkbox-id': false,
    });
  });

  it('should reflect store updates in isChecked', async () => {
    fixture.detectChanges();

    const matCheckbox = await harnessLoader.getHarness(MatCheckboxHarness);
    expect(await matCheckbox.isChecked()).toBeFalse();

    parameterStore.set('test-checkbox-id', true);
    fixture.detectChanges();

    expect(await matCheckbox.isChecked()).toBeTrue();
  });
});
