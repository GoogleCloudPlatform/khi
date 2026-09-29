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

import {
  Component,
  computed,
  DestroyRef,
  inject,
  OnDestroy,
  signal,
} from '@angular/core';
import {
  takeUntilDestroyed,
  toObservable,
  toSignal,
} from '@angular/core/rxjs-interop';
import {
  MAT_DIALOG_DATA,
  MatDialog,
  MatDialogRef,
} from '@angular/material/dialog';
import {
  BehaviorSubject,
  filter,
  firstValueFrom,
  fromEvent,
  map,
  shareReplay,
  switchMap,
  take,
  takeUntil,
  tap,
} from 'rxjs';
import {
  GetInspectionTypesResponse,
  InspectionType,
} from 'src/app/common/schema/api-types';
import {
  BACKEND_API,
  BackendAPI,
} from 'src/app/services/api/backend-api-interface';
import { InspectionClient } from 'src/app/services/api/backend-api.service';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { BackendSyncService } from 'src/app/services/api/backend-sync-interface';
import {
  DefaultParameterStore,
  haveEqualKeyValues,
  PARAMETER_STORE,
} from 'src/app/dialogs/new-inspection/components/service/parameter-store';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';
import { NewInspectionLayoutComponent } from 'src/app/dialogs/new-inspection/components/new-inspection-layout.component';
import {
  CancellationError,
  NewInspectionDialogData,
  NewInspectionDialogResult,
  NewInspectionStepIndex,
  ParameterStepViewModel,
} from 'src/app/dialogs/new-inspection/types/new-inspection.types';
import {
  buildParameterStepViewModel,
  flattenDefaultValues,
} from 'src/app/dialogs/new-inspection/utils/new-inspection.utils';

/**
 * Opens the New Inspection dialog.
 *
 * @param dialog The Angular Material MatDialog service instance.
 * @param data Optional prefilled configuration data.
 * @returns Reference to the opened dialog.
 */
export function openNewInspectionDialog(
  dialog: MatDialog,
  data?: NewInspectionDialogData,
): MatDialogRef<NewInspectionDialogComponent, NewInspectionDialogResult> {
  return dialog.open<
    NewInspectionDialogComponent,
    NewInspectionDialogData,
    NewInspectionDialogResult
  >(NewInspectionDialogComponent, {
    width: '80%',
    maxWidth: '1200px',
    height: '90%',
    data,
  });
}

/**
 * Smart component for the New Inspection wizard dialog.
 */
@Component({
  templateUrl: './new-inspection-smart.component.html',
  styleUrl: './new-inspection-smart.component.scss',
  imports: [NewInspectionLayoutComponent],
  providers: [
    {
      provide: PARAMETER_STORE,
      useClass: DefaultParameterStore,
    },
  ],
})
export class NewInspectionDialogComponent implements OnDestroy {
  private readonly dialogRef =
    inject<MatDialogRef<object, NewInspectionDialogResult>>(MatDialogRef);
  private readonly dialogData = inject<NewInspectionDialogData | null>(
    MAT_DIALOG_DATA,
    { optional: true },
  );
  private readonly backendSync = inject<BackendSyncService>(BACKEND_SYNC);
  private readonly apiClient = inject<BackendAPI>(BACKEND_API);
  private readonly extension = inject<ExtensionStore>(EXTENSION_STORE);

  /**
   * Parameter store holding current, default, and validated parameter values.
   */
  readonly parameterStore = inject(PARAMETER_STORE);

  private readonly destroyRef = inject(DestroyRef);

  /**
   * The currently selected step index in the wizard.
   */
  readonly selectedStepIndex = signal<NewInspectionStepIndex>(
    NewInspectionStepIndex.InspectionType,
  );

  /**
   * Computed list of available inspection types.
   */
  readonly inspectionTypes = computed<readonly InspectionType[] | null>(
    () => this.backendSync.inspectionTypes.value()?.types ?? null,
  );

  private readonly currentInspectionTypeSubject =
    new BehaviorSubject<InspectionType | null>(null);

  /**
   * Signal storing the currently chosen inspection type.
   */
  readonly currentInspectionType = toSignal(this.currentInspectionTypeSubject, {
    requireSync: true,
  });

  private readonly currentInspectionClient$ =
    this.currentInspectionTypeSubject.pipe(
      takeUntilDestroyed(this.destroyRef),
      filter((type): type is InspectionType => type !== null),
      switchMap((inspectionType) =>
        this.apiClient.createInspection(inspectionType.id),
      ),
      shareReplay(1),
    );

  /**
   * Signal containing the inspection features of the active inspection type.
   */
  readonly currentInspectionFeatures = toSignal(
    this.currentInspectionClient$.pipe(switchMap((client) => client.features)),
    { initialValue: [] },
  );

  /**
   * Signal holding the view model for the parameter input step.
   */
  readonly parameterViewModel = signal<ParameterStepViewModel | null>(null);

  private loopAbortController: AbortController | null = null;

  constructor() {
    if (this.dialogData?.initialInspectionTypeId) {
      this.initializeFromDialogData(this.dialogData);
    }
  }

