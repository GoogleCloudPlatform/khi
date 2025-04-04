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
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting,
} from '@angular/platform-browser-dynamic/testing';
import { TextParameterComponent } from './text-parameter.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatIconRegistry } from '@angular/material/icon';
import {
  ParameterHintType,
  TextParameterFormField,
} from 'src/app/common/schema/form-types';
import { MatInputHarness } from '@angular/material/input/testing';
import { HarnessLoader } from '@angular/cdk/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from './service/parameter-store';
import { firstValueFrom } from 'rxjs';

describe('TextParameterComponent', () => {
  let fixture: ComponentFixture<TextParameterComponent>;
  let harnessLoader: HarnessLoader;
  let parameterStore: DefaultParameterStore;

  const defaultParameter = {
    id: 'test-parameter-id',
    label: 'test-label',
    default: 'test-default-value',
    description:
      'Lorem ipsum dolor sit amet, consectetur adipiscing elit, <br> sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
    hintType: ParameterHintType.Error,
    hint: 'parameter test validation failed',
    readonly: false,
    suggestions: ['foo', 'bar', 'qux'],
  } as TextParameterFormField;

  beforeAll(() => {
    TestBed.resetTestEnvironment();
    TestBed.initTestEnvironment(
      BrowserDynamicTestingModule,
      platformBrowserDynamicTesting(),
      { teardown: { destroyAfterEach: false } },
    );
  });

  beforeEach(async () => {
    parameterStore = new DefaultParameterStore();
    await TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: parameterStore,
        },
      ],
    }).compileComponents();
    const matIconRegistry = TestBed.inject(MatIconRegistry);
    matIconRegistry.setDefaultFontSetClass('material-symbols-outlined');
    fixture = TestBed.createComponent(TextParameterComponent);
    fixture.componentRef.setInput('parameter', defaultParameter);
    parameterStore.setDefaultValues({
      'test-parameter-id': 'the default value',
    });
    harnessLoader = TestbedHarnessEnvironment.loader(fixture);
  });

  afterEach(() => {
    parameterStore.destroy();
  });

  it('should create', async () => {
    fixture.detectChanges();

    expect(fixture.componentInstance).toBeTruthy();
    const matInput = await harnessLoader.getHarness(MatInputHarness);

    expect(await matInput.isDisabled()).toBeFalse();
    expect(await matInput.getPlaceholder()).toBe('test-default-value');
  });

  it('should set the value to store when input received', async () => {
    fixture.detectChanges();

    expect(fixture.componentInstance).toBeTruthy();
    const matInput = await harnessLoader.getHarness(MatInputHarness);

    await matInput.setValue('updated value');
    expect(await firstValueFrom(parameterStore.watchAll())).toEqual({
      'test-parameter-id': 'updated value',
    });
  });

  it('should make its input disabled when parameter.readonly = true', async () => {
    fixture.componentRef.setInput('parameter', {
      ...defaultParameter,
      readonly: true,
    });
    fixture.detectChanges();
    const matInput = await harnessLoader.getHarness(MatInputHarness);

    expect(await matInput.isDisabled()).toBeTrue();
  });
});
