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
import { PopupFormSchema } from 'src/app/generated/api/v1/popup_pb';
import { OAuthLoginPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/oauth-login-popup-content.component';

describe('OAuthLoginPopupContentComponent', () => {
  let component: OAuthLoginPopupContentComponent;
  let fixture: ComponentFixture<OAuthLoginPopupContentComponent>;

  const mockForm = create(PopupFormSchema, {
    id: 'oauth-id',
    title: 'OAuth Token',
    description: 'Please login to Google',
    payload: {
      case: 'oauthLogin',
      value: { authUrl: 'http://example.com/auth' },
    },
  });

  beforeEach(async () => {
    spyOn(window, 'open');

    await TestBed.configureTestingModule({
      imports: [OAuthLoginPopupContentComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(OAuthLoginPopupContentComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('form', mockForm);
    fixture.detectChanges();
  });

  it('should open auth window on init', () => {
    expect(window.open).toHaveBeenCalledWith(
      'http://example.com/auth',
      'oauth login',
      'width=400px,height=500px',
    );
  });

  it('should reopen auth window when button is clicked', () => {
    (window.open as jasmine.Spy).calls.reset();
    component.openAuthWindow();
    expect(window.open).toHaveBeenCalledWith(
      'http://example.com/auth',
      'oauth login',
      'width=400px,height=500px',
    );
  });
});