  private initializeFromDialogData(dialogData: NewInspectionDialogData): void {
    toObservable(this.backendSync.inspectionTypes.value)
      .pipe(
        filter(
          (response): response is GetInspectionTypesResponse =>
            !!response && response.types.length > 0,
        ),
        map((r) =>
          r.types.find((t) => t.id === dialogData.initialInspectionTypeId),
        ),
        filter((matched): matched is InspectionType => !!matched),
        take(1),
        tap((matched) => {
          this.currentInspectionTypeSubject.next(matched);
          if (dialogData.initialParameters) {
            this.parameterStore.setDefaultValues(dialogData.initialParameters);
          }
        }),
        switchMap(() => this.currentInspectionClient$.pipe(take(1))),
        tap((client) => {
          if (dialogData.initialFeatureIds?.length) {
            const featureMap = Object.fromEntries(
              dialogData.initialFeatureIds.map((f) => [f, true]),
            );
            client.setFeatures(featureMap);
          }
        }),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe(() => {
        this.selectedStepChange(NewInspectionStepIndex.ParameterInput);
      });
  }

  /**
   * Selects an inspection type and moves to feature selection step.
   *
   * @param inspectionType The chosen inspection type.
   */
  public setInspectionType(inspectionType: InspectionType): void {
    this.currentInspectionTypeSubject.next(inspectionType);
    this.selectedStepChange(NewInspectionStepIndex.FeatureSelection);
  }

  /**
   * Handles step transitions in the wizard stepper.
   *
   * @param stepIndex The new step index to navigate to.
   */
  public selectedStepChange(stepIndex: NewInspectionStepIndex): void {
    if (
      this.selectedStepIndex() === stepIndex &&
      stepIndex === NewInspectionStepIndex.ParameterInput &&
      this.loopAbortController !== null
    ) {
      return;
    }
    this.selectedStepIndex.set(stepIndex);
    if (stepIndex === NewInspectionStepIndex.ParameterInput) {
      this.parameterViewModel.set(null);
      void this.startDryrunLoop();
    } else {
      this.stopDryrunLoop();
    }
  }

  /**
   * Toggles the enabled state of a feature.
   *
   * @param featureId The unique identifier of the feature.
   */
  public toggleFeature(featureId: string): void {
    const currentFeature = this.currentInspectionFeatures().find(
      (f) => f.id === featureId,
    );
    this.currentInspectionClient$
      .pipe(take(1), takeUntilDestroyed(this.destroyRef))
      .subscribe((client) => {
        client.setFeatures({ [featureId]: !currentFeature?.enabled });
      });
  }

  /**
   * Initiates the inspection task and closes the dialog upon success.
   */
  public onRunButtonClick(): void {
    this.currentInspectionClient$
      .pipe(
        take(1),
        switchMap((client) =>
          client.run(this.parameterStore.currentParameters()),
        ),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe(() => {
        this.extension.notifyLifecycleOnInspectionStart();
        this.dialogRef.close({ inspectionTaskStarted: true });
      });
  }

  private async startDryrunLoop(): Promise<void> {
    this.stopDryrunLoop();
    const abortController = new AbortController();
    this.loopAbortController = abortController;
    const signal = abortController.signal;

    let client: InspectionClient;
    try {
      client = await firstValueFrom(
        this.currentInspectionClient$.pipe(
          takeUntil(fromEvent(signal, 'abort')),
        ),
      );
      if (signal.aborted) {
        return;
      }
    } catch (err) {
      if (signal.aborted) {
        return;
      }
      throw new Error('Critical failure: Failed to fetch inspection client', {
        cause: err,
      });
    }

    while (!signal.aborted) {
      try {
        await this.runSingleDryrunIteration(client, signal);
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

  private async runSingleDryrunIteration(
    client: InspectionClient,
    signal: AbortSignal,
  ): Promise<void> {
    const sentParams = this.parameterStore.currentParameters();
    const res = await firstValueFrom(
      client
        .dryrunDirect(sentParams)
        .pipe(takeUntil(fromEvent(signal, 'abort'))),
    );
    if (signal.aborted) {
      return;
    }

    const currentParams = this.parameterStore.currentParameters();
    const changedDuringFlight = !haveEqualKeyValues(sentParams, currentParams);
    if (!changedDuringFlight) {
      this.parameterStore.setValidatedParameters(sentParams);
      this.parameterViewModel.set(buildParameterStepViewModel(res.metadata));
      this.parameterStore.setDefaultValues(
        flattenDefaultValues(res.metadata.form),
      );

      const paramsAfterDefaults = this.parameterStore.currentParameters();
      if (haveEqualKeyValues(sentParams, paramsAfterDefaults)) {
        await this.delay(800, signal);
      }
    }
  }

  private stopDryrunLoop(): void {
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
      const onAbort = () => {
        clearTimeout(timer);
        reject(new CancellationError('Delay aborted'));
      };
      const timer = setTimeout(() => {
        signal.removeEventListener('abort', onAbort);
        resolve();
      }, ms);
      signal.addEventListener('abort', onAbort, { once: true });
    });
  }

  ngOnDestroy(): void {
    this.stopDryrunLoop();
    if (this.parameterStore instanceof DefaultParameterStore) {
      this.parameterStore.destroy();
    }
  }
}
