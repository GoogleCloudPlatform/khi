/**
 * Copyright 2025 Google LLC
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

import {
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  input,
  signal,
  viewChild,
} from '@angular/core';
import { DiagramViewportService } from './diagram-viewport.service';
import { toSignal } from '@angular/core/rxjs-interop';
import {
  DIAGRAM_ELEMENT_ROLE,
  DiagramElementRole,
  MAX_LOD,
} from './diagram-element.directive';
import { LOD } from './lod.service';

/**
 * MinimapComponent shows a small minimap for user to see where they are looking easier.
 */
@Component({
  selector: 'diagram-minimap',
  templateUrl: './minimap.component.html',
  styleUrls: ['./minimap.component.sass'],
  host: {
    '[style.width.px]': 'width()',
    '[style.height.px]': 'height()',
  },
  providers: [
    {
      provide: DIAGRAM_ELEMENT_ROLE,
      useValue: DiagramElementRole.MINIMAP,
    },
    {
      provide: MAX_LOD,
      useValue: LOD.CONTAINER_ONLY,
    },
  ],
})
export class MinimapComponent {
  diagramViewportService = inject(DiagramViewportService);

  maskContainer = viewChild<ElementRef<HTMLDivElement>>('maskContainer');

  private viewportState = toSignal(this.diagramViewportService.viewportState, {
    initialValue: {
      contentWidth: 1,
      contentHeight: 1,
      contentScale: 1,
      viewportLeft: 0,
      viewportTop: 0,
      viewportWidth: 1,
      viewportHeight: 1,
    },
  });

  private aspectRatio = computed(() => {
    const state = this.viewportState();
    return state.contentWidth / state.contentHeight;
  });

  minimapContentScaleTransform = computed(() => {
    const state = this.viewportState();
    return `scale(${Math.min(this.width() / state.contentWidth, this.height() / state.contentHeight) * state.contentScale})`;
  });

  /**
   * The width of this minimap.
   */
  width = input.required<number>();

  /**
   * The height of this minimap.
   * This height is calculated from the aspect ratio of the diagram and given width.
   */
  height = computed(() => this.width() / this.aspectRatio());

  maskGridTemplateColumns = signal('1fr 1fr 1fr');

  maskGridTemplateRows = signal('1fr 1fr 1fr');

  constructor() {
    effect(() => {
      const area = this.viewportState();
      const width = this.width();
      const height = this.height();
      this.maskGridTemplateColumns.set(
        `${area.viewportLeft * width}px ${area.viewportWidth * width}px 1fr`,
      );
      this.maskGridTemplateRows.set(
        `${area.viewportTop * height}px ${area.viewportHeight * height}px 1fr`,
      );
    });
  }
}
