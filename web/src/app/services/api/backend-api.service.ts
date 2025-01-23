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
  CreateInspectionTaskResponse,
  GetInspectionTaskFeatureResponse,
  PutInspectionTaskFeatureRequest,
  InspectionFeature,
  InspectionDryRunResponse,
  GetInspectionTasksResponse,
  InspectionDryRunRequest,
  InspectionRunRequest,
  PopupAnswerResponse,
  PopupAnswerValidationResult,
  PopupFormRequest,
  InspectionMetadataOfRunResult,
} from '../../common/schema/api-types';
import {
  HttpClient,
  HttpEventType,
  HttpRequest,
  HttpResponse,
} from '@angular/common/http';
import {
  Observable,
  ReplaySubject,
  Subject,
  concat,
  debounceTime,
  filter,
  forkJoin,
  last,
  map,
  mergeMap,
  of,
  shareReplay,
  switchMap,
  takeWhile,
  withLatestFrom,
} from 'rxjs';
import { ViewStateService } from '../view-state.service';
import { BackendAPI, DownloadProgressReporter } from './backend-api-interface';
import { ProgressDialogStatusUpdator } from '../progress/progress-interface';
import { ProgressUtil } from '../progress/progress-util';

/**
 * An implementation of BackendAPI interface.
 * All of the actual request calls against the backend must be through this class.
 */
@Injectable({
  providedIn: 'root',
})
export class BackendAPIImpl implements BackendAPI {
  private readonly MAX_INSPECTION_DATA_DOWNLOAD_CHUNK_SIZE = 16 * 1024 * 1024;

  /**
   * The base address of the backend server.
   *
   * The index HTML file contains `<base>` tag to control the base address of resources in frontend to supporting KHI to be hosted with path rewriting behind reverse proxies.
   * This backend address can't rely on this feature, because the backend can be placed on the other servers from this frontend and addresses in this class must be in the absolute format. (any path beginning with `/` or the address begining with `http`).
   * (The development server usually runs the backend with the port 8080, but runs the angular development server for frontend with the port 4200. The origin is different and frontend needs to access the backend.)
   *
   * KHI uses the environment variable `NG_APP_BACKEND_URL_PREFIX` at the build time, and another parameter given from the backend via the meta tag.
   * The format will be in `(The environment variable NG_APP_BACKEND_URL_PREFIX)(The prefix supplied from the backend)` and it must not have the trailing slash.
   */
  private readonly baseUrl: string;

