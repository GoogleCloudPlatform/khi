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

import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import {
  BackendAPIImpl,
  BackendAPIUtil,
  InspectionClient,
} from './backend-api.service';
import {
  environment,
  DownloadEnvironmentConfig,
} from 'src/environments/environment';
import { ViewStateService } from '../view-state.service';
import { ProgressDialogStatusUpdator } from 'src/app/services/progress/progress-interface';
import {
  InspectionDryRunRequest,
  InspectionDryRunResponse,
  InspectionRunRequest,
} from '../../common/schema/api-types';
import { BackendAPI } from './backend-api-interface';
import { firstValueFrom, of } from 'rxjs';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { create } from '@bufbuild/protobuf';
import {
  CancelInspectionResponseSchema,
  CreateInspectionResponseSchema,
  DryRunInspectionResponseSchema,
  GetInspectionDataChunkResponseSchema,
  GetInspectionFeaturesResponseSchema,
  GetInspectionMetadataResponseSchema,
  GetInspectionTypesResponseSchema,
  GetInspectionsResponseSchema,
  InspectionPhase,
  RunInspectionResponseSchema,
  UpdateInspectionFeaturesResponseSchema,
} from 'src/app/generated/api/v1/inspection_pb';

describe('BackendAPIImpl testing', () => {
  let api: BackendAPIImpl;
  let mockInspectionClient: {
    getInspectionTypes: jasmine.Spy;
    getInspections: jasmine.Spy;
    createInspection: jasmine.Spy;
    patchInspection: jasmine.Spy;
    getInspectionFeatures: jasmine.Spy;
    updateInspectionFeatures: jasmine.Spy;
    getInspectionMetadata: jasmine.Spy;
    getInspectionDataChunk: jasmine.Spy;
    runInspection: jasmine.Spy;
    dryRunInspection: jasmine.Spy;
    cancelInspection: jasmine.Spy;
  };

  beforeEach(() => {
    mockInspectionClient = {
      getInspectionTypes: jasmine.createSpy('getInspectionTypes'),
      getInspections: jasmine.createSpy('getInspections'),
      createInspection: jasmine.createSpy('createInspection'),
      patchInspection: jasmine.createSpy('patchInspection'),
      getInspectionFeatures: jasmine
        .createSpy('getInspectionFeatures')
        .and.returnValue(
          Promise.resolve(
            create(GetInspectionFeaturesResponseSchema, { features: [] }),
          ),
        ),
      updateInspectionFeatures: jasmine.createSpy('updateInspectionFeatures'),
      getInspectionMetadata: jasmine.createSpy('getInspectionMetadata'),
      getInspectionDataChunk: jasmine.createSpy('getInspectionDataChunk'),
      runInspection: jasmine.createSpy('runInspection'),
      dryRunInspection: jasmine.createSpy('dryRunInspection'),
      cancelInspection: jasmine.createSpy('cancelInspection'),
    };

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        BackendAPIImpl,
        ViewStateService,
        {
          provide: ConnectClientService,
          useValue: { inspectionClient: mockInspectionClient },
        },
      ],
    });

    api = TestBed.inject(BackendAPIImpl);
  });

  it('read server-base-path from meta tag', () => {
    document.head.innerHTML += `<meta id="server-base-path" content="/api/v3">`;
    expect(BackendAPIImpl.getServerBasePath()).toEqual('/api/v3');
    document.getElementById('server-base-path')?.remove();
    expect(BackendAPIImpl.getServerBasePath()).toEqual('');
  });

  it('can call getInspectionTypes', async () => {
    mockInspectionClient.getInspectionTypes.and.returnValue(
      Promise.resolve(
        create(GetInspectionTypesResponseSchema, {
          types: [
            {
              id: 'test',
              name: 'test',
              description: 'test',
              icon: 'test.png',
            },
          ],
        }),
      ),
    );

    const data = await firstValueFrom(api.getInspectionTypes());
    expect(data).toEqual({
      types: [
        {
          id: 'test',
          name: 'test',
          description: 'test',
          icon: 'test.png',
        },
      ],
    });
    expect(mockInspectionClient.getInspectionTypes).toHaveBeenCalledWith({});
  });

  it('can call getTaskStatuses', async () => {
    mockInspectionClient.getInspections.and.returnValue(
      Promise.resolve(
        create(GetInspectionsResponseSchema, {
          inspections: [
            {
              id: 'test-id',
              progress: {
                phase: InspectionPhase.RUNNING,
              },
            },
          ],
        }),
      ),
    );

    const data = await firstValueFrom(api.getInspections());
    expect(data.inspections['test-id']).toBeDefined();
    expect(data.inspections['test-id'].progress.phase).toEqual('RUNNING');
    expect(mockInspectionClient.getInspections).toHaveBeenCalledWith({});
  });

  it('can call createInspection', async () => {
    mockInspectionClient.createInspection.and.returnValue(
      Promise.resolve(
        create(CreateInspectionResponseSchema, {
          inspectionId: 'test',
        }),
      ),
    );

    const result = await firstValueFrom(
      api.createInspection('test-inspection-type'),
    );
    expect(result.inspectionID).toEqual('test');
    expect(mockInspectionClient.createInspection).toHaveBeenCalledWith({
      inspectionTypeId: 'test-inspection-type',
    });
  });

  it('can call downloadFeatureList', async () => {
    mockInspectionClient.getInspectionFeatures.and.returnValue(
      Promise.resolve(
        create(GetInspectionFeaturesResponseSchema, {
          features: [],
        }),
      ),
    );

    const data = await firstValueFrom(api.getFeatureList('test'));
    expect(data).toEqual({ features: [] });
    expect(mockInspectionClient.getInspectionFeatures).toHaveBeenCalledWith({
      inspectionId: 'test',
    });
  });

  it('can call setEnabledFeatures', async () => {
    mockInspectionClient.updateInspectionFeatures.and.returnValue(
      Promise.resolve(create(UpdateInspectionFeaturesResponseSchema, {})),
    );

    await firstValueFrom(api.setEnabledFeatures('test', {}));
    expect(mockInspectionClient.updateInspectionFeatures).toHaveBeenCalledWith(
      jasmine.objectContaining({ inspectionId: 'test' }),
    );
  });

  it('can call getInspectionMetadata', async () => {
    mockInspectionClient.getInspectionMetadata.and.returnValue(
      Promise.resolve(
        create(GetInspectionMetadataResponseSchema, {
          header: {
            inspectionType: 'test',
            inspectionName: 'test',
            inspectionTypeIconPath: 'test',
            inspectTimeUnixSeconds: 10n,
            startTimeUnixSeconds: 10n,
            endTimeUnixSeconds: 10n,
            suggestedFilename: 'test',
          },
          queries: [],
          plan: { taskGraph: '' },
          logs: [],
          error: { errorMessages: [] },
        }),
      ),
    );

    const data = await firstValueFrom(api.getInspectionMetadata('test'));
    expect(data.header.inspectionName).toEqual('test');
    expect(mockInspectionClient.getInspectionMetadata).toHaveBeenCalledWith({
      inspectionId: 'test',
    });
  });

  it('can call getInspectionData', async () => {
    const fileSize = 42;
    mockInspectionClient.getInspectionMetadata.and.returnValue(
      Promise.resolve(
        create(GetInspectionMetadataResponseSchema, {
          header: {
            inspectionType: 'test',
            inspectionName: 'test',
            suggestedFilename: 'test',
            fileSize: BigInt(fileSize),
          },
        }),
      ),
    );
    mockInspectionClient.getInspectionDataChunk.and.returnValue(
      Promise.resolve(
        create(GetInspectionDataChunkResponseSchema, {
          data: new Uint8Array(fileSize),
        }),
      ),
    );

    const data = await firstValueFrom(api.getInspectionData('test', () => {}));
    expect(data.fileName).toEqual('test');
    expect(data.content.size).toEqual(fileSize);
    expect(mockInspectionClient.getInspectionDataChunk).toHaveBeenCalled();
  });

  it('reports granular progress when getInspectionData receives chunks', async () => {
    const fileSize = 100;
    mockInspectionClient.getInspectionMetadata.and.returnValue(
      Promise.resolve(
        create(GetInspectionMetadataResponseSchema, {
          header: {
            inspectionType: 'test',
            inspectionName: 'test',
            suggestedFilename: 'test',
            fileSize: BigInt(fileSize),
          },
        }),
      ),
    );
    mockInspectionClient.getInspectionDataChunk.and.returnValue(
      Promise.resolve(
        create(GetInspectionDataChunkResponseSchema, {
          data: new Uint8Array(fileSize),
        }),
      ),
    );
    const reporterSpy = jasmine.createSpy('reporter');
    await firstValueFrom(api.getInspectionData('test', reporterSpy));
    expect(reporterSpy).toHaveBeenCalledWith(100, 100);
  });

  it('retries chunk download when getInspectionDataChunk encounters 502 error', async () => {
    const fileSize = 10;
    mockInspectionClient.getInspectionMetadata.and.returnValue(
      Promise.resolve(
        create(GetInspectionMetadataResponseSchema, {
          header: {
            inspectionType: 'test',
            inspectionName: 'test',
            suggestedFilename: 'retry-test.khi',
            fileSize: BigInt(fileSize),
          },
        }),
      ),
    );

    let callCount = 0;
    mockInspectionClient.getInspectionDataChunk.and.callFake(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.reject(new Error('502 Bad Gateway'));
      }
      return Promise.resolve(
        create(GetInspectionDataChunkResponseSchema, {
          data: new Uint8Array(fileSize),
        }),
      );
    });

    const data = await firstValueFrom(api.getInspectionData('test', () => {}));
    expect(data.fileName).toEqual('retry-test.khi');
    expect(data.content.size).toEqual(fileSize);
    expect(callCount).toBe(2);
  });

  it('respects environment download chunk size and concurrency in getInspectionData', async () => {
    const originalDownload = environment.download;
    try {
      (environment as { download: DownloadEnvironmentConfig }).download = {
        chunkSizeBytes: 20,
        maxConcurrency: 2,
      };

      const fileSize = 50;
      mockInspectionClient.getInspectionMetadata.and.returnValue(
        Promise.resolve(
          create(GetInspectionMetadataResponseSchema, {
            header: {
              inspectionType: 'test',
              inspectionName: 'test',
              suggestedFilename: 'test.khi',
              fileSize: BigInt(fileSize),
            },
          }),
        ),
      );

      mockInspectionClient.getInspectionDataChunk.and.callFake((req) => {
        const size = Number(req.maxSizeBytes);
        return Promise.resolve(
          create(GetInspectionDataChunkResponseSchema, {
            data: new Uint8Array(size),
          }),
        );
      });

      const data = await firstValueFrom(
        api.getInspectionData('test', () => {}),
      );
      expect(data.fileName).toEqual('test.khi');
      expect(data.content.size).toEqual(fileSize);
      // 50 bytes total with 20 bytes chunks => 3 chunks (20, 20, 10)
      expect(mockInspectionClient.getInspectionDataChunk).toHaveBeenCalledTimes(
        3,
      );
    } finally {
      (environment as { download: DownloadEnvironmentConfig }).download =
        originalDownload;
    }
  });

  it('initializes progress dialog immediately when downloadInspectionDataAsFile is called', async () => {
    const progressSpy = jasmine.createSpyObj<ProgressDialogStatusUpdator>(
      'Progress',
      ['show', 'updateProgress', 'dismiss'],
    );
    mockInspectionClient.getInspectionMetadata.and.returnValue(
      Promise.resolve(
        create(GetInspectionMetadataResponseSchema, {
          header: {
            suggestedFilename: 'test.khi',
            fileSize: 10n,
          },
        }),
      ),
    );
    mockInspectionClient.getInspectionDataChunk.and.returnValue(
      Promise.resolve(
        create(GetInspectionDataChunkResponseSchema, {
          data: new Uint8Array(10),
        }),
      ),
    );

    await firstValueFrom(
      BackendAPIUtil.downloadInspectionDataAsFile(api, 'test', progressSpy),
    );

    expect(progressSpy.show).toHaveBeenCalled();
    expect(progressSpy.updateProgress).toHaveBeenCalledWith({
      message: 'Downloading inspection data...',
      percent: 0,
      mode: 'determinate',
    });
    expect(progressSpy.dismiss).toHaveBeenCalled();
  });

  it('can call runTask', async () => {
    const testParameters: InspectionRunRequest = {
      test: 'foo',
    };
    mockInspectionClient.runInspection.and.returnValue(
      Promise.resolve(create(RunInspectionResponseSchema, {})),
    );

    await firstValueFrom(api.runInspection('test', testParameters));
    expect(mockInspectionClient.runInspection).toHaveBeenCalledWith(
      jasmine.objectContaining({ inspectionId: 'test' }),
    );
  });

  it('can call dryRunTask', async () => {
    const testParameters: InspectionDryRunRequest = {
      test: 'foo',
    };
    mockInspectionClient.dryRunInspection.and.returnValue(
      Promise.resolve(
        create(DryRunInspectionResponseSchema, {
          form: [],
          queries: [],
          plan: { taskGraph: '' },
        }),
      ),
    );

    const response = await firstValueFrom(
      api.dryRunInspection('test', testParameters),
    );
    expect(response.metadata).toBeDefined();
    expect(mockInspectionClient.dryRunInspection).toHaveBeenCalledWith(
      jasmine.objectContaining({ inspectionId: 'test' }),
    );
  });

  it('can call cancelInspection', async () => {
    mockInspectionClient.cancelInspection.and.returnValue(
      Promise.resolve(create(CancelInspectionResponseSchema, {})),
    );

    await firstValueFrom(api.cancelInspection('test'));
    expect(mockInspectionClient.cancelInspection).toHaveBeenCalledWith({
      inspectionId: 'test',
    });
  });
});

