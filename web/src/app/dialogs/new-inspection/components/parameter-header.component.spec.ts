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
import { ParameterHeaderComponent } from './parameter-header.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatIconRegistry } from '@angular/material/icon';
import { By } from '@angular/platform-browser';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { HarnessLoader } from '@angular/cdk/testing';
import { MatIconHarness } from '@angular/material/icon/testing';
import { ParameterHintType } from 'src/app/common/schema/form-types';
describe('ParameterHeaderComponent', () => {
  let fixture: ComponentFixture<ParameterHeaderComponent>;
  let harnessLoader: HarnessLoader;

  beforeAll(() => {
    TestBed.resetTestEnvironment();
    TestBed.initTestEnvironment(
      BrowserDynamicTestingModule,
      platformBrowserDynamicTesting(),
      { teardown: { destroyAfterEach: false } },
    );
  });

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [BrowserAnimationsModule],
    }).compileComponents();
    const matIconRegistry = TestBed.inject(MatIconRegistry);
    matIconRegistry.setDefaultFontSetClass('material-symbols-outlined');
    fixture = TestBed.createComponent(ParameterHeaderComponent);
    fixture.componentRef.setInput('parameter', {
      label: 'test-label',
      description:
        'Lorem ipsum dolor sit amet, consectetur adipiscing elit, \n sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
      hintType: ParameterHintType.None,
    });
    harnessLoader = TestbedHarnessEnvironment.loader(fixture);
  });

  it('should pass input values', async () => {
    fixture.detectChanges();
    const matIcon = await harnessLoader.getAllHarnesses(MatIconHarness);

    expect(fixture.componentInstance).toBeTruthy();
    const label = fixture.debugElement.query(By.css('.label'));
    expect(label.nativeElement.textContent).toBe('test-label');
    const description = fixture.debugElement.query(By.css('.description'));
    expect(description.nativeElement.innerHTML).toBe(
      'Lorem ipsum dolor sit amet, consectetur adipiscing elit, <br> sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
    );

    expect(matIcon.length).toBe(1);
    expect(await matIcon[0].getName()).toBe('check_circle');
  });

  it('should show error icon when hintType = ERROR', async () => {
    fixture.componentRef.setInput('parameter', {
      label: 'test-label',
      description:
        'Lorem ipsum dolor sit amet, consectetur adipiscing elit, \n sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.',
      hintType: ParameterHintType.Error,
    });
    fixture.detectChanges();
    const matIcon = await harnessLoader.getAllHarnesses(MatIconHarness);

    expect(matIcon.length).toBe(1);
    expect(await matIcon[0].getName()).toBe('error');
  });
});
