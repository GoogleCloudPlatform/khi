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

import { Component, inject, Input } from '@angular/core';
import { WindowConnectorService } from '../services/frame-connection/window-connector.service';
import { map } from 'rxjs';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { RainbowPipe } from '../common/rainbow.pipe';
import { VERSION } from 'src/environments/version';
import { MatButtonModule } from '@angular/material/button';
import { BACKEND_API } from '../services/api/backend-api-interface';

@Component({
  selector: 'khi-title',
  templateUrl: './titlebar.component.html',
  styleUrls: ['./titlebar.component.sass'],
  imports: [
    CommonModule,
    MatIconModule,
    MatButtonModule,
    MatMenuModule,
    RainbowPipe,
  ],
})
export class TitleBarComponent {
  @Input()
  pageName = 'N/A';

  backendAPI = inject(BACKEND_API);

  version = VERSION;

  isViewerMode = this.backendAPI
    .getConfig()
    .pipe(map((config) => config.viewerMode));

  mainPageConenctionEstablished =
    this.windowConnector.mainPageConenctionEstablished;

  sessionId = this.windowConnector.sessionEstablished.pipe(
    map(() => this.windowConnector.sessionId),
  );

  sessionPages = this.windowConnector.sessionPages;

  constructor(private readonly windowConnector: WindowConnectorService) {}

  focusWindow(frameId: string) {
    this.windowConnector.focusWindow(frameId);
  }
}
