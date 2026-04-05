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
import { ReporterErrorHandler } from './reporter-error-handler';
import { Reporter } from './reporter';

describe('ReporterErrorHandler', () => {
  let errorHandler: ReporterErrorHandler;
  let mockReporter: jasmine.SpyObj<Reporter>;

  beforeEach(() => {
    mockReporter = jasmine.createSpyObj('Reporter', ['send']);

    TestBed.configureTestingModule({
      providers: [
        ReporterErrorHandler,
        { provide: Reporter, useValue: mockReporter },
      ],
    });

    errorHandler = TestBed.inject(ReporterErrorHandler);
    spyOn(console, 'error'); // Suppresses console.error in tests.
  });

  it('should be created', () => {
    expect(errorHandler).toBeTruthy();
  });

  it('should report error and delegate to console.error', () => {
    const error = new Error('Test error');
    errorHandler.handleError(error);

    expect(mockReporter.send).toHaveBeenCalledWith({
      event: 'unhandled_error',
      message: 'Test error',
      stack: error.stack,
    });

    expect(console.error).toHaveBeenCalledWith(error);
  });

  it('should handle non-Error objects', () => {
    errorHandler.handleError('String error');

    expect(mockReporter.send).toHaveBeenCalledWith({
      event: 'unhandled_error',
      message: 'String error',
      stack: '',
    });

    expect(console.error).toHaveBeenCalledWith('String error');
  });

  it('should handle null error', () => {
    errorHandler.handleError(null);

    expect(mockReporter.send).toHaveBeenCalledWith({
      event: 'unhandled_error',
      message: 'null',
      stack: '',
    });

    expect(console.error).toHaveBeenCalledWith(null);
  });

  it('should handle undefined error', () => {
    errorHandler.handleError(undefined);

    expect(mockReporter.send).toHaveBeenCalledWith({
      event: 'unhandled_error',
      message: 'undefined',
      stack: '',
    });

    expect(console.error).toHaveBeenCalledWith(undefined);
  });
});