describe('InspectionTaskClient testing', () => {
  let taskClient: InspectionClient;
  let backendAPISpy: jasmine.SpyObj<BackendAPI>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });

    backendAPISpy = jasmine.createSpyObj<BackendAPI>('BackendAPI', [
      'getFeatureList',
      'setEnabledFeatures',
      'getInspectionMetadata',
      'runInspection',
      'dryRunInspection',
    ]);
    backendAPISpy.getFeatureList.and.returnValue(
      of({
        features: [
          {
            id: 'feat1',
            description: 'feat1',
            label: 'feat1',
            enabled: true,
          },
          {
            id: 'feat2',
            description: 'feat2',
            label: 'feat2',
            enabled: false,
          },
        ],
      }),
    );
    backendAPISpy.setEnabledFeatures.and.returnValue(of(undefined));
    backendAPISpy.runInspection.and.returnValue(of(undefined));
    backendAPISpy.dryRunInspection.and.returnValue(
      of({
        metadata: {
          query: [],
          form: [],
          plan: {
            taskGraph: 'test',
          },
        },
      }),
    );
    taskClient = new InspectionClient(
      backendAPISpy as unknown as BackendAPI,
      'test',
      new ViewStateService(),
    );
  });

  it('loads the features list at the beginning', (done) => {
    expect(backendAPISpy.getFeatureList).toHaveBeenCalledWith('test');
    taskClient.features.subscribe((features) => {
      expect(features).toEqual([
        {
          id: 'feat1',
          description: 'feat1',
          label: 'feat1',
          enabled: true,
        },
        {
          id: 'feat2',
          description: 'feat2',
          label: 'feat2',
          enabled: false,
        },
      ]);
      done();
    });
  });

  it('sets the features list by calling setFeatures', () => {
    taskClient.setFeatures(
      Object.fromEntries([
        ['feat1', true],
        ['feat2', false],
      ]),
    );
    expect(backendAPISpy.setEnabledFeatures).toHaveBeenCalledWith(
      'test',
      Object.fromEntries([
        ['feat1', true],
        ['feat2', false],
      ]),
    );
  });

  it('call run with right parameter set', (done) => {
    taskClient
      .run({
        test: 'foo',
      })
      .subscribe(() => {
        expect(backendAPISpy.runInspection).toHaveBeenCalledWith('test', {
          test: 'foo',

          timezoneShift: -new Date().getTimezoneOffset() / 60, // This parameter should come from view state
        });
        done();
      });
  });

  it('call dryrun with right parameter set', (done) => {
    const testData: InspectionDryRunResponse = {
      metadata: {
        query: [],
        form: [],
        plan: {
          taskGraph: 'test',
        },
      },
    };
    taskClient
      .dryrunDirect({
        test: 'foo',
      })
      .subscribe((response) => {
        expect(backendAPISpy.dryRunInspection).toHaveBeenCalledWith('test', {
          test: 'foo',

          timezoneShift: -new Date().getTimezoneOffset() / 60, // This parameter should come from view state
        });
        expect(response).toEqual(testData);
        done();
      });
  });

  it('call dryrunResult with right parameter set', (done) => {
    const testData: InspectionDryRunResponse = {
      metadata: {
        query: [],
        form: [],
        plan: {
          taskGraph: 'test',
        },
      },
    };
    taskClient.dryRunResult.subscribe((response) => {
      expect(response).toEqual(testData);
      done();
    });
    taskClient.dryrun({ test: 'foo' });
  });
});
