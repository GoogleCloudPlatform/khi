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
import { ConnectClientService } from 'src/app/services/api/connect-client.service';

describe('ConnectClientService', () => {
  let service: ConnectClientService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ConnectClientService],
    });
    service = TestBed.inject(ConnectClientService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should initialize workbenchClient with openWorkbench method', () => {
    expect(service.workbenchClient).toBeTruthy();
    expect(typeof service.workbenchClient.openWorkbench).toBe('function');
  });
  it('should initialize serverStatusClient with watchServerStat method', () => {
    expect(service.serverStatusClient).toBeTruthy();
    expect(typeof service.serverStatusClient.watchServerStat).toBe('function');
  });
  it('should initialize fileParameterUploadClient', () => {
    expect(service.fileParameterUploadClient).toBeTruthy();
    expect(typeof service.fileParameterUploadClient.startFileUpload).toBe(
      'function',
    );
  });

  it('should initialize importInspectionClient', () => {
    expect(service.importInspectionClient).toBeTruthy();
    expect(typeof service.importInspectionClient.startImportInspection).toBe(
      'function',
    );
  });

  it('should use JSON content-type when environment.production is false', async () => {
    let capturedContentType: string | null = null;
    spyOn(globalThis, 'fetch').and.callFake((_input, init) => {
      const headers = new Headers(init?.headers);
      capturedContentType = headers.get('content-type');
      return Promise.resolve(
        new Response(JSON.stringify({ active: true }), {
          status: 200,
          headers: { 'content-type': 'application/json' },
        }),
      );
    });

    await service.workbenchClient.heartbeatWorkbench({ workbenchId: 'test' });

    expect(capturedContentType).toContain('application/json');
  });

  it('should initialize inspectionClient with RPC methods', () => {
    expect(service.inspectionClient).toBeTruthy();
    expect(typeof service.inspectionClient.getInspectionTypes).toBe('function');
    expect(typeof service.inspectionClient.watchInspections).toBe('function');
    expect(typeof service.inspectionClient.createInspection).toBe('function');
  });
});
