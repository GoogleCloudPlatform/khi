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

import { Injectable } from '@angular/core';
import {
  GetInspectionTypesResponse,
  CreateInspectionResponse,
  GetInspectionFeatureResponse,
  PatchInspectionFeatureRequest,
  InspectionFeature,
  InspectionDryRunResponse,
  GetInspectionResponse,
  InspectionDryRunRequest,
  InspectionRunRequest,
  PopupAnswerResponse,
  PopupAnswerValidationResult,
  PopupFormRequest,
  InspectionMetadataOfRunResult,
  GetConfigResponse,
} from '../../common/schema/api-types';
import { HttpClient, HttpEvent } from '@angular/common/http';
import {
  Observable,
  ReplaySubject,
  Subject,
  concat,
  debounceTime,
  map,
  mergeMap,
  of,
  range,
  reduce,
  retry,
  shareReplay,
  switchMap,
  withLatestFrom,
} from 'rxjs';
import { ViewStateService } from '../view-state.service';
import { BackendAPI, DownloadProgressReporter } from './backend-api-interface';
import { ProgressDialogStatusUpdator } from '../progress/progress-interface';
import { ProgressUtil } from '../progress/progress-util';
import { UploadToken } from 'src/app/common/schema/form-types';

/**
 * An implementation of BackendAPI interface.
 * All of the actual request calls against the backend must be through this class.
 */
@Injectable({
  providedIn: 'root',
})
export class BackendAPIImpl implements BackendAPI {
  private readonly API_BASE_PATH = '/api/v3';

  private readonly MAX_INSPECTION_DATA_DOWNLOAD_CHUNK_SIZE = 16 * 1024 * 1024;
  private readonly INSPECTION_DATA_DOWNLOAD_CONCURRENCY = 10;

  /**
   * The base address of the backend server.
   *
   * The index HTML file contains `<base>` tag to control the base address of resources in frontend to supporting KHI to be hosted with path rewriting behind reverse proxies.
   */
  private readonly baseUrl: string;

  private readonly getConfigObservable: Observable<GetConfigResponse>;

  constructor(
    private http: HttpClient,
    private readonly viewState: ViewStateService,
  ) {
    this.baseUrl = BackendAPIImpl.getServerBasePath() + this.API_BASE_PATH;

    const getConfigUrl = this.baseUrl + '/config';
    this.getConfigObservable = this.http
      .get<GetConfigResponse>(getConfigUrl)
      .pipe(
        retry({ delay: 1000 }),
        shareReplay(1), // the config is cached at the first time of the loading.
      );
  }

  /**
   * Get the server base path configuration path which is a configuration given as meta tag from backend.
   */
  public static getServerBasePath(): string {
    const basePathTag = document.getElementById('server-base-path');
    if (basePathTag === null) return '';
    let content = basePathTag.getAttribute('content');
    if (content?.endsWith('/')) {
      content = content.substring(0, content.length - 1);
    }
    return content ?? '';
  }

  /**
   * Get configuration of this frontend from the server.
   */
  public getConfig(): Observable<GetConfigResponse> {
    return this.getConfigObservable;
  }

  public getInspectionTypes() {
    const url = this.baseUrl + '/inspection/types';
    return this.http.get<GetInspectionTypesResponse>(url);
  }

  public getInspections() {
    const url = this.baseUrl + '/inspection';
    return this.http.get<GetInspectionResponse>(url);
  }

  public createInspection(
    inspectionTypeId: string,
  ): Observable<InspectionClient> {
    const url = this.baseUrl + '/inspection/types/' + inspectionTypeId;
    return this.http
      .post<CreateInspectionResponse>(url, null)
      .pipe(
        map(
          (response) =>
            new InspectionClient(this, response.inspectionID, this.viewState),
        ),
      );
  }

  public getFeatureList(inspectionID: string) {
    const url = this.baseUrl + `/inspection/${inspectionID}/features`;
    return this.http.get<GetInspectionFeatureResponse>(url);
  }

  public setEnabledFeatures(
    inspectionID: string,
    featureMap: { [key: string]: boolean },
  ) {
    const url = this.baseUrl + `/inspection/${inspectionID}/features`;
    const request: PatchInspectionFeatureRequest = {
      features: featureMap,
    };
    return this.http.patch(url, request, {
      responseType: 'text',
    }) as Observable<unknown> as Observable<void>;
  }

