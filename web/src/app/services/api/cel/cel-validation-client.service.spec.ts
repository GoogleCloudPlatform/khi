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
import { CelValidationClientService } from 'src/app/services/api/cel/cel-validation-client.service';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';

describe('CelValidationClientService', () => {
  let service: CelValidationClientService;
  let mockCelValidationClient: {
    validateTimelineQuery: jasmine.Spy;
    validateLogQuery: jasmine.Spy;
  };

  beforeEach(() => {
    mockCelValidationClient = {
      validateTimelineQuery: jasmine.createSpy('validateTimelineQuery'),
      validateLogQuery: jasmine.createSpy('validateLogQuery'),
    };

    TestBed.configureTestingModule({
      providers: [
        CelValidationClientService,
        {
          provide: ConnectClientService,
          useValue: {
            celValidationClient: mockCelValidationClient,
          },
        },
      ],
    });

    service = TestBed.inject(CelValidationClientService);
  });

  describe('validateTimelineQuery', () => {
    it('should return valid without calling RPC if query is empty', async () => {
      const res = await service.validateTimelineQuery('');
      expect(res.valid).toBeTrue();
      expect(res.errorMessage).toBe('');
      expect(
        mockCelValidationClient.validateTimelineQuery,
      ).not.toHaveBeenCalled();
    });

    it('should return RPC response when query is valid', async () => {
      mockCelValidationClient.validateTimelineQuery.and.resolveTo({
        valid: true,
        errorMessage: '',
      });

      const res = await service.validateTimelineQuery('name == "pod-1"');
      expect(res.valid).toBeTrue();
      expect(res.errorMessage).toBe('');
      expect(
        mockCelValidationClient.validateTimelineQuery,
      ).toHaveBeenCalledWith(
        { query: 'name == "pod-1"' },
        { signal: undefined },
      );
    });

    it('should return invalid when RPC returns validation error', async () => {
      mockCelValidationClient.validateTimelineQuery.and.resolveTo({
        valid: false,
        errorMessage: 'syntax error',
      });

      const res = await service.validateTimelineQuery('name ==');
      expect(res.valid).toBeFalse();
      expect(res.errorMessage).toBe('syntax error');
    });

    it('should catch network errors and return failure result', async () => {
      mockCelValidationClient.validateTimelineQuery.and.rejectWith(
        new Error('Network error'),
      );

      const res = await service.validateTimelineQuery('name == "pod-1"');
      expect(res.valid).toBeFalse();
      expect(res.errorMessage).toBe('Network error');
    });
  });

  describe('validateLogQuery', () => {
    it('should return valid without calling RPC if query is empty', async () => {
      const res = await service.validateLogQuery('   ');
      expect(res.valid).toBeTrue();
      expect(res.errorMessage).toBe('');
      expect(mockCelValidationClient.validateLogQuery).not.toHaveBeenCalled();
    });

    it('should return RPC response when log query is valid', async () => {
      mockCelValidationClient.validateLogQuery.and.resolveTo({
        valid: true,
        errorMessage: '',
      });

      const res = await service.validateLogQuery('severity >= INFO');
      expect(res.valid).toBeTrue();
      expect(res.errorMessage).toBe('');
      expect(mockCelValidationClient.validateLogQuery).toHaveBeenCalledWith(
        { query: 'severity >= INFO' },
        { signal: undefined },
      );
    });

    it('should return invalid when RPC returns error', async () => {
      mockCelValidationClient.validateLogQuery.and.resolveTo({
        valid: false,
        errorMessage: 'invalid variable',
      });

      const res = await service.validateLogQuery('foo == 1');
      expect(res.valid).toBeFalse();
      expect(res.errorMessage).toBe('invalid variable');
    });
  });
});
