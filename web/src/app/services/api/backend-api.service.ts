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

import { Injectable, inject } from '@angular/core';
import {
  GetInspectionTypesResponse,
  GetInspectionFeatureResponse,
  InspectionFeature,
  InspectionDryRunResponse,
  GetInspectionResponse,
  InspectionDryRunRequest,
  InspectionRunRequest,
  InspectionMetadataOfRunResult,
  InspectionPatchRequest,
} from '../../common/schema/api-types';
import {
  EMPTY,
  Observable,
  ReplaySubject,
  Subject,
  catchError,
  concat,
  debounceTime,
  exhaustMap,
  from,
  map,
  mergeMap,
  of,
  range,
  reduce,
  shareReplay,
  switchMap,
  withLatestFrom,
} from 'rxjs';
import { ViewStateService } from '../view-state.service';
import { BackendAPI, DownloadProgressReporter } from './backend-api-interface';
import { ProgressDialogStatusUpdator } from '../progress/progress-interface';
import { ProgressUtil } from '../progress/progress-util';
import { ApiPathUtil } from 'src/app/services/api/api-path-util';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { resolveDownloadConfig } from 'src/app/services/api/download-config-resolver';
import {
  convertMapToParameterValues,
  convertProtoDryRunResponseToFrontend,
  convertProtoListItemToInspectionMetadata,
  convertProtoMetadataToInspectionMetadataOfRunResult,
} from 'src/app/services/api/inspection-converter';

/**
 * An implementation of BackendAPI interface.
 * All of the actual request calls against the backend must be through this class.
 */
@Injectable({
  providedIn: 'root',
})
export class BackendAPIImpl implements BackendAPI {
  private readonly viewState = inject(ViewStateService);
  private readonly connectClient = inject(ConnectClientService);

  /**
   * Get the server base path configuration path which is a configuration given as meta tag from backend.
   */
  public static getServerBasePath(): string {
    return ApiPathUtil.getServerBasePath();
  }

  public getInspectionTypes(): Observable<GetInspectionTypesResponse> {
    return from(
      this.connectClient.inspectionClient.getInspectionTypes({}),
    ).pipe(
      map((res) => ({
        types: res.types.map((t) => ({
          id: t.id,
          name: t.name,
          description: t.description,
          icon: t.icon,
        })),
      })),
    );
  }

  public getInspections(): Observable<GetInspectionResponse> {
    return from(this.connectClient.inspectionClient.getInspections({})).pipe(
      map((res) => {
        const inspections: {
          [key: string]: ReturnType<
            typeof convertProtoListItemToInspectionMetadata
          >;
        } = {};
        for (const item of res.inspections) {
          inspections[item.id] = convertProtoListItemToInspectionMetadata(item);
        }
        return {
          inspections,
          serverStat: { currentMemoryUsage: 0, totalMemory: 0 },
        };
      }),
    );
  }

  public createInspection(
    inspectionTypeId: string,
  ): Observable<InspectionClient> {
    return from(
      this.connectClient.inspectionClient.createInspection({
        inspectionTypeId,
      }),
    ).pipe(
      map(
        (res) => new InspectionClient(this, res.inspectionId, this.viewState),
      ),
    );
  }

  patchInspection(
    inspectionID: string,
    request: InspectionPatchRequest,
  ): Observable<void> {
    return from(
      this.connectClient.inspectionClient.updateInspection({
        inspectionId: inspectionID,
        name: request.name,
      }),
    ).pipe(map(() => void 0));
  }

  public getFeatureList(
    inspectionID: string,
  ): Observable<GetInspectionFeatureResponse> {
    return from(
      this.connectClient.inspectionClient.getInspectionFeatures({
        inspectionId: inspectionID,
      }),
    ).pipe(
      map((res) => ({
        features: res.features.map((f) => ({
          id: f.id,
          label: f.label,
          description: f.description,
          enabled: f.enabled,
        })),
      })),
    );
  }

