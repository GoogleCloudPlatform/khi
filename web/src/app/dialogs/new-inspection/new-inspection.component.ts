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

import {
  Component,
  inject,
  Inject,
  OnDestroy,
  signal,
  ViewChild,
} from '@angular/core';
import { MatStepper, MatStepperModule } from '@angular/material/stepper';
import {
  BehaviorSubject,
  Subject,
  filter,
  map,
  shareReplay,
  switchMap,
  take,
  takeUntil,
  withLatestFrom,
} from 'rxjs';
import {
  InspectionDryRunRequest,
  InspectionType,
} from 'src/app/common/schema/api-types';
import { ReactiveFormsModule } from '@angular/forms';
import {
  MatDialog,
  MatDialogModule,
  MatDialogRef,
} from '@angular/material/dialog';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { BACKEND_CONNECTION } from 'src/app/services/api/backend-connection.service';
import { BackendConnectionService } from 'src/app/services/api/backend-connection-interface';
import { MatCardModule } from '@angular/material/card';
import { CommonModule } from '@angular/common';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { KHICommonModule } from 'src/app/common/common.module';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import {
  DefaultParameterStore,
  PARAMETER_STORE,
} from './components/service/parameter-store';
import {
  GroupParameterFormField,
  ParameterFormField,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { GroupParameterComponent } from './components/group-parameter.component';
import {
  InspectionMetadataPlan,
  InspectionMetadataQuery,
} from 'src/app/common/schema/metadata-types';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';

export interface NewInspectionDialogResult {
  inspectionTaskStarted: boolean;
}

export interface ParameterPageViewModel {
  rootGroupForm: GroupParameterFormField;
  queries: InspectionMetadataQuery[];
  plan: InspectionMetadataPlan;
  errorFieldCount: number;
  fieldCount: number;
}

export function openNewInspectionDialog(dialog: MatDialog) {
  return dialog.open(NewInspectionDialogComponent, {
    width: '80%',
    maxWidth: '1200px',
    height: '90%',
  });
}

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
    GroupParameterComponent,
  ],
  providers: [
    {
      provide: PARAMETER_STORE,
      useClass: DefaultParameterStore,
    },
  ],
})
export class NewInspectionDialogComponent implements OnDestroy {
  static readonly STEP_INDEX_CLUSTER_TYPE = 0;
  static readonly STEP_INDEX_FEATURE_SELECTION = 1;
  static readonly STEP_INDEX_PARAMETER_INPUT = 2;

  private destroyed = new Subject<void>();

  private readonly store = inject(PARAMETER_STORE);

  /**
   * It's true only when the run button has already pressed.
   */
  public hadRun = signal(false);

  constructor(
    private readonly dialogRef: MatDialogRef<object, NewInspectionDialogResult>,
    @Inject(BACKEND_CONNECTION)
    private readonly backendConnection: BackendConnectionService,
    @Inject(BACKEND_API) private readonly apiClient: BackendAPI,
    @Inject(EXTENSION_STORE) private readonly extension: ExtensionStore,
  ) {
    this.featureToggleRequest
      .pipe(
        takeUntil(this.destroyed),
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
      .pipe(takeUntil(this.destroyed), withLatestFrom(this.currentTaskClient))
      .subscribe(([req, client]) => {
        client.dryrun(req);
      });

    // Send dryrun request to server when any of the parameters changed to validate parameters.
    this.store
      .watchAll()
      .pipe(takeUntil(this.destroyed))
      .subscribe((values) => {
        this.dryrunRequest.next(values);
      });

    // Receive the form field parameters and extract default values, then set it to the store.
    this.currentDryrunMetadata
      .pipe(takeUntil(this.destroyed))
      .subscribe((metadata) => {
        const defaultValues = this.flattenDefaultValues(metadata.form);
        this.store.setDefaultValues(defaultValues);
      });

    // Event handler reacting to the `Run` button click.
    this.startInspectionSubject
      .pipe(
        takeUntil(this.destroyed),
        take(1),
        withLatestFrom(this.currentTaskClient, this.store.watchAll()),
        switchMap(([, client, parameters]) => client.run(parameters)),
      )
      .subscribe(() => {
        this.extension.notifyLifecycleOnInspectionStart();
        this.dialogRef.close({
          inspectionTaskStarted: true,
        });
      });
  }

  @ViewChild('stepper') private stepper!: MatStepper;

  public inspectionTypes = this.backendConnection.inspectionTypes();

  public currentInspectionType = new BehaviorSubject<InspectionType | null>(
    null,
  );

  public currentTaskClient = this.currentInspectionType.pipe(
    takeUntil(this.destroyed),
    filter((type) => !!type),
    switchMap((taskType) => this.apiClient.createInspection(taskType!.id)),
    shareReplay(1),
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

  private startInspectionSubject = new Subject<void>();

  private currentDryrunMetadata = this.currentTaskClient.pipe(
    switchMap((client) => client.dryRunResult),
    map((result) => result.metadata),
  );

  parameterViewModel = this.currentDryrunMetadata.pipe(
    map((metadata) => {
      const errorFieldCount = this.countErrorFields(metadata.form);
      const fieldCount = this.countAllFields(metadata.form);
      return {
        rootGroupForm: {
          type: ParameterInputType.Group,
          children: metadata.form,
        },
        queries: metadata.query,
        plan: metadata.plan,
        errorFieldCount: errorFieldCount,
        fieldCount: fieldCount,
      } as ParameterPageViewModel;
    }),
  );

  public setInspectionType(inspectionType: InspectionType) {
    this.currentInspectionType.next(inspectionType);
    setTimeout(() => {
      this.stepper.next();
    }, 10);
  }

  public selectedStepChange(stepIndex: number) {
    if (stepIndex === NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT) {
      this.dryrunRequest.next({});
    }
  }

  public toggleFeature(featureId: string) {
    this.featureToggleRequest.next(featureId);
  }

  public onRunButtonClick() {
    this.hadRun.set(true);
    this.startInspectionSubject.next();
  }

  /**
   * Convert the array of form fields to the flatten map of default values.
   */
  private flattenDefaultValues(parameters: ParameterFormField[]): {
    [key: string]: unknown;
  } {
    let result: { [key: string]: unknown } = {};
    for (const parameter of parameters) {
      switch (parameter.type) {
        case ParameterInputType.Text:
          result[parameter.id] = parameter.default;
          break;
        case ParameterInputType.Group:
          result = {
            ...result,
            ...this.flattenDefaultValues(parameter.children),
          };
          break;
        default:
          break;
      }
    }
    return result;
  }

  /**
   * Count error fields.
   * This ignores Group type form because the group itself isn't a field.
   */
  private countErrorFields(parameters: ParameterFormField[]): number {
    let result = 0;
    for (const parameter of parameters) {
      if (parameter.type === ParameterInputType.Group) {
        result += this.countErrorFields(parameter.children);
      } else if (parameter.hintType === ParameterHintType.Error) {
        result++;
      }
    }
    return result;
  }

  /**
   * Count fields.
   * This ignores Group type form because the group itself isn't a field.
   */
  private countAllFields(parameters: ParameterFormField[]): number {
    let result = 0;
    for (const parameter of parameters) {
      if (parameter.type === ParameterInputType.Group) {
        result += this.countAllFields(parameter.children);
      } else {
        result++;
      }
    }
    return result;
  }

  ngOnDestroy(): void {
    if (this.store instanceof DefaultParameterStore) {
      this.store.destroy();
    }
    this.destroyed.next();
  }
}
