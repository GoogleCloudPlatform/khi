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

import {
  ComponentFixture,
  fakeAsync,
  TestBed,
  tick,
} from '@angular/core/testing';
import {
  RequestUserActionPopupComponent,
  RequestUserActionPopupRequest,
} from './request-user-action-popup.component';
import {
  MAT_DIALOG_DATA,
  MatDialog,
  MatDialogModule,
  MatDialogRef,
} from '@angular/material/dialog';
import { MatDialogHarness } from '@angular/material/dialog/testing';
import { HarnessLoader } from '@angular/cdk/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { Component } from '@angular/core';
import { PopupFormRequestWithClient } from 'src/app/services/popup/popup-manager';
import { By } from '@angular/platform-browser';
import { MockPopupClient } from 'src/app/services/popup/mock';

describe('RequestUserActionPopup in dialog context', () => {
  @Component({
    template: '<div></div>',
    standalone: true,
  })
  class TestingDialogWrapComponent {}

  let testingWrapper: ComponentFixture<TestingDialogWrapComponent>;
  let loader: HarnessLoader;
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestingDialogWrapComponent, MatDialogModule],
    }).compileComponents();
    testingWrapper = TestBed.createComponent(TestingDialogWrapComponent);
    testingWrapper.detectChanges();
    loader = await TestbedHarnessEnvironment.documentRootLoader(testingWrapper);
  });

  async function testIfDialogShowingUpWithParam(
    request: PopupFormRequestWithClient,
  ) {
    const matDialog = TestBed.inject(MatDialog);
    matDialog.open<
      RequestUserActionPopupComponent,
      RequestUserActionPopupRequest
    >(RequestUserActionPopupComponent, {
      data: {
        formRequest: request,
      },
    });
    const dialogs = await loader.getAllHarnesses(MatDialogHarness);
    expect(dialogs.length).toBe(1);
  }

  it('should be instanciated with type=text', async () => {
    await testIfDialogShowingUpWithParam({
      id: 'foo',
      type: 'text',
      title: 'foo title',
      description: 'test description',
      placeholder: 'test placeholder',
      options: {},
      client: new MockPopupClient(),
    });
  });
});

describe('RequestUserActionPopup', () => {
  let matDialogRefSpy: jasmine.SpyObj<
    MatDialogRef<RequestUserActionPopupRequest, void>
  >;
  beforeEach(async () => {
    matDialogRefSpy = jasmine.createSpyObj('MatDialogRef', ['close'], {
      disableClose: false,
    });
  });
  it('should have disbaled submit button at first', async () => {
    await TestBed.configureTestingModule({
      imports: [RequestUserActionPopupComponent, MatDialogModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            formRequest: {
              id: 'foo',
              type: 'text',
              title: 'foo title',
              description: 'test description',
              placeholder: 'test placeholder',
              client: new MockPopupClient(),
            },
          },
        },
        {
          provide: MatDialogRef,
          useValue: matDialogRefSpy,
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(RequestUserActionPopupComponent);
    fixture.detectChanges();
    const button = fixture.debugElement.query(By.css('.submit-button'));
    expect(button.nativeElement.disabled).toBe(true);
  });

  it('should update the disabled status of submit button by input', fakeAsync(async () => {
    await TestBed.configureTestingModule({
      imports: [RequestUserActionPopupComponent, MatDialogModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            formRequest: {
              id: 'foo',
              type: 'text',
              title: 'foo title',
              description: 'test description',
              placeholder: 'test placeholder',
              client: new MockPopupClient(),
            },
          },
        },
        {
          provide: MatDialogRef,
          useValue: matDialogRefSpy,
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(RequestUserActionPopupComponent);
    fixture.detectChanges();
    const textarea = fixture.debugElement.query(
      By.css('.input-text-type-textarea'),
    );
    const button = fixture.debugElement.query(By.css('.submit-button'));

    textarea.nativeElement.value = 'valid';
    textarea.nativeElement.dispatchEvent(new Event('input'));
    tick(1000);
    fixture.detectChanges();
    expect(button.nativeElement.disabled).toBe(false);

    textarea.nativeElement.value = 'invalid';
    textarea.nativeElement.dispatchEvent(new Event('input'));
    tick(1000);
    fixture.detectChanges();
    expect(button.nativeElement.disabled).toBe(true);
  }));

  it('should close dialog after submit', fakeAsync(async () => {
    await TestBed.configureTestingModule({
      imports: [RequestUserActionPopupComponent, MatDialogModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            formRequest: {
              id: 'foo',
              type: 'text',
              title: 'foo title',
              description: 'test description',
              placeholder: 'test placeholder',
              client: new MockPopupClient(),
            },
          },
        },
        {
          provide: MatDialogRef,
          useValue: matDialogRefSpy,
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(RequestUserActionPopupComponent);
    fixture.detectChanges();

    const textarea = fixture.debugElement.query(
      By.css('.input-text-type-textarea'),
    );
    const button = fixture.debugElement.query(By.css('.submit-button'));

    textarea.nativeElement.value = 'valid';
    textarea.nativeElement.dispatchEvent(new Event('input'));
    tick(1000);
    fixture.detectChanges();
    button.nativeElement.click();
    tick(1000);
    expect(matDialogRefSpy.close).toHaveBeenCalled();
  }));

  it('should show the valdiation error', fakeAsync(async () => {
    await TestBed.configureTestingModule({
      imports: [RequestUserActionPopupComponent, MatDialogModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            formRequest: {
              id: 'foo',
              type: 'text',
              title: 'foo title',
              description: 'test description',
              placeholder: 'test placeholder',
              client: new MockPopupClient(),
            },
          },
        },
        {
          provide: MatDialogRef,
          useValue: matDialogRefSpy,
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(RequestUserActionPopupComponent);
    fixture.detectChanges();

    const textarea = fixture.debugElement.query(
      By.css('.input-text-type-textarea'),
    );
    const validationError = fixture.debugElement.query(
      By.css('.validation-error'),
    );

    textarea.nativeElement.value = 'invalid';
    textarea.nativeElement.dispatchEvent(new Event('input'));
    tick(1000);
    fixture.detectChanges();
    expect(validationError.nativeElement.textContent).toBe(
      "invalid isn't valid",
    );
  }));
});
