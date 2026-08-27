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
import { GraphRenderer } from '../architecture-graph/graph/renderer';

/**
 * Provides download capabilities for the architecture graph view.
 */
@Injectable({ providedIn: 'root' })
export class GraphDownloadService {
  private renderer: GraphRenderer | null = null;

  /**
   * Registers the active GraphRenderer instance.
   *
   * @param renderer - The active renderer instance.
   */
  public registerRenderer(renderer: GraphRenderer): void {
    this.renderer = renderer;
  }

  /**
   * Downloads the graph as an SVG image.
   *
   * @param filename - Optional destination filename.
   */
  public downloadSvg(filename?: string): void {
    this.renderer?.downloadSvg(filename);
  }

  /**
   * Downloads the graph as a PNG image.
   *
   * @param filename - Optional destination filename.
   */
  public downloadPng(filename?: string): void {
    void this.renderer?.downloadPng(filename);
  }
}
