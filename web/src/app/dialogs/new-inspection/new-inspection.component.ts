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
  computed,
  inject,
  OnDestroy,
  signal,
  ViewChild,
} from '@angular/core';
import { MatStepper, MatStepperModule } from '@angular/material/stepper';
import {
  BehaviorSubject,
  Subject,
  filter,
  firstValueFrom,
  fromEvent,
  map,
  shareReplay,
  switchMap,
  take,
  takeUntil,
  withLatestFrom,
} from 'rxjs';
import {
  InspectionMetadataInDryrun,
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
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { BackendSyncService } from 'src/app/services/api/backend-sync-interface';
import { MatCardModule } from '@angular/material/card';
import { CommonModule } from '@angular/common';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { KHICommonModule } from 'src/app/common/common.module';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import {
  DefaultParameterStore,
  haveEqualKeyValues,
  PARAMETER_STORE,
} from './components/service/parameter-store';
import {
  GroupParameterFormField,
  ParameterFormField,
  ParameterHintType,
  ParameterInputType,
} from 'src/app/common/schema/form-types';
import { GroupParameterComponent } from './components/group-parameter.component';
import { JobCommandComponent } from 'src/app/dialogs/new-inspection/components/job-command.component';
import {
  InspectionMetadataPlan,
  InspectionMetadataQuery,
  InspectionMetadataJobModeCommand,
  EstimatedCountPreset,
} from 'src/app/common/schema/metadata-types';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';

/**
 * Error indicating that an asynchronous operation or delay was cancelled.
 */
export class CancellationError extends Error {
  constructor(message = 'Operation was cancelled') {
    super(message);
    this.name = 'CancellationError';
  }
}

export interface NewInspectionDialogResult {
  readonly inspectionTaskStarted: boolean;
}

/**
 * Severity level based on estimated log volume.
 */
export enum TotalEstimatedLogsSeverity {
  Normal = 'normal',
  Warning = 'warning',
  Danger = 'danger',
}

/**
 * Summary of total estimated log count across all queries.
 */
export interface TotalEstimatedLogsSummary {
  readonly knownCount: number;
  readonly isComplete: boolean;
  readonly isEstimating: boolean;
  readonly isIncomplete: boolean;
  readonly displayText: string;
  readonly severity: TotalEstimatedLogsSeverity;
}

/**
 * Computes the aggregate estimated log count summary across all queries.
 */
export function computeTotalEstimatedLogs(
  queries?: InspectionMetadataQuery[],
): TotalEstimatedLogsSummary | undefined {
  if (!queries || queries.length === 0) {
    return undefined;
  }
  const hasIncomplete = queries.some((q) => q.incomplete);
  const hasUnestimated = queries.some(
    (q) =>
      (q.estimatedCount === undefined || q.pending) &&
      !q.incomplete &&
      (!q.estimatedCountPreset ||
        q.estimatedCountPreset === EstimatedCountPreset.None),
  );
  const hasPreset = queries.some(
    (q) =>
      q.estimatedCountPreset &&
      q.estimatedCountPreset !== EstimatedCountPreset.None,
  );
  const knownCount = queries
    .filter((q) => q.estimatedCount !== undefined && !q.pending)
    .reduce((sum, q) => sum + (q.estimatedCount ?? 0), 0);

  let severity = TotalEstimatedLogsSeverity.Normal;
  if (knownCount >= 5_000_000) {
    severity = TotalEstimatedLogsSeverity.Danger;
  } else if (knownCount >= 1_000_000) {
    severity = TotalEstimatedLogsSeverity.Warning;
  }

  const formattedCount = knownCount.toLocaleString('en-US');
  if (hasIncomplete) {
    if (knownCount > 0) {
      return {
        knownCount,
        isComplete: false,
        isEstimating: hasUnestimated,
        isIncomplete: true,
        displayText: `>${formattedCount} logs estimated (some parameters incomplete)`,
        severity,
      };
    }
    return {
      knownCount: 0,
      isComplete: false,
      isEstimating: hasUnestimated,
      isIncomplete: true,
      displayText: 'Incomplete parameters',
      severity: TotalEstimatedLogsSeverity.Normal,
    };
  }

  if (hasUnestimated) {
    if (knownCount > 0) {
      return {
        knownCount,
        isComplete: false,
        isEstimating: true,
        isIncomplete: false,
        displayText: `>${formattedCount} logs estimated so far`,
        severity,
      };
    }
    return {
      knownCount: 0,
      isComplete: false,
      isEstimating: true,
      isIncomplete: false,
      displayText: 'Estimating total logs...',
      severity: TotalEstimatedLogsSeverity.Normal,
    };
  }

  if (knownCount === 0 && hasPreset) {
    return {
      knownCount: 0,
      isComplete: true,
      isEstimating: false,
      isIncomplete: false,
      displayText: 'Few total logs estimated',
      severity,
    };
  }

  return {
    knownCount,
    isComplete: true,
    isEstimating: false,
    isIncomplete: false,
    displayText: `~${formattedCount} total logs estimated`,
    severity,
  };
}

export interface ParameterPageViewModel {
  readonly rootGroupForm: GroupParameterFormField;
  readonly queries: InspectionMetadataQuery[];
  readonly plan: InspectionMetadataPlan;
  readonly job?: InspectionMetadataJobModeCommand;
  readonly errorFieldCount: number;
  readonly pendingFieldCount: number;
  readonly fieldCount: number;
  readonly totalEstimatedSummary?: TotalEstimatedLogsSummary;
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
  styleUrls: ['./new-inspection.component.scss'],
  imports: [
    CommonModule,
    KHICommonModule,
    MatButtonModule,
    MatInputModule,
    MatDialogModule,
    MatStepperModule,
    MatCardModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatIconModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatAutocompleteModule,
    GroupParameterComponent,
    JobCommandComponent,
  ],
  providers: [
    {
      provide: PARAMETER_STORE,
      useClass: DefaultParameterStore,
    },
  ],
})
export class NewInspectionDialogComponent implements OnDestroy {
  protected readonly EstimatedCountPreset = EstimatedCountPreset;