  public setEnabledFeatures(
    inspectionID: string,
    featureMap: { [key: string]: boolean },
  ): Observable<void> {
    return from(
      this.connectClient.inspectionClient.updateInspectionFeatures({
        inspectionId: inspectionID,
        featureStates: featureMap,
      }),
    ).pipe(map(() => void 0));
  }

  public getInspectionMetadata(
    inspectionID: string,
  ): Observable<InspectionMetadataOfRunResult> {
    return from(
      this.connectClient.inspectionClient.getInspectionMetadata({
        inspectionId: inspectionID,
      }),
    ).pipe(
      map((res) => convertProtoMetadataToInspectionMetadataOfRunResult(res)),
    );
  }

  public runInspection(
    inspectionID: string,
    request: InspectionRunRequest,
  ): Observable<void> {
    const params = convertMapToParameterValues(
      request as Record<string, unknown>,
    );
    const tzShift =
      typeof request['timezoneShift'] === 'number'
        ? (request['timezoneShift'] as number)
        : 0;
    return from(
      this.connectClient.inspectionClient.runInspection({
        inspectionId: inspectionID,
        parameters: {
          parameters: params,
          timezoneShiftHours: tzShift,
        },
      }),
    ).pipe(map(() => void 0));
  }

  public dryRunInspection(
    inspectionID: string,
    request: InspectionDryRunRequest,
  ): Observable<InspectionDryRunResponse> {
    const params = convertMapToParameterValues(
      request as Record<string, unknown>,
    );
    const tzShift =
      typeof request['timezoneShift'] === 'number'
        ? (request['timezoneShift'] as number)
        : 0;
    return from(
      this.connectClient.inspectionClient.dryRunInspection({
        inspectionId: inspectionID,
        parameters: {
          parameters: params,
          timezoneShiftHours: tzShift,
        },
      }),
    ).pipe(map((res) => convertProtoDryRunResponseToFrontend(res)));
  }

  public getInspectionData(
    inspectionID: string,
    reporter: DownloadProgressReporter,
  ): Observable<{ fileName: string; content: Blob }> {
    const downloadConfig = resolveDownloadConfig();
    return from(
      this.connectClient.inspectionClient.getInspectionMetadata({
        inspectionId: inspectionID,
      }),
    ).pipe(
      switchMap((metadata) => {
        const totalSize = Number(metadata.header?.fileSize ?? 0n);
        const fileName = metadata.header?.suggestedFilename || 'inspection.khi';
        const chunks = Math.max(
          1,
          Math.ceil(totalSize / downloadConfig.chunkSize),
        );
        const chunkLoaded: number[] = new Array(chunks).fill(0);
        return range(0, chunks).pipe(
          map((index) => {
            const startInBytes = index * downloadConfig.chunkSize;
            const maxSizeInBytes = Math.min(
              downloadConfig.chunkSize,
              totalSize - startInBytes,
            );
            return { index, startInBytes, maxSizeInBytes };
          }),
          mergeMap(({ index, startInBytes, maxSizeInBytes }) => {
            return from(
              this.connectClient.inspectionClient.getInspectionDataChunk({
                inspectionId: inspectionID,
                offsetBytes: BigInt(startInBytes),
                maxSizeBytes: BigInt(maxSizeInBytes),
              }),
            ).pipe(
              map((chunk) => {
                const blob = new Blob([chunk.data as BlobPart]);
                chunkLoaded[index] = blob.size;
                const doneBytes = chunkLoaded.reduce((a, b) => a + b, 0);
                reporter(totalSize, doneBytes);
                return { index, blob };
              }),
            );
          }, downloadConfig.maxConcurrency),
          reduce(
            (acc: Blob[], downloadResult: { index: number; blob: Blob }) => {
              acc[downloadResult.index] = downloadResult.blob;
              return acc;
            },
            [],
          ),
          map((blobs) => {
            const content = new Blob(blobs);
            if (content.size !== totalSize) {
              throw new Error(
                `Downloaded size: ${content.size} != Content-Length: ${totalSize}`,
              );
            }
            return { fileName, content };
          }),
        );
      }),
    );
  }

