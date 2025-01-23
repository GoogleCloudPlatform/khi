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

import { Component } from '@angular/core';
import { DownloadService } from '../pages/graph/services/donwload-service';

@Component({
  selector: 'khi-graph-menu',
  templateUrl: './graph-menu.component.html',
  styleUrls: ['./graph-menu.component.sass'],
})
export class GraphMenuComponent {
  constructor(private downloadService: DownloadService) {}

  downloadAsPng() {
    this.downloadService.downloadAsPng();
  }

  downloadAsSvg() {
    this.downloadService.downloadAsSvg();
  }
}
