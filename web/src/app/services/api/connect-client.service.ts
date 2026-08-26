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

import { Injectable } from '@angular/core';
import { createClient, Client } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { CELValidationService } from 'src/app/generated/api/v1/cel_validation_pb';
import { FileParameterUploadService } from 'src/app/generated/api/v1/file_parameter_upload_pb';
import { ImportInspectionService } from 'src/app/generated/api/v1/import_inspection_pb';
import { PopupService } from 'src/app/generated/api/v1/popup_pb';
import { ServerStatusService } from 'src/app/generated/api/v1/server_status_pb';
import { WorkbenchService } from 'src/app/generated/api/v1/workbench_pb';
import { InspectionService } from 'src/app/generated/api/v1/inspection_pb';
import { ApiPathUtil } from 'src/app/services/api/api-path-util';
import { createLegacyPollingInterceptor } from 'src/app/services/api/legacy-polling.interceptor';
import { environment } from 'src/environments/environment';

/**
 * ConnectClientService manages Connect-RPC clients and transports.
 */
@Injectable({
  providedIn: 'root',
})
export class ConnectClientService {
  private readonly transport = createConnectTransport({
    baseUrl: ApiPathUtil.getServerBaseUrl(),
    useBinaryFormat: environment.production,
    interceptors: [createLegacyPollingInterceptor()],
  });

  /**
   * WorkbenchService Connect-RPC client.
   */
  public readonly workbenchClient: Client<typeof WorkbenchService> =
    createClient(WorkbenchService, this.transport);

  /**
   * CELValidationService Connect-RPC client.
   */
  public readonly celValidationClient: Client<typeof CELValidationService> =
    createClient(CELValidationService, this.transport);

  /**
   * PopupService Connect-RPC client.
   */
  public readonly popupClient: Client<typeof PopupService> = createClient(
    PopupService,
    this.transport,
  );

  /**
   * ServerStatusService Connect-RPC client.
   */
  public readonly serverStatusClient: Client<typeof ServerStatusService> =
    createClient(ServerStatusService, this.transport);
  /**
   * FileParameterUploadService Connect-RPC client.
   */
  public readonly fileParameterUploadClient: Client<
    typeof FileParameterUploadService
  > = createClient(FileParameterUploadService, this.transport);

  /**
   * ImportInspectionService Connect-RPC client.
   */
  public readonly importInspectionClient: Client<
    typeof ImportInspectionService
  > = createClient(ImportInspectionService, this.transport);

  /**
   * InspectionService Connect-RPC client.
   */
  public readonly inspectionClient: Client<typeof InspectionService> =
    createClient(InspectionService, this.transport);
}
