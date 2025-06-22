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

import { Component, inject } from '@angular/core';
import { DownloadService } from '../pages/graph/services/donwload-service';
import { MatMenuModule } from '@angular/material/menu';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'khi-graph-menu',
  templateUrl: './graph-menu.component.html',
  styleUrls: ['./graph-menu.component.scss'],
  imports: [MatMenuModule, MatIconModule, MatButtonModule],
})
export class GraphMenuComponent {
  private readonly downloadService = inject(DownloadService);

  downloadAsPng() {
    this.downloadService.downloadAsPng();
  }

  downloadAsSvg() {
    this.downloadService.downloadAsSvg();
  }
}
