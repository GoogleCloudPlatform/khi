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

import { AfterViewInit, Component, ElementRef, ViewChild } from '@angular/core';
import { GraphRenderer } from './graph/renderer';
import { emptyGraphData } from '../../../common/schema/graph-schema';
import { DownloadService } from '../services/donwload-service';
import { GraphPageDataSource } from 'src/app/services/frame-connection/frames/graph-page-datasource.service';
@Component({
  selector: 'graph-architecture-graph',
  templateUrl: './architecture-graph.component.html',
  styleUrls: ['./architecture-graph.component.sass'],
})
export class ArchitectureGraphComponent implements AfterViewInit {
  constructor(
    private dataStore: GraphPageDataSource,
    private downloadService: DownloadService,
  ) {}

  @ViewChild('graphContainer')
  graphContainer!: ElementRef<HTMLDivElement>;

  graphRenderer!: GraphRenderer;

  ngAfterViewInit(): void {
    this.graphRenderer = new GraphRenderer(this.graphContainer.nativeElement);
    this.graphRenderer.updateGraphData(emptyGraphData());
    this.dataStore.data$.subscribe((d) => {
      this.graphRenderer.updateGraphData(d);
    });
    this.downloadService.registerRenderer(this.graphRenderer);
  }
}