  constructor(
    private http: HttpClient,
    private readonly viewState: ViewStateService,
  ) {
    const urlPrefix = process.env['NG_APP_BACKEND_URL_PREFIX'] ?? '';
    this.baseUrl = urlPrefix + BackendAPIImpl.getServerBasePath();
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

  public getInspectionTypes() {
    const url = this.baseUrl + '/api/v2/inspection/types';
    return this.http.get<GetInspectionTypesResponse>(url);
  }

  public getTaskStatuses() {
    const url = this.baseUrl + '/api/v2/inspection/tasks';
    return this.http.get<GetInspectionTasksResponse>(url);
  }

  public createInspection(
    inspectionTypeId: string,
  ): Observable<InspectionTaskClient> {
    const url = this.baseUrl + '/api/v2/inspection/types/' + inspectionTypeId;
    return this.http
      .post<CreateInspectionTaskResponse>(url, null)
      .pipe(
        map(
          (response) =>
            new InspectionTaskClient(
              this,
              response.inspectionId,
              this.viewState,
            ),
        ),
      );
  }

  public getFeatureList(taskId: string) {
    const url = this.baseUrl + `/api/v2/inspection/tasks/${taskId}/features`;
    return this.http.get<GetInspectionTaskFeatureResponse>(url);
  }

  public setEnabledFeatures(taskId: string, featureIds: string[]) {
    const url = this.baseUrl + `/api/v2/inspection/tasks/${taskId}/features`;
    const request: PutInspectionTaskFeatureRequest = {
      features: featureIds,
    };
    return this.http.put(url, request, {
      responseType: 'text',
    }) as Observable<unknown> as Observable<void>;
  }

  public getInspectionMetadata(taskId: string) {
    const url = this.baseUrl + `/api/v2/inspection/tasks/${taskId}/metadata`;
    return this.http.get<InspectionMetadataOfRunResult>(url);
  }

  public runTask(
    taskId: string,
    request: InspectionRunRequest,
  ): Observable<void> {
    const url = this.baseUrl + `/api/v2/inspection/tasks/${taskId}/run`;
    return this.http
      .post(url, request, { responseType: 'text' })
      .pipe(map(() => void 0));
  }

  public dryRunTask(
    taskId: string,
    request: InspectionDryRunRequest,
  ): Observable<InspectionDryRunResponse> {
    const url = this.baseUrl + `/api/v2/inspection/tasks/${taskId}/dryrun`;
    return this.http.post<InspectionDryRunResponse>(url, request);
  }

  public getInspectionData(taskId: string, reporter: DownloadProgressReporter) {
    const receivedBuffers = [] as ArrayBuffer[];
    let loadedBytes = 0;
    const responseSubject = new ReplaySubject<Blob | null>(1);
    const partialRequestsSubject = new Subject<HttpRequest<ArrayBuffer>>();
    partialRequestsSubject
      .pipe(
        mergeMap((request) =>
          this.http.request<ArrayBuffer>(request).pipe(
            filter((event) => event.type === HttpEventType.Response),
            map((response) => response as HttpResponse<ArrayBuffer>),
            last(),
          ),
        ),
        takeWhile((response) => {
          if (!response.body)
            throw new Error('unexpected response. body is null.');
          return response.body.byteLength > 0;
        }),
      )
      .subscribe({
        next: (chunk) => {
          receivedBuffers.push(chunk.body!);
          loadedBytes += chunk.body!.byteLength;
          // request the next chunk
          partialRequestsSubject.next(
            new HttpRequest<ArrayBuffer>(
              'GET',
              this.getRangedDataURL(
                taskId,
                loadedBytes,
                this.MAX_INSPECTION_DATA_DOWNLOAD_CHUNK_SIZE,
              ),
              null,
              { responseType: 'arraybuffer' },
            ),
          );
          reporter(loadedBytes);
        },
        complete: () => {
          responseSubject.next(
            new Blob(receivedBuffers, { type: 'application/octet-stream' }),
          );
          responseSubject.complete();
        },
      });

    // request the initial chunk
    partialRequestsSubject.next(
      new HttpRequest<ArrayBuffer>(
        'GET',
        this.getRangedDataURL(
          taskId,
          0,
          this.MAX_INSPECTION_DATA_DOWNLOAD_CHUNK_SIZE,
        ),
        null,
        { responseType: 'arraybuffer' },
      ),
    );

    return responseSubject;
  }

  private getRangedDataURL(
    taskId: string,
    startInBytes: number,
    maxSizeInBytes: number,
  ): string {
    const url = this.baseUrl + `/api/v2/inspection/tasks/${taskId}/data`;
    return url + `?start=${startInBytes}&maxSize=${maxSizeInBytes}`;
  }

  public getPopup(): Observable<PopupFormRequest | null> {
    const url = this.baseUrl + `/api/v2/popup`;
    return this.http.get<PopupFormRequest | null>(url);
  }

  public validatePopupAnswer(
    answer: PopupAnswerResponse,
  ): Observable<PopupAnswerValidationResult> {
    const url = this.baseUrl + `/api/v2/popup/validate`;
    return this.http.post<PopupAnswerValidationResult>(url, answer);
  }
  public answerPopup(answer: PopupAnswerResponse): Observable<void> {
    const url = this.baseUrl + `/api/v2/popup/answer`;
    return this.http.post(url, answer).pipe(map(() => {}));
  }

  public cancelInspection(taskId: string) {
    const url = this.baseUrl + `/api/v2/inspection/tasks/${taskId}/cancel`;
    return this.http
      .post(url, null, { responseType: 'text' })
      .pipe(map(() => {}));
  }
}

export class InspectionTaskClient {
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
    debounceTime(InspectionTaskClient.DRYRUN_DEBOUNCE_DURATION),
    switchMap((param) => this.dryrunDirect(param)),
    shareReplay(1),
  );

  constructor(
    private readonly api: BackendAPI,
    public readonly taskId: string,
    private readonly viewState: ViewStateService,
  ) {
    this.downloadFeatureList();
  }

  public downloadFeatureList() {
    return this.api
      .getFeatureList(this.taskId)
      .pipe(map((r) => r.features))
      .subscribe((features) => this.features.next(features));
  }

  public setFeatures(featureIds: string[]) {
    return this.api
      .setEnabledFeatures(this.taskId, featureIds)
      .subscribe(() => {
        this.downloadFeatureList();
      });
  }

  public run(request: InspectionRunRequest) {
    return this.getRunParameter(request).pipe(
      switchMap((request) => {
        return this.api.runTask(this.taskId, request);
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
      switchMap((request) => this.api.dryRunTask(this.taskId, request)),
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
    taskId: string,
    progress: ProgressDialogStatusUpdator,
  ) {
    progress.show();
    return api.getInspectionMetadata(taskId).pipe(
      switchMap((metadata) =>
        forkJoin([
          of(metadata),
          api.getInspectionData(taskId, (done) => {
            const fileSize = metadata.header.fileSize ?? 0;
            progress.updateProgress({
              message: `Downloading inspection data (${ProgressUtil.formatPogressMessageByBytes(done, fileSize)})`,
              percent: (done / fileSize) * 100,
              mode: 'determinate',
            });
          }),
        ]),
      ),
      map(([metadata, blob]) => {
        if (blob === null) return;
        const link = document.createElement('a');
        link.download = metadata.header.suggestedFilename;
        link.href = window.URL.createObjectURL(blob);
        link.style.display = 'none';
        document.body.appendChild(link);
        link.click();
        link.remove();
        progress.dismiss();
        return metadata.header.suggestedFilename;
      }),
    );
  }
}
