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

import { Component, Input, OnChanges } from '@angular/core';
import { ResizingCalculator } from '../resizable-pane/resizing-calculator';
import { Observable } from 'rxjs';

@Component({
  selector: 'khi-side-pane',
  templateUrl: './side-pane.component.html',
  styleUrls: ['./side-pane.component.sass'],
})
export class SidePaneComponent implements OnChanges {
  static readonly DEFAULT_PANE_WIDTH = 300;

  static readonly MINIMUM_PANE_WIDTH = 100;

  @Input()
  paneTitle = '';

  @Input()
  icon = '';

  @Input()
  resizeCalculator!: ResizingCalculator;

  @Input()
  areaNameInResizer: string = '';

  areaSize!: Observable<number>;

  ngOnChanges(): void {
    this.areaSize = this.resizeCalculator.areaSize(this.areaNameInResizer);
  }

  resizeStart() {
    const resizeMove = (e: MouseEvent) => {
      const size = this.resizeCalculator.getAreaSize(this.areaNameInResizer);
      this.resizeCalculator.resizeArea(
        this.areaNameInResizer,
        size - e.movementX,
      );
    };
    window.addEventListener('mouseup', () => {
      window.removeEventListener('mousemove', resizeMove);
    });
    window.addEventListener('mousemove', resizeMove);
  }
}
