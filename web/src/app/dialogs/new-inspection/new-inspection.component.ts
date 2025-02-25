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

import { Component, Inject, OnDestroy, ViewChild } from '@angular/core';
import { MatStepper, MatStepperModule } from '@angular/material/stepper';
import {
  BehaviorSubject,
  Subject,
  filter,
  map,
  share,
  startWith,
  switchMap,
  take,
  takeUntil,
  withLatestFrom,
} from 'rxjs';
import {
  InspectionDryRunRequest,
  InspectionMetadataInDryrun,
  InspectionType,
} from 'src/app/common/schema/api-types';
import { FormControl, FormGroup, ReactiveFormsModule } from '@angular/forms';
import {
  MatDialog,
  MatDialogModule,
  MatDialogRef,
} from '@angular/material/dialog';
import { InspectionMetadataFormField } from 'src/app/common/schema/metadata-types';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { BACKEND_CONNECTION } from 'src/app/services/api/backend-connection.service';
import { BackendConnectionService } from 'src/app/services/api/backend-connection-interface';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';
import { MatCardModule } from '@angular/material/card';
import { CommonModule } from '@angular/common';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { KHICommonModule } from 'src/app/common/common.module';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';

export interface NewInspectionDialogResult {
  inspectionTaskStarted: boolean;
}

export function openNewInspectionDialog(dialog: MatDialog) {
  return dialog.open(NewInspectionDialogComponent, {
    width: '80%',
    maxWidth: '1200px',
    height: '90%',
  });
}

const initCurrentValue: FormFieldValues = {};

type FormFieldValues = { [key: string]: string };

type FormFieldViewModel = {
  formGroup: FormGroup;
  metadata: InspectionMetadataInDryrun;
  fieldCount: number;
  errorCount: number;
};

@Component({
  templateUrl: './new-inspection.component.html',
  styleUrls: ['./new-inspection.component.sass'],
  imports: [
    CommonModule,
    KHICommonModule,
    MatButtonModule,
    MatInputModule,
    MatDialogModule,
    MatStepperModule,
    MatCardModule,
    MatProgressBarModule,
    MatIconModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatAutocompleteModule,
  ],
})
export class NewInspectionDialogComponent implements OnDestroy {
  private destoroyed = new Subject<void>();

  static readonly STEP_INDEX_CLUSTER_TYPE = 0;
  static readonly STEP_INDEX_FEATURE_SELECTION = 1;
  static readonly STEP_INDEX_PARAMETER_INPUT = 2;

  constructor(
    private readonly dialogRef: MatDialogRef<object, NewInspectionDialogResult>,
    @Inject(BACKEND_CONNECTION)
    private readonly backendConnection: BackendConnectionService,
    @Inject(BACKEND_API) private readonly apiClient: BackendAPI,
    @Inject(EXTENSION_STORE) private readonly extension: ExtensionStore,
  ) {
    this.featureToggleRequest
      .pipe(
        takeUntil(this.destoroyed),
        withLatestFrom(this.currentEnabledFeatures),
        map(([toggleFeature, features]) => {
          if (!features.has(toggleFeature)) {
            return [...features, toggleFeature];
          } else {
            return [...features].filter((f) => f !== toggleFeature);
          }
        }),
        withLatestFrom(this.currentTaskClient),
      )
      .subscribe(([featureIds, client]) => {
        client.setFeatures(featureIds);
      });
    this.dryrunRequest
      .pipe(takeUntil(this.destoroyed), withLatestFrom(this.currentTaskClient))
      .subscribe(([req, client]) => {
        client.dryrun(req);
      });
    this.formViewModel
      .pipe(
        takeUntil(this.destoroyed),
        switchMap((fv) => fv.formGroup.valueChanges),
      )
      .subscribe((values) => {
        this.currentValues.next(values);
        this.dryrunRequest.next(values);
      });
    this.runRequest
      .pipe(
        takeUntil(this.destoroyed),
        withLatestFrom(this.currentTaskClient, this.currentValues),
        take(1), // Block multiple clicks
        switchMap(([, client, values]) => client.run(values)),
      )
      .subscribe(() => {
        this.extension.notifyLifecycleOnInspectionStart();
        this.dialogRef.close({
          inspectionTaskStarted: true,
        });
      });
  }

  ngOnDestroy(): void {
    this.destoroyed.next();
  }

  @ViewChild('stepper') private stepper!: MatStepper;

  public inspectionTypes = this.backendConnection.inspectionTypes();

  public currentInspectionType = new BehaviorSubject<InspectionType | null>(
    null,
  );

  public currentTaskClient = this.currentInspectionType.pipe(
    filter((type) => !!type),
    switchMap((taskType) => this.apiClient.createInspection(taskType!.id)),
    share(),
  );

  public currentTaskFeatures = this.currentTaskClient.pipe(
    switchMap((tc) => tc.features),
  );

  public currentEnabledFeatures = this.currentTaskFeatures.pipe(
    map(
      (features) => new Set(features.filter((f) => f.enabled).map((f) => f.id)),
    ),
  );

  private featureToggleRequest = new Subject<string>();

  private dryrunRequest = new Subject<InspectionDryRunRequest>();

  private currentDryrunMetadata = this.currentTaskClient.pipe(
    switchMap((client) => client.dryRunResult),
    map((result) => result.metadata),
  );

  /**
   * A behavior emits values when the input values were changed
   */
  private currentValues = new BehaviorSubject<FormFieldValues>(
    initCurrentValue,
  );

  private runRequest = new Subject<null>();

  public formViewModel = this.currentDryrunMetadata.pipe(
    withLatestFrom(this.currentValues.pipe(startWith(initCurrentValue))),
    map(([metadata, values]) => {
      const fields: { [key: string]: FormControl } = {};
      const currentValues: FormFieldValues = {};
      let errorCount = 0;
      for (const field of metadata.form) {
        let value = field.default;
        if (field.id in values) {
          value = values[field.id];
        }
        currentValues[field.id] = value;
        const control = new FormControl<string>(value);
        if (!field.allowEdit) {
          control.disable();
        }
        if (field.validationError != '') {
          errorCount += 1;
        }
        fields[field.id] = control;
      }
      if (values === initCurrentValue) {
        this.currentValues.next(currentValues);
      }
      return {
        formGroup: new FormGroup(fields),
        metadata: metadata,
        fieldCount: metadata.form.length,
        errorCount: errorCount,
      } as FormFieldViewModel;
    }),
    share(),
  );

  public setInspectionType(inspectionType: InspectionType) {
    this.currentInspectionType.next(inspectionType);
    setTimeout(() => {
      this.stepper.next();
    }, 10);
  }

  public selectedStepChange(stepIndex: number) {
    if (stepIndex === NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT) {
      this.dryrunRequest.next(this.currentValues.value);
    }
  }

  public toggleFeature(featureId: string) {
    this.featureToggleRequest.next(featureId);
  }

  /**
   * track function for ngFor loop in displaying fields.
   * Need this track mechanism not to recreate form fields and lose focus due to the field collection regeneration.
   */
  public fieldCollectionTrack(
    index: number,
    field: InspectionMetadataFormField,
  ): string {
    return field.id;
  }

  public run() {
    this.runRequest.next(null);
  }
}