  private readonly dialogRef =
    inject<MatDialogRef<object, NewInspectionDialogResult>>(MatDialogRef);
  private readonly backendSync = inject<BackendSyncService>(BACKEND_SYNC);
  private readonly apiClient = inject<BackendAPI>(BACKEND_API);
  private readonly extension = inject<ExtensionStore>(EXTENSION_STORE);

  static readonly STEP_INDEX_CLUSTER_TYPE = 0;
  static readonly STEP_INDEX_FEATURE_SELECTION = 1;
  static readonly STEP_INDEX_PARAMETER_INPUT = 2;

  private destroyed = new Subject<void>();

  private readonly store = inject(PARAMETER_STORE);

  /**
   * It's true only when the run button has already pressed.
   */
  public hadRun = signal(false);

  constructor() {
    this.featureToggleRequest
      .pipe(
        takeUntil(this.destroyed),
        withLatestFrom(this.featureStatusMap),
        map(([featureId, currentFeatures]) => {
          return Object.fromEntries([[featureId, !currentFeatures[featureId]]]);
        }),
        withLatestFrom(this.currentTaskClient),
      )
      .subscribe(([featureIds, client]) => {
        client.setFeatures(featureIds);
      });
    // Event handler reacting to the `Run` button click.
    this.startInspectionSubject
      .pipe(
        takeUntil(this.destroyed),
        take(1),
        withLatestFrom(this.currentTaskClient),
        switchMap(([, client]) => client.run(this.store.currentParameters())),
      )
      .subscribe(() => {
        this.extension.notifyLifecycleOnInspectionStart();
        this.dialogRef.close({
          inspectionTaskStarted: true,
        });
      });
  }

  @ViewChild('stepper') private stepper!: MatStepper;

  public inspectionTypes = this.backendSync.inspectionTypes;

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

  /**
   * A map of feature id and its status - true if enabled
   */
  public featureStatusMap = this.currentTaskFeatures.pipe(
    map((features) =>
      Object.fromEntries(
        features.map((feature) => [feature.id, feature.enabled]),
      ),
    ),
  );

  public featuresEnabled = this.currentTaskFeatures.pipe(
    map((features) => features.some((f) => f.enabled)),
  );

  private featureToggleRequest = new Subject<string>();

  private startInspectionSubject = new Subject<void>();

  /**
   * parameterViewModel contains the current ParameterPageViewModel.
   * Holds null initially until the first dryrun finishes, or when reset.
   */
  readonly parameterViewModel = signal<ParameterPageViewModel | null>(null);

  /**
   * Abort controller for cancelling the ongoing dryrun loop.
   */
  private loopAbortController: AbortController | null = null;

  /**
   * Computed signal of pending field count.
   * Counts fields that are currently undergoing server pending task or client validation.
   */
  readonly pendingFieldCount = computed(() => {
    const vm = this.parameterViewModel();
    if (!vm) {
      return 0;
    }
    return this.countPendingFields(vm.rootGroupForm.children);
  });

  /**
   * Computed signal of error field count.
   * Suppresses stale errors for fields that are currently undergoing validation or server pending.
   */
  readonly errorFieldCount = computed(() => {
    const vm = this.parameterViewModel();
    if (!vm) {
      return 0;
    }
    return this.countErrorFields(vm.rootGroupForm.children);
  });

  /**
   * Computed signal indicating if the run button should be disabled.
   */
  readonly isRunButtonDisabled = computed(() => {
    return (
      this.hadRun() ||
      this.errorFieldCount() !== 0 ||
      this.pendingFieldCount() !== 0
    );
  });

  public setInspectionType(inspectionType: InspectionType) {
    this.currentInspectionType.next(inspectionType);
    setTimeout(() => {
      this.stepper.next();
    }, 10);
  }

