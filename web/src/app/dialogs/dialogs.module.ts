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

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ProgressDialogComponent } from './progress/progress.component';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatDialogModule } from '@angular/material/dialog';
import { NewInspectionDialogComponent } from './new-inspection/new-inspection.component';
import { MatFormFieldModule } from '@angular/material/form-field';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatOptionModule } from '@angular/material/core';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { InspectionMetadataDialogComponent } from './inspection-metadata/inspection-metadata.component';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { MatStepperModule } from '@angular/material/stepper';
import { MatCardModule } from '@angular/material/card';

import { MatSelectModule } from '@angular/material/select';
import { KHICommonModule } from '../common/common.module';
import { StartupDialogComponent } from './startup/startup.component';
import { NgxEnvModule } from '@ngx-env/core';
import { MatTooltipModule } from '@angular/material/tooltip';

@NgModule({
  declarations: [
    ProgressDialogComponent,
    NewInspectionDialogComponent,
    InspectionMetadataDialogComponent,
  ],
  imports: [
    CommonModule,
    KHICommonModule,
    MatInputModule,
    MatProgressBarModule,
    MatDialogModule,
    MatStepperModule,
    MatSelectModule,
    MatFormFieldModule,
    ReactiveFormsModule,
    FormsModule,
    MatAutocompleteModule,
    MatOptionModule,
    MatCheckboxModule,
    MatIconModule,
    MatButtonModule,
    MatCardModule,
    NgxEnvModule,
    MatTooltipModule,
    StartupDialogComponent,
  ],
  exports: [
    ProgressDialogComponent,
    NewInspectionDialogComponent,
    InspectionMetadataDialogComponent,
  ],
})
export class DialogsModule {}
