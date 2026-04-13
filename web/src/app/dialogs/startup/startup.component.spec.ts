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
import { signal } from '@angular/core';
import { MatDialogRef } from '@angular/material/dialog';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { ProgressDialogService } from 'src/app/services/progress/progress-dialog.service';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';

import { of } from 'rxjs';
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

  let backendAPISpy: jasmine.SpyObj<BackendAPI>;

  beforeEach(async () => {
    const tasksSignal = signal<GetInspectionResponse>({
      inspections: {},
      serverStat: { currentMemoryUsage: 0, totalMemory: 0 },
    });

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
          provide: BACKEND_SYNC,
          useValue: {
            tasks: {
              value: tasksSignal,
            },
          },
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
