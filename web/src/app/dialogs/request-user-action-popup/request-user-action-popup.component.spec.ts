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
import {
  RequestUserActionPopupComponent,
  RequestUserActionPopupRequest,
} from 'src/app/dialogs/request-user-action-popup/request-user-action-popup.component';
import {
  MAT_DIALOG_DATA,
  MatDialog,
  MatDialogRef,
} from '@angular/material/dialog';
import { MatDialogHarness } from '@angular/material/dialog/testing';
import { HarnessLoader } from '@angular/cdk/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { Component } from '@angular/core';
import { PopupFormWithClient } from 'src/app/services/popup/popup-manager';
import { create } from '@bufbuild/protobuf';
import {
  PopupFormSchema,
  PopupService,
} from 'src/app/generated/api/v1/popup_pb';
import { Client } from '@connectrpc/connect';
import { TextPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/text-popup-content.component';
import { OAuthLoginPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/oauth-login-popup-content.component';
import { By } from '@angular/platform-browser';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('RequestUserActionPopup in dialog context', () => {
  @Component({
    template: '<div></div>',
    standalone: true,
  })
  class TestingDialogWrapComponent {}

  let testingWrapper: ComponentFixture<TestingDialogWrapComponent>;
  let loader: HarnessLoader;
  let mockClient: jasmine.SpyObj<Client<typeof PopupService>>;

  beforeEach(async () => {
    mockClient = jasmine.createSpyObj('PopupService', [
      'validatePopupAnswer',
      'submitPopupAnswer',
    ]);
    mockClient.validatePopupAnswer.and.resolveTo({
      id: 'foo',
      validationError: '',
      $typeName: 'api.v1.ValidatePopupAnswerResponse',
    });

    await TestBed.configureTestingModule({
      imports: [TestingDialogWrapComponent, NoopAnimationsModule],
    }).compileComponents();
    testingWrapper = TestBed.createComponent(TestingDialogWrapComponent);
    testingWrapper.detectChanges();
    loader = await TestbedHarnessEnvironment.documentRootLoader(testingWrapper);
  });

  async function testIfDialogShowingUpWithParam(
    request: PopupFormWithClient,
  ): Promise<void> {
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
    matDialog.closeAll();
  }

  it('should be instantiated with text popup', async () => {
    await testIfDialogShowingUpWithParam({
      form: create(PopupFormSchema, {
        id: 'foo',
        title: 'foo title',
        description: 'test description',
        payload: {
          case: 'text',
          value: {
            placeholder: 'test placeholder',
          },
        },
      }),
      client: mockClient,
    });
  });
});

describe('RequestUserActionPopup dynamic rendering', () => {
  let matDialogRefSpy: jasmine.SpyObj<
    MatDialogRef<RequestUserActionPopupRequest, void>
  >;
  let mockClient: jasmine.SpyObj<Client<typeof PopupService>>;

  beforeEach(async () => {
    matDialogRefSpy = jasmine.createSpyObj('MatDialogRef', ['close'], {
      disableClose: false,
    });
    mockClient = jasmine.createSpyObj('PopupService', [
      'validatePopupAnswer',
      'submitPopupAnswer',
    ]);
    mockClient.validatePopupAnswer.and.resolveTo({
      id: 'foo',
      validationError: '',
      $typeName: 'api.v1.ValidatePopupAnswerResponse',
    });
  });

  it('should render TextPopupContentComponent for text popup', async () => {
    await TestBed.configureTestingModule({
      imports: [RequestUserActionPopupComponent, NoopAnimationsModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            formRequest: {
              form: create(PopupFormSchema, {
                id: 'foo',
                title: 'Text Popup Title',
                description: 'Text description',
                payload: {
                  case: 'text',
                  value: { placeholder: 'test' },
                },
              }),
              client: mockClient,
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
    await fixture.whenStable();

    const textContent = fixture.debugElement.query(
      By.directive(TextPopupContentComponent),
    );
    expect(textContent).not.toBeNull();
  });

  it('should render OAuthLoginPopupContentComponent for oauthLogin popup', async () => {
    spyOn(window, 'open');
    await TestBed.configureTestingModule({
      imports: [RequestUserActionPopupComponent, NoopAnimationsModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            formRequest: {
              form: create(PopupFormSchema, {
                id: 'foo',
                title: 'OAuth Login Title',
                description: 'OAuth description',
                payload: {
                  case: 'oauthLogin',
                  value: { authUrl: 'http://example.com/auth' },
                },
              }),
              client: mockClient,
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
    await fixture.whenStable();

    const oauthContent = fixture.debugElement.query(
      By.directive(OAuthLoginPopupContentComponent),
    );
    expect(oauthContent).not.toBeNull();
  });

  it('should close dialog when onCompleted is called', () => {
    TestBed.configureTestingModule({
      imports: [RequestUserActionPopupComponent, NoopAnimationsModule],
      providers: [
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            formRequest: {
              form: create(PopupFormSchema, {
                id: 'foo',
                title: 'Text Popup Title',
                description: 'Text description',
                payload: {
                  case: 'text',
                  value: { placeholder: 'test' },
                },
              }),
              client: mockClient,
            },
          },
        },
        {
          provide: MatDialogRef,
          useValue: matDialogRefSpy,
        },
      ],
    });

    const fixture = TestBed.createComponent(RequestUserActionPopupComponent);
    fixture.componentInstance.onCompleted();
    expect(matDialogRefSpy.close).toHaveBeenCalled();
  });
});
