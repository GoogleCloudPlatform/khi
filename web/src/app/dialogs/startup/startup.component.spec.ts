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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StartupDialogComponent } from './startup.component';
import { MatDialogRef } from '@angular/material/dialog';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { ProgressDialogService } from 'src/app/services/progress/progress-dialog.service';
import { BACKEND_CONNECTION } from 'src/app/services/api/backend-connection.service';
import { BackendConnectionService } from 'src/app/services/api/backend-connection-interface';
import { of, ReplaySubject, Subject } from 'rxjs';
import {
  GetConfigResponse,
  GetInspectionResponse,
} from 'src/app/common/schema/api-types';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';

describe('StartupDialogComponent', () => {
  let component: ComponentFixture<StartupDialogComponent>;
  let backendConnectionSpy: jasmine.SpyObj<BackendConnectionService>;
  let backendAPISpy: jasmine.SpyObj<BackendAPI>;
  let taskListSubject: Subject<GetInspectionResponse>;
  beforeEach(async () => {
    taskListSubject = new ReplaySubject(1);
    backendConnectionSpy = jasmine.createSpyObj<BackendConnectionService>(
      'BackendConnectionService',
      ['tasks'],
    );
    backendConnectionSpy.tasks.and.returnValue(taskListSubject);
    backendAPISpy = jasmine.createSpyObj<BackendAPI>('BackendAPIService', [
      'getConfig',
      'patchInspection',
    ]);
    backendAPISpy.getConfig.and.returnValue(
      of<GetConfigResponse>({
        viewerMode: false,
      }),
    );
    backendAPISpy.patchInspection.and.returnValue(of());
    TestBed.configureTestingModule({
      providers: [
        ...ProgressDialogService.providers(),
        {
          provide: MatDialogRef,
          useValue: {},
        },
        {
          provide: BACKEND_API,
          useValue: backendAPISpy,
        },
        {
          provide: BACKEND_CONNECTION,
          useValue: backendConnectionSpy,
        },
        {
          provide: EXTENSION_STORE,
          useValue: new ExtensionStore(),
        },
        {
          provide: InspectionDataLoaderService,
          useClass: InspectionDataLoaderService,
        },
      ],
    });
    component = TestBed.createComponent(StartupDialogComponent);
    component.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should save title to backend', () => {
    component.componentInstance.updateInspectionTitle({
      id: 'test-task',
      changeTo: 'New Title',
    });
    expect(backendAPISpy.patchInspection).toHaveBeenCalledOnceWith(
      'test-task',
      {
        name: 'New Title',
      },
    );
  });
});
