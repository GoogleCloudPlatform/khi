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
import { WorkbenchService } from 'src/app/generated/api/v1/workbench_pb';
import { CELValidationService } from 'src/app/generated/api/v1/cel_validation_pb';
import { ApiPathUtil } from 'src/app/services/api/api-path-util';

/**
 * ConnectClientService manages Connect-RPC clients and transports.
 */
@Injectable({
  providedIn: 'root',
})
export class ConnectClientService {
  private readonly transport = createConnectTransport({
    baseUrl: ApiPathUtil.getServerBaseUrl(),
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
}
