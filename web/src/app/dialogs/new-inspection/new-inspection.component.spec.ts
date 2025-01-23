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
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatOptionModule } from '@angular/material/core';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

import { NewInspectionDialogComponent } from './new-inspection.component';
import { MatStepperModule } from '@angular/material/stepper';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { BACKEND_API } from 'src/app/services/api/backend-api-interface';
import { BACKEND_CONNECTION } from 'src/app/services/api/backend-connection.service';
import { BackendConnectionService } from 'src/app/services/api/backend-connection-interface';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';

describe('NewInspectionDialogTest', () => {
  let component: NewInspectionDialogComponent;
  let fixture: ComponentFixture<NewInspectionDialogComponent>;
  let backendConnectionSpy: jasmine.SpyObj<BackendConnectionService>;

  beforeEach(async () => {
    backendConnectionSpy = jasmine.createSpyObj<BackendConnectionService>(
      'BackendConnectionService',
      ['inspectionTypes'],
    );
    await TestBed.configureTestingModule({
      declarations: [NewInspectionDialogComponent],
      imports: [
        NoopAnimationsModule,
        MatDialogModule,
        MatButtonModule,
        MatSelectModule,
        MatOptionModule,
        MatInputModule,
        MatCheckboxModule,
        ReactiveFormsModule,
        FormsModule,
        MatAutocompleteModule,
        MatStepperModule,
        MatProgressBarModule,
      ],
      providers: [
        {
          provide: MatDialogRef,
          useValue: null,
        },
        {
          provide: BACKEND_API,
          useValue: {},
        },
        {
          provide: BACKEND_CONNECTION,
          useValue: backendConnectionSpy,
        },
        {
          provide: EXTENSION_STORE,
          useValue: new ExtensionStore(),
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(NewInspectionDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
