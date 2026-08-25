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
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { create } from '@bufbuild/protobuf';
import {
  PopupFormSchema,
  PopupService,
} from 'src/app/generated/api/v1/popup_pb';
import { Client } from '@connectrpc/connect';
import { TextPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/text-popup-content.component';

describe('TextPopupContentComponent', () => {
  let component: TextPopupContentComponent;
  let fixture: ComponentFixture<TextPopupContentComponent>;
  let mockClient: jasmine.SpyObj<Client<typeof PopupService>>;

  const mockForm = create(PopupFormSchema, {
    id: 'test-id',
    title: 'Cluster Name',
    description: 'Enter your cluster name',
    payload: {
      case: 'text',
      value: { placeholder: 'e.g. cluster-1' },
    },
  });

  beforeEach(async () => {
    mockClient = jasmine.createSpyObj('PopupService', [
      'validatePopupAnswer',
      'submitPopupAnswer',
    ]);
    mockClient.validatePopupAnswer.and.resolveTo({
      id: 'test-id',
      validationError: '',
      $typeName: 'api.v1.ValidatePopupAnswerResponse',
    });
    mockClient.submitPopupAnswer.and.resolveTo({
      $typeName: 'api.v1.SubmitPopupAnswerResponse',
    });

    await TestBed.configureTestingModule({
      imports: [TextPopupContentComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(TextPopupContentComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('form', mockForm);
    fixture.componentRef.setInput('client', mockClient);
    fixture.detectChanges();
  });

  it('should initialize and validate initial empty input', async () => {
    await fixture.whenStable();
    expect(mockClient.validatePopupAnswer).toHaveBeenCalledWith({
      id: 'test-id',
      payload: {
        case: 'text',
        value: { value: '' },
      },
    });
    expect(component.isValid()).toBeTrue();
  });

  it('should revalidate when user types input', async () => {
    mockClient.validatePopupAnswer.and.resolveTo({
      id: 'test-id',
      validationError: 'Cluster not found',
      $typeName: 'api.v1.ValidatePopupAnswerResponse',
    });

    await component.onTextInput({
      target: { value: 'invalid-name' },
    } as unknown as Event);

    expect(component.inputValue()).toBe('invalid-name');
    expect(component.validationError()).toBe('Cluster not found');
    expect(component.isValid()).toBeFalse();
  });

  it('should submit answer and emit completed event', async () => {
    let completedEmitted = false;
    component.completed.subscribe(() => {
      completedEmitted = true;
    });

    await component.onTextInput({
      target: { value: 'valid-cluster' },
    } as unknown as Event);

    mockClient.validatePopupAnswer.and.resolveTo({
      id: 'test-id',
      validationError: '',
      $typeName: 'api.v1.ValidatePopupAnswerResponse',
    });

    await component.onSubmit();

    expect(mockClient.submitPopupAnswer).toHaveBeenCalledWith({
      id: 'test-id',
      payload: {
        case: 'text',
        value: { value: 'valid-cluster' },
      },
    });
    expect(completedEmitted).toBeTrue();
  });

  it('should discard out-of-order validation responses', async () => {
    let resolveFirstValidation!: (value: {
      id: string;
      validationError: string;
      $typeName: 'api.v1.ValidatePopupAnswerResponse';
    }) => void;
    const firstPromise = new Promise<{
      id: string;
      validationError: string;
      $typeName: 'api.v1.ValidatePopupAnswerResponse';
    }>((resolve) => {
      resolveFirstValidation = resolve;
    });

    // Request 1 will hang until resolveFirstValidation is called.
    mockClient.validatePopupAnswer.and.returnValue(
      firstPromise as unknown as ReturnType<
        typeof mockClient.validatePopupAnswer
      >,
    );

    // User types first value
    const inputPromise1 = component.onTextInput({
      target: { value: 'first-slow' },
    } as unknown as Event);

    // Now user immediately types second value, which resolves quickly
    mockClient.validatePopupAnswer.and.resolveTo({
      id: 'test-id',
      validationError: '',
      $typeName: 'api.v1.ValidatePopupAnswerResponse',
    });

    await component.onTextInput({
      target: { value: 'second-fast' },
    } as unknown as Event);

    expect(component.validationError()).toBe('');
    expect(component.isValid()).toBeTrue();

    // Stale first request finally resolves with an error after second request finished
    resolveFirstValidation({
      id: 'test-id',
      validationError: 'Stale error from first input',
      $typeName: 'api.v1.ValidatePopupAnswerResponse',
    });
    await inputPromise1;

    // Stale error must be discarded and NOT overwrite current valid state
    expect(component.validationError()).toBe('');
    expect(component.isValid()).toBeTrue();
  });
});
