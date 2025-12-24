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

import { provideZoneChangeDetection, NgModule } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SetParameterComponent } from './set-parameter.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatIconRegistry } from '@angular/material/icon';
import {
  ParameterHintType,
  ParameterInputType,
  SetParameterFormField,
} from 'src/app/common/schema/form-types';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from './service/parameter-store';
import { firstValueFrom } from 'rxjs';
import {
  BrowserTestingModule,
  platformBrowserTesting,
} from '@angular/platform-browser/testing';
import { By } from '@angular/platform-browser';
import { SetInputComponent } from 'src/app/shared/components/set-input/set-input.component';

@NgModule({ providers: [provideZoneChangeDetection()] })
export class ZoneChangeDetectionModule {}

describe('SetParameterComponent', () => {
  let fixture: ComponentFixture<SetParameterComponent>;
  let parameterStore: DefaultParameterStore;

  const defaultParameter: SetParameterFormField = {
    id: 'test-parameter-id',
    label: 'test-label',
    type: ParameterInputType.Set,
    default: ['opt1'],
    description:
      'Lorem ipsum dolor sit amet, consectetur adipiscing elit, <br> sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
    hintType: ParameterHintType.Info,
    hint: 'test hint',
    options: [
      { id: 'opt1', description: 'Option 1' },
      { id: 'opt2', description: 'Option 2' },
    ],
    allowAddAll: true,
    allowRemoveAll: true,
    allowCustomValue: true,
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
      imports: [BrowserAnimationsModule, SetParameterComponent],
      providers: [
        {
          provide: PARAMETER_STORE,
          useValue: parameterStore,
        },
      ],
    }).compileComponents();
    const matIconRegistry = TestBed.inject(MatIconRegistry);
    matIconRegistry.setDefaultFontSetClass('material-symbols-outlined');
    fixture = TestBed.createComponent(SetParameterComponent);
    fixture.componentRef.setInput('parameter', defaultParameter);
    parameterStore.setDefaultValues({
      'test-parameter-id': ['opt1'],
    });
    fixture.detectChanges();
  });

  afterEach(() => {
    parameterStore.destroy();
  });

  it('should create', () => {
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('should pass correct properties to set-input', () => {
    const setInputDe = fixture.debugElement.query(
      By.directive(SetInputComponent),
    );
    expect(setInputDe).toBeTruthy();
    const setInputComponent = setInputDe.componentInstance as SetInputComponent;

    expect(setInputComponent.choices().length).toBe(2);
    expect(setInputComponent.choices()[0].id).toBe('opt1');
    expect(setInputComponent.allowCustomValues()).toBeTrue();
    expect(setInputComponent.showAddAll()).toBeTrue();
    expect(setInputComponent.showRemoveAll()).toBeTrue();
  });

  it('should update store when selection changes', async () => {
    const component = fixture.componentInstance;
    // Simulate selection change
    component.onSelectionChange(['opt1', 'opt2']);

    expect(await firstValueFrom(parameterStore.watchAll())).toEqual({
      'test-parameter-id': ['opt1', 'opt2'],
    });
  });

  it('should update stagingInput when store updates', async () => {
    const component = fixture.componentInstance;
    parameterStore.set('test-parameter-id', ['opt2']);
    fixture.detectChanges();

    expect(component.stagingInput()).toEqual(['opt2']);
  });
});