  public cancelInspection(inspectionID: string): Observable<void> {
    return from(
      this.connectClient.inspectionClient.cancelInspection({
        inspectionId: inspectionID,
      }),
    ).pipe(map(() => void 0));
  }
}

export class InspectionClient {
  private static DRYRUN_DEBOUNCE_DURATION = 100;

  public features = new ReplaySubject<InspectionFeature[]>(1);

  private dryRunParameter = new Subject<InspectionDryRunRequest>();

  private nonFormParameters = concat(this.viewState.timezoneShift).pipe(
    map((tzShift) => ({
      timezoneShift: tzShift,
    })),
    shareReplay(1),
  );

  public dryRunResult = this.dryRunParameter.pipe(
    debounceTime(InspectionClient.DRYRUN_DEBOUNCE_DURATION),
    exhaustMap((param) =>
      this.dryrunDirect(param).pipe(catchError(() => EMPTY)),
    ), // This must be exhaustMap not to cancel a request sent before in slow network environment with switchMap.
    shareReplay(1),
  );

  constructor(
    private readonly api: BackendAPI,
    public readonly inspectionID: string,
    private readonly viewState: ViewStateService,
  ) {
    this.downloadFeatureList();
  }

  public patchInspection(request: InspectionPatchRequest) {
    return this.api.patchInspection(this.inspectionID, request);
  }

  public downloadFeatureList() {
    return this.api
      .getFeatureList(this.inspectionID)
      .pipe(map((r) => r.features))
      .subscribe((features) => this.features.next(features));
  }

  public setFeatures(featuresMap: { [key: string]: boolean }) {
    return this.api
      .setEnabledFeatures(this.inspectionID, featuresMap)
      .subscribe(() => {
        this.downloadFeatureList();
      });
  }

  public run(request: InspectionRunRequest) {
    return this.getRunParameter(request).pipe(
      switchMap((request) => {
        return this.api.runInspection(this.inspectionID, request);
      }),
      map(() => {}),
    );
  }

  public dryrun(request: InspectionDryRunRequest) {
    this.dryRunParameter.next(request);
  }

  /**
   * dryrunDirect calls the dryrun API directly without debouncing.
   * This method is public for testing purpose. Use dryrun method instead.
   */
  public dryrunDirect(request: InspectionDryRunRequest) {
    return this.getRunParameter(request).pipe(
      switchMap((request) =>
        this.api.dryRunInspection(this.inspectionID, request),
      ),
    );
  }

  private getRunParameter(
    request: InspectionRunRequest | InspectionDryRunRequest,
  ): Observable<{ [key: string]: unknown }> {
    return of(request).pipe(
      withLatestFrom(this.nonFormParameters),
      map(([request, nonForm]) => ({
        ...request,
        ...nonForm,
      })),
    );
  }
}

/**
 * Utility functions using BackendAPI interface
 */
export class BackendAPIUtil {
  /**
   * Save the inspection data as a file
   */
  public static downloadInspectionDataAsFile(
    api: BackendAPI,
    inspectionID: string,
    progress: ProgressDialogStatusUpdator,
  ) {
    progress.show();
    progress.updateProgress({
      message: 'Downloading inspection data...',
      percent: 0,
      mode: 'determinate',
    });
    return api
      .getInspectionData(inspectionID, (fileSize, done) => {
        progress.updateProgress({
          message: `Downloading inspection data (${ProgressUtil.formatPogressMessageByBytes(done, fileSize)})`,
          percent: (done / fileSize) * 100,
          mode: 'determinate',
        });
      })
      .pipe(
        map(({ fileName, content }) => {
          const link = document.createElement('a');
          link.download = fileName;
          link.href = window.URL.createObjectURL(content);
          link.style.display = 'none';
          document.body.appendChild(link);
          link.click();
          link.remove();
          progress.dismiss();
          return fileName;
        }),
      );
  }
}
