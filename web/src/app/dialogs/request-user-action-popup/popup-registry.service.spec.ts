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

import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { PopupRegistry } from 'src/app/dialogs/request-user-action-popup/popup-registry.service';
import { TextPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/text-popup-content.component';
import { OAuthLoginPopupContentComponent } from 'src/app/dialogs/request-user-action-popup/components/oauth-login-popup-content.component';

@Component({ template: '' })
class DummyComponent {}

describe('PopupRegistry', () => {
  let registry: PopupRegistry;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    registry = TestBed.inject(PopupRegistry);
  });

  it('should have default components registered', () => {
    expect(registry.getComponent('text')).toBe(TextPopupContentComponent);
    expect(registry.getComponent('oauthLogin')).toBe(
      OAuthLoginPopupContentComponent,
    );
  });

  it('should return undefined for unregistered case', () => {
    expect(registry.getComponent('unregistered')).toBeUndefined();
  });

  it('should allow registering new components', () => {
    registry.register('custom', DummyComponent);
    expect(registry.getComponent('custom')).toBe(DummyComponent);
  });
});
