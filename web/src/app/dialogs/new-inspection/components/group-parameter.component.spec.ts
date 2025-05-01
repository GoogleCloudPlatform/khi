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
import { GroupParameterComponent } from './group-parameter.component';
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting,
} from '@angular/platform-browser-dynamic/testing';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatIconRegistry } from '@angular/material/icon';
import {
  GroupParameterFormField,
  ParameterFormField,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { FILE_UPLOADER, MockFileUploader } from './service/file-uploader';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from './service/parameter-store';
import { By } from '@angular/platform-browser';

describe('GroupParameterComponent', () => {
  let fixture: ComponentFixture<GroupParameterComponent>;
  let parameterStore: DefaultParameterStore;

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
          provide: FILE_UPLOADER,
          useValue: new MockFileUploader(),
        },
        {
          provide: PARAMETER_STORE,
          useValue: parameterStore,
        },
      ],
    }).compileComponents();
    const matIconRegistry = TestBed.inject(MatIconRegistry);
    matIconRegistry.setDefaultFontSetClass('material-symbols-outlined');
    fixture = TestBed.createComponent(GroupParameterComponent);
  });

  afterEach(() => {
    parameterStore.destroy();
  });

  it('should pass input values', () => {
    fixture.componentRef.setInput('parameter', {
      type: ParameterInputType.Group,
      label: 'group',
      hintType: ParameterHintType.None,
      collapsible: false,
      collapsedByDefault: false,
      children: [
        {
          type: ParameterInputType.Text,
          label: 'text-form-1',
          hintType: ParameterHintType.None,
        },
        {
          type: ParameterInputType.File,
          label: 'file-form-1',
          hintType: ParameterHintType.None,
        },
        {
          type: ParameterInputType.Group,
          label: 'child-group',
          hintType: ParameterHintType.None,
          children: [
            {
              type: ParameterInputType.Text,
              label: 'text-form-children-1',
              hintType: ParameterHintType.None,
            },
            {
              type: ParameterInputType.File,
              label: 'file-form-children-1',
              hintType: ParameterHintType.None,
            },
          ],
        },
        {
          type: ParameterInputType.Text,
          label: 'text-form-2',
          hintType: ParameterHintType.None,
        },
        {
          type: ParameterInputType.File,
          label: 'file-form-2',
          hintType: ParameterHintType.None,
        },
      ] as ParameterFormField[],
    } as GroupParameterFormField);
    fixture.detectChanges();
    expect(fixture.componentInstance).toBeTruthy();

    const containerElement = fixture.debugElement.query(By.css('.container'));
    expect(containerElement.classes['collapsable']).toBeFalsy();
  });

  it('becomes collapsable when `collapsable` = true', () => {
    fixture.componentRef.setInput('parameter', {
      type: ParameterInputType.Group,
      label: 'group',
      description: 'this is a test description',
      collapsible: true,
      collapsedByDefault: true,
      hintType: ParameterHintType.None,
      children: [
        {
          type: ParameterInputType.Text,
          label: 'text-form-1',
          hintType: ParameterHintType.None,
        },
      ],
    } as GroupParameterFormField);
    fixture.detectChanges();

    const containerElement = fixture.debugElement.query(By.css('.container'));
    expect(containerElement.classes['collapsable']).toBeTruthy();
    expect(fixture.componentInstance.childrenStatus()).toBe('collapsed');
  });

  it('expands by default when `collapsedByDefault` = false', () => {
    fixture.componentRef.setInput('parameter', {
      type: ParameterInputType.Group,
      label: 'group',
      description: 'this is a test description',
      collapsible: true,
      collapsedByDefault: false,
      hintType: ParameterHintType.None,
      children: [
        {
          type: ParameterInputType.Text,
          label: 'text-form-1',
          hintType: ParameterHintType.None,
        },
      ],
    } as GroupParameterFormField);
    fixture.detectChanges();

    expect(fixture.componentInstance.childrenStatus()).toBe('expanded');
  });
});
