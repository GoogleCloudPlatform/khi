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

import { HarnessLoader } from '@angular/cdk/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ParameterHintComponent } from './parameter-hint.component';
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting,
} from '@angular/platform-browser-dynamic/testing';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MatIconRegistry } from '@angular/material/icon';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { By } from '@angular/platform-browser';
import { MatIconHarness } from '@angular/material/icon/testing';
import { ParameterHintType } from 'src/app/common/schema/form-types';

describe('ParameterHintComponent', () => {
  let fixture: ComponentFixture<ParameterHintComponent>;
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
    fixture = TestBed.createComponent(ParameterHintComponent);
    fixture.componentRef.setInput('parameter', {
      hintType: ParameterHintType.Error,
      hint: 'test-hint1 \n test-hint2 \n test-hint3',
    });
    harnessLoader = TestbedHarnessEnvironment.loader(fixture);
  });

  it('should pass input values', async () => {
    fixture.detectChanges();
    const matIcon = await harnessLoader.getAllHarnesses(MatIconHarness);

    expect(fixture.componentInstance).toBeTruthy();
    const container = fixture.debugElement.query(By.css('.container'));
    expect('error' in container.classes).toBeTrue();
    expect('warning' in container.classes).toBeFalse();
    expect('info' in container.classes).toBeFalse();

    const hint = fixture.debugElement.query(By.css('.hint'));
    expect(hint.nativeElement.innerHTML).toBe(
      'test-hint1 <br> test-hint2 <br> test-hint3',
    );

    expect(matIcon.length).toBe(1);
    expect(await matIcon[0].getName()).toBe('error');
  });

  it('shows with type = warning', async () => {
    fixture.componentRef.setInput('parameter', {
      hintType: ParameterHintType.Warning,
      hint: 'test-hint1 \n test-hint2 \n test-hint3',
    });
    fixture.detectChanges();
    const matIcon = await harnessLoader.getAllHarnesses(MatIconHarness);

    expect(fixture.componentInstance).toBeTruthy();
    const container = fixture.debugElement.query(By.css('.container'));
    expect('error' in container.classes).toBeFalse();
    expect('warning' in container.classes).toBeTrue();
    expect('info' in container.classes).toBeFalse();

    expect(matIcon.length).toBe(1);
    expect(await matIcon[0].getName()).toBe('warning');
  });

  it('shows with type = info', async () => {
    fixture.componentRef.setInput('parameter', {
      hintType: ParameterHintType.Info,
      hint: 'test-hint1 \n test-hint2 \n test-hint3',
    });
    fixture.detectChanges();
    const matIcon = await harnessLoader.getAllHarnesses(MatIconHarness);

    expect(fixture.componentInstance).toBeTruthy();
    const container = fixture.debugElement.query(By.css('.container'));
    expect('error' in container.classes).toBeFalse();
    expect('warning' in container.classes).toBeFalse();
    expect('info' in container.classes).toBeTrue();

    expect(matIcon.length).toBe(1);
    expect(await matIcon[0].getName()).toBe('info');
  });
});
