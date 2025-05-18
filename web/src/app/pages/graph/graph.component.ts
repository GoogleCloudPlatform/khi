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

import { Component, input } from '@angular/core';
import { GraphMenuComponent } from 'src/app/header/graph-menu.component';
import { TitleBarComponent } from 'src/app/header/titlebar.component';
import { DiagramViewportComponent } from '../../services/diagram/diagram-viewport.component';
import { DiagramViewportService } from 'src/app/services/diagram/diagram-viewport.service';
import { MinimapComponent } from '../../services/diagram/minimap.component';
import {
  DIAGRAM_ELEMENT_ROLE,
  DiagramElementRole,
  MAX_LOD,
} from 'src/app/services/diagram/diagram-element.directive';
import { CommonModule } from '@angular/common';
import { LOD } from 'src/app/services/diagram/lod.service';
import { DiagramSVGViewportOverlayComponent } from '../../services/diagram/diagram-svg-viewport-overlay.component';
import { WaypointManagerService } from 'src/app/services/diagram/waypoint-manager.service';
import { DiagramK8sRootComponent } from '../../services/diagram/diagram-root/diagram-k8s-root.component';
import { DiagramK8sSVGRootComponent } from 'src/app/services/diagram/diagram-root/diagram-k8s-svg-root.component';
import { DiagramModel } from 'src/app/services/diagram/diagram-model-types';
import { SAMPLE_DIAGRAM } from 'src/app/services/diagram/sample-diagram-models';

@Component({
  selector: 'graph-root',
  templateUrl: './graph.component.html',
  styleUrls: ['./graph.component.sass'],
  imports: [
    CommonModule,
    TitleBarComponent,
    GraphMenuComponent,
    DiagramViewportComponent,
    MinimapComponent,
    DiagramSVGViewportOverlayComponent,
    DiagramK8sRootComponent,
    DiagramK8sSVGRootComponent,
  ],
  providers: [
    DiagramViewportService,
    {
      provide: DIAGRAM_ELEMENT_ROLE,
      useValue: DiagramElementRole.Content,
    },
    {
      provide: MAX_LOD,
      useValue: LOD.UNLIMITED,
    },
    {
      provide: WaypointManagerService,
      useClass: WaypointManagerService,
    },
  ],
})
export class GraphComponent {
  model = input<DiagramModel>(SAMPLE_DIAGRAM);
}
