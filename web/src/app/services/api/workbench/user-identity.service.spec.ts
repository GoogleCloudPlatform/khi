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

import { TestBed } from '@angular/core/testing';
import {
  KHI_USER_ID_STORAGE_KEY,
  UserIdentityService,
} from 'src/app/services/api/workbench/user-identity.service';

describe('UserIdentityService', () => {
  beforeEach(() => {
    localStorage.removeItem(KHI_USER_ID_STORAGE_KEY);
  });

  afterEach(() => {
    localStorage.removeItem(KHI_USER_ID_STORAGE_KEY);
  });

  it('should generate a new user ID if none is stored in localStorage', () => {
    TestBed.configureTestingModule({
      providers: [UserIdentityService],
    });
    const service = TestBed.inject(UserIdentityService);

    expect(service.userId).toBeTruthy();
    expect(service.userId.length).toBeGreaterThan(0);
    expect(localStorage.getItem(KHI_USER_ID_STORAGE_KEY)).toBe(service.userId);
  });

  it('should reuse existing user ID if already in localStorage', () => {
    localStorage.setItem(KHI_USER_ID_STORAGE_KEY, 'existing-uuid-1234');

    TestBed.configureTestingModule({
      providers: [UserIdentityService],
    });
    const service = TestBed.inject(UserIdentityService);

    expect(service.userId).toBe('existing-uuid-1234');
  });
});