  public selectedStepChange(stepIndex: number) {
    if (stepIndex === NewInspectionDialogComponent.STEP_INDEX_PARAMETER_INPUT) {
      // Reset the parameter view model every time entering STEP_INDEX_PARAMETER_INPUT otherwise parameter list can be stale.
      this.parameterViewModel.set(null);
      this.startDryrunLoop();
    } else {
      this.stopDryrunLoop();
    }
  }

  /**
   * Runs the dryrun loop sequentially. Executes a dryrun request, checks if user input changed during flight,
   * updates the store and view model if unchanged, or immediately re-runs if inputs changed while in-flight.
   */
  private async startDryrunLoop() {
    this.stopDryrunLoop();
    const abortController = new AbortController();
    this.loopAbortController = abortController;
    const signal = abortController.signal;

    const client = await firstValueFrom(this.currentTaskClient);
    if (signal.aborted || !client) {
      return;
    }

    while (!signal.aborted) {
      const sentParams = this.store.currentParameters();
      try {
        const res = await firstValueFrom(
          client
            .dryrunDirect(sentParams)
            .pipe(takeUntil(fromEvent(signal, 'abort'))),
        );
        if (signal.aborted) {
          break;
        }

        const currentParams = this.store.currentParameters();
        const changedDuringFlight = !haveEqualKeyValues(
          sentParams,
          currentParams,
        );

        if (!changedDuringFlight) {
          this.store.setValidatedParameters(sentParams);
          this.updateParameterViewModel(res.metadata);
          this.store.setDefaultValues(
            this.flattenDefaultValues(res.metadata.form),
          );

          // If setDefaultValues assigned new defaults, currentParameters now differs from sentParams.
          // In this case, do not delay so the next dryrun validating those defaults executes immediately.
          const paramsAfterDefaults = this.store.currentParameters();
          if (haveEqualKeyValues(sentParams, paramsAfterDefaults)) {
            await this.delay(800, signal);
          }
        }
        // If changedDuringFlight is true, immediately loop without delay to validate the new inputs.
      } catch (err) {
        if (signal.aborted || err instanceof CancellationError) {
          break;
        }
        try {
          await this.delay(1000, signal);
        } catch {
          if (signal.aborted) {
            break;
          }
        }
      }
    }
  }

  private stopDryrunLoop() {
    if (this.loopAbortController) {
      this.loopAbortController.abort();
      this.loopAbortController = null;
    }
  }

  private delay(ms: number, signal: AbortSignal): Promise<void> {
    if (signal.aborted) {
      return Promise.reject(new CancellationError('Delay aborted'));
    }
    return new Promise((resolve, reject) => {
      const timer = setTimeout(resolve, ms);
      signal.addEventListener(
        'abort',
        () => {
          clearTimeout(timer);
          reject(new CancellationError('Delay aborted'));
        },
        { once: true },
      );
    });
  }

  private updateParameterViewModel(metadata: InspectionMetadataInDryrun) {
    const errorFieldCount = this.countErrorFields(metadata.form);
    const pendingFieldCount = this.countPendingFields(metadata.form);
    const fieldCount = this.countAllFields(metadata.form);
    this.parameterViewModel.set({
      rootGroupForm: {
        id: 'root',
        label: '',
        description: '',
        hint: '',
        hintType: ParameterHintType.None,
        type: ParameterInputType.Group,
        collapsible: false,
        collapsedByDefault: false,
        children: metadata.form,
      },
      queries: metadata.query,
      plan: metadata.plan,
      job: metadata.jobCommand,
      errorFieldCount: errorFieldCount,
      pendingFieldCount: pendingFieldCount,
      fieldCount: fieldCount,
      totalEstimatedSummary: computeTotalEstimatedLogs(metadata.query),
    });
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
        case ParameterInputType.Set:
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
   * Suppresses error count when the field is pending or validating.
   * This ignores Group type form because the group itself isn't a field.
   */
  private countErrorFields(parameters: readonly ParameterFormField[]): number {
    let result = 0;
    for (const parameter of parameters) {
      if (parameter.type === ParameterInputType.Group) {
        result += this.countErrorFields(parameter.children);
      } else {
        const isClientValidating = this.store.isValidating(parameter.id)();
        const isPending = !!parameter.pending || isClientValidating;
        if (parameter.hintType === ParameterHintType.Error && !isPending) {
          result++;
        }
      }
    }
    return result;
  }

  /**
   * Count pending fields.
   * This ignores Group type form because the group itself isn't a field.
   */
  private countPendingFields(
    parameters: readonly ParameterFormField[],
  ): number {
    let result = 0;
    for (const parameter of parameters) {
      if (parameter.type === ParameterInputType.Group) {
        result += this.countPendingFields(parameter.children);
      } else {
        const isClientValidating = this.store.isValidating(parameter.id)();
        if (parameter.pending || isClientValidating) {
          result++;
        }
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
    this.stopDryrunLoop();
    if (this.store instanceof DefaultParameterStore) {
      this.store.destroy();
    }
    this.destroyed.next();
  }
}
