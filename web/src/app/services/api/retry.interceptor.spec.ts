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

import {
  DescMethodStreaming,
  DescMethodUnary,
  DescService,
  Message,
} from '@bufbuild/protobuf';
import {
  Code,
  ConnectError,
  createContextValues,
  StreamRequest,
  StreamResponse,
  UnaryRequest,
  UnaryResponse,
} from '@connectrpc/connect';
import {
  createRetryInterceptor,
  DEFAULT_RETRYABLE_METHODS,
} from 'src/app/services/api/retry.interceptor';
import { CancellationError } from 'src/app/store/domain/filter/types';

describe('retry.interceptor', () => {
  function createMockUnaryRequest(
    methodName: string,
    signal?: AbortSignal,
  ): UnaryRequest {
    const service: DescService = {
      typeName: 'khi.api.v1.TestService',
      methods: {},
    } as unknown as DescService;

    const method: DescMethodUnary = {
      name: methodName,
      methodKind: 'unary',
      parent: service,
    } as unknown as DescMethodUnary;

    return {
      stream: false,
      service,
      method,
      url: `http://localhost/${methodName}`,
      requestMethod: 'POST',
      signal: signal ?? new AbortController().signal,
      header: new Headers(),
      contextValues: createContextValues(),
      message: {} as Message,
    };
  }

  function createMockStreamRequest(methodName: string): StreamRequest {
    const service: DescService = {
      typeName: 'khi.api.v1.TestService',
      methods: {},
    } as unknown as DescService;

    const method: DescMethodStreaming = {
      name: methodName,
      methodKind: 'server_streaming',
      parent: service,
    } as unknown as DescMethodStreaming;

    return {
      stream: true,
      service,
      method,
      url: `http://localhost/${methodName}`,
      requestMethod: 'POST',
      signal: new AbortController().signal,
      header: new Headers(),
      contextValues: createContextValues(),
      message: (async function* () {})(),
    };
  }

  it('passes through streaming requests unchanged', async () => {
    const interceptor = createRetryInterceptor({ maxRetries: 3 });
    const req = createMockStreamRequest('WatchInspections');
    let called = false;
    const next = async (
      r: StreamRequest | UnaryRequest,
    ): Promise<StreamResponse | UnaryResponse> => {
      called = true;
      return {
        stream: true,
        service: r.service,
        method: r.method,
        header: new Headers(),
        trailer: new Headers(),
        message: (async function* () {})(),
      } as StreamResponse;
    };

    await interceptor(next)(req);
    expect(called).toBeTrue();
  });

  it('does not retry non-retryable methods', async () => {
    const interceptor = createRetryInterceptor({
      maxRetries: 3,
      baseDelayMs: 1,
      maxDelayMs: 2,
    });
    const req = createMockUnaryRequest('CreateInspection');
    let callCount = 0;
    const next = async () => {
      callCount++;
      throw new ConnectError('503 Service Unavailable', Code.Unavailable);
    };

    await expectAsync(interceptor(next)(req)).toBeRejectedWithError(
      ConnectError,
    );
    expect(callCount).toBe(1);
  });

  it('retries retryable method on Code.Unavailable and succeeds when transient error resolves', async () => {
    const interceptor = createRetryInterceptor({
      maxRetries: 3,
      baseDelayMs: 1,
      maxDelayMs: 5,
    });
    const req = createMockUnaryRequest('ReadStructYAMLs');
    let callCount = 0;
    const mockResponse: UnaryResponse = {
      stream: false,
      service: req.service,
      method: req.method,
      header: new Headers(),
      trailer: new Headers(),
      message: {} as Message,
    };

    const next = async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError('HTTP 502 Bad Gateway', Code.Unavailable);
      }
      return mockResponse;
    };

    const res = await interceptor(next)(req);
    expect(callCount).toBe(2);
    expect(res).toBe(mockResponse);
  });

  it('throws error after exhausting maxRetries', async () => {
    const interceptor = createRetryInterceptor({
      maxRetries: 2,
      baseDelayMs: 1,
      maxDelayMs: 5,
    });
    const req = createMockUnaryRequest('ReadStructYAMLs');
    let callCount = 0;

    const next = async () => {
      callCount++;
      throw new ConnectError('503 Service Unavailable', Code.Unavailable);
    };

    await expectAsync(interceptor(next)(req)).toBeRejectedWithError(
      ConnectError,
    );
    // Initial call + 2 retries = 3 calls
    expect(callCount).toBe(3);
  });

  it('does not retry non-retryable errors like InvalidArgument', async () => {
    const interceptor = createRetryInterceptor({
      maxRetries: 3,
      baseDelayMs: 1,
      maxDelayMs: 5,
    });
    const req = createMockUnaryRequest('ReadStructYAMLs');
    let callCount = 0;

    const next = async () => {
      callCount++;
      throw new ConnectError('Invalid arguments', Code.InvalidArgument);
    };

    await expectAsync(interceptor(next)(req)).toBeRejectedWithError(
      ConnectError,
    );
    expect(callCount).toBe(1);
  });

  it('passes through directly if request signal is already aborted before execution', async () => {
    const controller = new AbortController();
    controller.abort();
    const interceptor = createRetryInterceptor({ maxRetries: 3 });
    const req = createMockUnaryRequest('ReadStructYAMLs', controller.signal);
    let callCount = 0;
    const next = async () => {
      callCount++;
      return { stream: false } as UnaryResponse;
    };

    await interceptor(next)(req);
    expect(callCount).toBe(1);
  });

  it('aborts immediately during delayWithSignal and does not issue subsequent retries', async () => {
    const controller = new AbortController();
    const interceptor = createRetryInterceptor({
      maxRetries: 3,
      baseDelayMs: 1000,
      maxDelayMs: 2000,
    });
    const req = createMockUnaryRequest('ReadStructYAMLs', controller.signal);
    let callCount = 0;

    const next = async () => {
      callCount++;
      throw new ConnectError('503 Service Unavailable', Code.Unavailable);
    };

    const interceptorPromise = interceptor(next)(req);
    // Abort shortly after entering delayWithSignal
    setTimeout(() => controller.abort(), 20);

    await expectAsync(interceptorPromise).toBeRejectedWith(
      jasmine.any(CancellationError),
    );
    expect(callCount).toBe(1);
  });

  it('stops retry if request signal is aborted inside next()', async () => {
    const controller = new AbortController();
    const interceptor = createRetryInterceptor({
      maxRetries: 3,
      baseDelayMs: 50,
      maxDelayMs: 100,
    });
    const req = createMockUnaryRequest('ReadStructYAMLs', controller.signal);
    let callCount = 0;

    const next = async () => {
      callCount++;
      controller.abort();
      throw new ConnectError('502 Bad Gateway', Code.Unavailable);
    };

    await expectAsync(interceptor(next)(req)).toBeRejectedWith(
      jasmine.any(CancellationError),
    );
    expect(callCount).toBe(1);
  });

  it('contains expected default retryable methods', () => {
    expect(DEFAULT_RETRYABLE_METHODS.has('ReadStructYAMLs')).toBeTrue();
    expect(DEFAULT_RETRYABLE_METHODS.has('OpenWorkbenchSync')).toBeTrue();
    expect(DEFAULT_RETRYABLE_METHODS.has('FilterTimelineSync')).toBeTrue();
    expect(DEFAULT_RETRYABLE_METHODS.has('ValidateTimelineQuery')).toBeTrue();
    expect(DEFAULT_RETRYABLE_METHODS.has('ValidateLogQuery')).toBeTrue();
    expect(DEFAULT_RETRYABLE_METHODS.has('GetInspectionDataChunk')).toBeFalse();
    expect(DEFAULT_RETRYABLE_METHODS.has('UploadFileChunk')).toBeFalse();
    expect(DEFAULT_RETRYABLE_METHODS.has('UploadInspectionChunk')).toBeFalse();
    expect(DEFAULT_RETRYABLE_METHODS.has('CreateInspection')).toBeFalse();
    expect(DEFAULT_RETRYABLE_METHODS.has('RunInspection')).toBeFalse();
  });
});
