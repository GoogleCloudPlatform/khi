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

import { Component, inject, OnInit } from '@angular/core';
import { ArchitectureGraphComponent } from './architecture-graph/architecture-graph.component';
import { HeaderV2SmartComponent } from 'src/app/headerv2/header-v2-smart.component';
import {
  MenuManager,
  MenuItemType,
} from 'src/app/services/menu/menu-manager.service';
import { DownloadService } from 'src/app/pages/graph/services/donwload-service';

/**
 * GraphComponent renders the architecture graph view.
 * It uses MenuManager to register page-specific menus.
 */
@Component({
  selector: 'khi-graph-root',
  templateUrl: './graph.component.html',
  styleUrls: ['./graph.component.scss'],
  imports: [HeaderV2SmartComponent, ArchitectureGraphComponent],
  providers: [MenuManager],
})
export class GraphComponent implements OnInit {
  private readonly menuManager = inject(MenuManager);
  private readonly downloadService = inject(DownloadService);

  /**
   * Initializes the component and registers the download menu.
   */
  ngOnInit() {
    this.menuManager.addGroup('graph-download', 'Download', 10, 'download');
    this.menuManager.addItem('graph-download', {
      id: 'download-png',
      label: 'Download as PNG',
      type: MenuItemType.Button,
      icon: 'image',
      action: () => this.downloadService.downloadAsPng(),
      priority: 1,
    });
    this.menuManager.addItem('graph-download', {
      id: 'download-svg',
      label: 'Download as SVG',
      type: MenuItemType.Button,
      icon: 'code',
      action: () => this.downloadService.downloadAsSvg(),
      priority: 2,
    });
  }
}
