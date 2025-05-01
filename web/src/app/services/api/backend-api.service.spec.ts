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

import { HttpClient } from '@angular/common/http';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { BackendAPIImpl, InspectionClient } from './backend-api.service';
import { ViewStateService } from '../view-state.service';
import {
  CreateInspectionResponse,
  GetConfigResponse,
  GetInspectionFeatureResponse,
  GetInspectionResponse,
  GetInspectionTypesResponse,
  InspectionDryRunRequest,
  InspectionDryRunResponse,
  InspectionMetadataOfRunResult,
  InspectionRunRequest,
  PopupAnswerResponse,
  PopupAnswerValidationResult,
  PopupFormRequest,
} from '../../common/schema/api-types';
import { BackendAPI } from './backend-api-interface';
import { of } from 'rxjs';

describe('BackendAPIImpl testing', () => {
  let api: BackendAPIImpl;
  let httpTestingController: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
    });

    const httpClient = TestBed.inject(HttpClient);
    api = new BackendAPIImpl(httpClient, new ViewStateService());
    httpTestingController = TestBed.inject(HttpTestingController);
  });

  it('read server-base-path from meta tag', () => {
    document.head.innerHTML += `<meta id="server-base-path" content="/api/v3">`;
    expect(BackendAPIImpl.getServerBasePath()).toEqual('/api/v3');
    document.getElementById('server-base-path')?.remove();
    expect(BackendAPIImpl.getServerBasePath()).toEqual('');
  });

  it('calls the config API only single time when client calls getConfig method multiple times', () => {
    const testData: GetConfigResponse = {
      viewerMode: true,
    };
    api.getConfig().subscribe((data) => {
      expect(data).toEqual({
        viewerMode: true,
      });
    });
    api.getConfig().subscribe((data) => {
      expect(data).toEqual({
        viewerMode: true,
      });
    });
    const req = httpTestingController.expectOne('/api/v3/config');
    expect(req.request.method).toEqual('GET');
    req.flush(testData);
  });

  it('can call getInspectionTypes', () => {
    const testData: GetInspectionTypesResponse = {
      types: [
        {
          id: 'test',
          name: 'test',
          description: 'test',
          icon: 'test.png',
        },
      ],
    };

    api.getInspectionTypes().subscribe((data) => {
      expect(data).toEqual(testData);
    });
    const req = httpTestingController.expectOne('/api/v3/inspection/types');

    expect(req.request.method).toEqual('GET');
    req.flush(testData);
  });

  it('can call getTaskStatuses', () => {
    const testData: GetInspectionResponse = {
      inspections: {},
      serverStat: {
        totalMemoryAvailable: 10,
      },
    };

    api.getInspections().subscribe((data) => {
      expect(data).toEqual(testData);
    });
    const req = httpTestingController.expectOne('/api/v3/inspection');

    expect(req.request.method).toEqual('GET');
    req.flush(testData);
  });

  it('can call createInspection', () => {
    const testData: CreateInspectionResponse = {
      inspectionID: 'test',
    };

    api.createInspection('test-inspection-type').subscribe((result) => {
      expect(result.inspectionID).toEqual('test');
    });
    const req = httpTestingController.expectOne(
      '/api/v3/inspection/types/test-inspection-type',
    );

    expect(req.request.method).toEqual('POST');
    req.flush(testData);
  });

  it('can call downloadFeatureList', () => {
    const testData: GetInspectionFeatureResponse = {
      features: [],
    };

    api.getFeatureList('test').subscribe((data) => {
      expect(data).toEqual(testData);
    });
    const req = httpTestingController.expectOne(
      '/api/v3/inspection/test/features',
    );

    expect(req.request.method).toEqual('GET');
    req.flush(testData);
  });

  it('can call setEnabledFeatures', () => {
    const apiSpy = jasmine.createSpy();
    api.setEnabledFeatures('test', {}).subscribe(() => {
      apiSpy();
    });
    const req = httpTestingController.expectOne(
      '/api/v3/inspection/test/features',
    );

    expect(req.request.method).toEqual('PATCH');
    req.flush('ok');

    expect(apiSpy).toHaveBeenCalledOnceWith();
  });

  it('can call getInspectionMetadata', () => {
    const testData: InspectionMetadataOfRunResult = {
      header: {
        inspectionType: 'test',
        inspectionTypeIconPath: 'test',
        inspectTimeUnixSeconds: 10,
        startTimeUnixSeconds: 10,
        endTimeUnixSeconds: 10,
        suggestedFilename: 'test',
      },
      query: [],
      plan: {
        taskGraph: '',
      },
      log: [],
      error: {
        errorMessages: [],
      },
    };

    api.getInspectionMetadata('test').subscribe((data) => {
      expect(data).toEqual(testData);
    });
    const req = httpTestingController.expectOne(
      '/api/v3/inspection/test/metadata',
    );

    expect(req.request.method).toEqual('GET');
    req.flush(testData);
  });

  it('can call getInspectionData', () => {
    const fileSize = 42;
    const testMetadata: InspectionMetadataOfRunResult = {
      header: {
        inspectionType: 'test',
        inspectionTypeIconPath: 'test',
        inspectTimeUnixSeconds: 10,
        startTimeUnixSeconds: 10,
        endTimeUnixSeconds: 10,
        suggestedFilename: 'test',
        fileSize: 42,
      },
      query: [],
      plan: {
        taskGraph: '',
      },
      log: [],
      error: {
        errorMessages: [],
      },
    };
    api
      .getInspectionData('test', () => {})
      .subscribe((data) => {
        expect(data.fileName).toEqual('test');
        expect(data.content).toEqual(testData);
      });
    const testData = new Blob([new ArrayBuffer(fileSize)]);
    const req0 = httpTestingController.expectOne(
      '/api/v3/inspection/test/metadata',
    );
    expect(req0.request.method).toEqual('GET');
    req0.flush(testMetadata);
    const req1 = httpTestingController.expectOne(
      `/api/v3/inspection/test/data?start=0&maxSize=${fileSize}`,
    );
    expect(req1.request.method).toEqual('GET');
    req1.flush(testData);
  });

  it('can call runTask', () => {
    const testParameters: InspectionRunRequest = {
      test: 'foo',
    };

    api.runInspection('test', testParameters).subscribe(() => {});
    const req = httpTestingController.expectOne('/api/v3/inspection/test/run');

    expect(req.request.method).toEqual('POST');
    expect(req.request.body).toEqual(testParameters);
    req.flush('');
  });

  it('can call dryRunTask', (done) => {
    const testParameters: InspectionDryRunRequest = {
      test: 'foo',
    };
    const testResponse: InspectionDryRunResponse = {
      metadata: {
        form: [],
        query: [],
        plan: {
          taskGraph: '',
        },
      },
    };

    api.dryRunInspection('test', testParameters).subscribe((response) => {
      expect(response).toBe(testResponse);
      done();
    });
    const req = httpTestingController.expectOne(
      '/api/v3/inspection/test/dryrun',
    );

    expect(req.request.method).toEqual('POST');
    expect(req.request.body).toEqual(testParameters);
    req.flush(testResponse);
  });

  it('can call cancelInspection', () => {
    api.cancelInspection('test').subscribe(() => {});
    const req = httpTestingController.expectOne(
      '/api/v3/inspection/test/cancel',
    );
    expect(req.request.method).toEqual('POST');

    req.flush('');
  });

  it('can call getPopup', (done) => {
    const testResponse: PopupFormRequest = {
      id: 'test',
      title: 'test',
      type: 'text',
      description: 'test',
      placeholder: 'test',
      options: {},
    };

    api.getPopup().subscribe((data) => {
      expect(data).toBe(testResponse);
      done();
    });
    const req = httpTestingController.expectOne('/api/v3/popup');

    expect(req.request.method).toBe('GET');
    req.flush(testResponse);
  });

  it('can call validatePopupAnswer', (done) => {
    const testRequest: PopupAnswerResponse = {
      id: 'test',
      value: 'foo',
    };
    const testResponse: PopupAnswerValidationResult = {
      id: 'test',
      validationError: 'foo',
    };

    api.validatePopupAnswer(testRequest).subscribe((data) => {
      expect(data).toBe(testResponse);
      done();
    });
    const req = httpTestingController.expectOne('/api/v3/popup/validate');

    expect(req.request.method).toBe('POST');
    expect(req.request.body).toBe(testRequest);
    req.flush(testResponse);
  });

  it('can call answerPopup', (done) => {
    const testRequest: PopupAnswerResponse = {
      id: 'test',
      value: 'foo',
    };

    api.answerPopup(testRequest).subscribe(() => {
      done();
    });
    const req = httpTestingController.expectOne('/api/v3/popup/answer');

    expect(req.request.method).toBe('POST');
    expect(req.request.body).toBe(testRequest);
    req.flush('ok');
  });
});

describe('InspectionTaskClient testing', () => {
  let taskClient: InspectionClient;
  let backendAPISpy: jasmine.SpyObj<BackendAPI>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
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