  public getInspectionMetadata(inspectionID: string) {
    const url = this.baseUrl + `/inspection/${inspectionID}/metadata`;
    return this.http.get<InspectionMetadataOfRunResult>(url);
  }

  public runInspection(
    inspectionID: string,
    request: InspectionRunRequest,
  ): Observable<void> {
    const url = this.baseUrl + `/inspection/${inspectionID}/run`;
    return this.http
      .post(url, request, { responseType: 'text' })
      .pipe(map(() => void 0));
  }

  public dryRunInspection(
    inspectionID: string,
    request: InspectionDryRunRequest,
  ): Observable<InspectionDryRunResponse> {
    const url = this.baseUrl + `/inspection/${inspectionID}/dryrun`;
    return this.http.post<InspectionDryRunResponse>(url, request);
  }

  public getInspectionData(
    inspectionID: string,
    reporter: DownloadProgressReporter,
  ) {
    // accumulator holds donwnloaded bytes for reporter
    let done = 0;
    return this.getInspectionMetadata(inspectionID).pipe(
      switchMap((metadata) => {
        const totalSize = metadata.header.fileSize ?? 0;
        const chunks = Math.ceil(
          totalSize / this.MAX_INSPECTION_DATA_DOWNLOAD_CHUNK_SIZE,
        );
        return range(0, chunks).pipe(
          map((index) => {
            const startInBytes =
              index * this.MAX_INSPECTION_DATA_DOWNLOAD_CHUNK_SIZE;
            const maxSizeInBytes = Math.min(
              this.MAX_INSPECTION_DATA_DOWNLOAD_CHUNK_SIZE,
              totalSize - startInBytes,
            );
            const params = `start=${startInBytes}&maxSize=${maxSizeInBytes}`;
            return { index, params };
          }),
          mergeMap(({ index, params }) => {
            const url = this.baseUrl + `/inspection/${inspectionID}/data`;
            return this.http
              .get(`${url}?${params}`, { responseType: 'blob' })
              .pipe(
                map((blob) => {
                  done += blob.size;
                  reporter(totalSize, done);
                  return { index, blob };
                }),
              );
          }, this.INSPECTION_DATA_DOWNLOAD_CONCURRENCY),
          reduce(
            (acc: Blob[], downloadResult: { index: number; blob: Blob }) => {
              acc[downloadResult.index] = downloadResult.blob;
              return acc;
            },
            [],
          ),
          map((blobs) => {
            const fileName = metadata.header.suggestedFilename;
            const content = new Blob(blobs);
            if (content.size != totalSize) {
              // The downloaded file is very likely broken if the inspection API works well.
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

  public getPopup(): Observable<PopupFormRequest | null> {
    const url = this.baseUrl + `/popup`;
    return this.http.get<PopupFormRequest | null>(url);
  }

  public validatePopupAnswer(
    answer: PopupAnswerResponse,
  ): Observable<PopupAnswerValidationResult> {
    const url = this.baseUrl + `/popup/validate`;
    return this.http.post<PopupAnswerValidationResult>(url, answer);
  }
  public answerPopup(answer: PopupAnswerResponse): Observable<void> {
    const url = this.baseUrl + `/popup/answer`;
    return this.http.post(url, answer).pipe(map(() => {}));
  }

  public cancelInspection(inspectionID: string) {
    const url = this.baseUrl + `/inspection/${inspectionID}/cancel`;
    return this.http
      .post(url, null, { responseType: 'text' })
      .pipe(map(() => {}));
  }

  public uploadFile(
    token: UploadToken,
    file: File,
  ): Observable<HttpEvent<unknown>> {
    const url = this.baseUrl + `/upload`;
    const formData = new FormData();
    formData.append('upload-token-id', token.id);
    formData.append('file', file, file.name);
    return this.http.post(url, formData, {
      reportProgress: true,
      observe: 'events',
    });
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
    switchMap((param) => this.dryrunDirect(param)),
    shareReplay(1),
  );

  constructor(
    private readonly api: BackendAPI,
    public readonly inspectionID: string,
    private readonly viewState: ViewStateService,
  ) {
    this.downloadFeatureList();
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
