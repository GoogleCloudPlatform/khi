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

import { Injectable } from '@angular/core';
import { GraphRenderer } from '../architecture-graph/graph/renderer';

@Injectable({ providedIn: 'root' })
export class DownloadService {
  private renderer: GraphRenderer | null = null;

  public registerRenderer(renderer: GraphRenderer) {
    this.renderer = renderer;
  }

  public downloadAsSvg() {
    const rawSVG = this.renderer?.getSVGForDownload();
    if (!rawSVG) return;

    const blob = new Blob([rawSVG.outerHTML], {
      type: 'image/svg+xml',
    });

    const blobUrl = window.URL.createObjectURL(blob);
    const downloadAnchor = document.createElement('a');
    downloadAnchor.download = 'diagram.svg';
    downloadAnchor.href = blobUrl;
    downloadAnchor.click();
  }

  public downloadAsPng() {
    const rawSVG = this.renderer?.getSVGForDownload();
    if (!rawSVG) return;
    const pngCanvas = document.createElement('canvas');
    document.body.appendChild(pngCanvas);
    const ctx = pngCanvas.getContext('2d');
    document.body.appendChild(rawSVG);
    pngCanvas.width = +rawSVG.getAttribute('width')!;
    pngCanvas.height = +rawSVG.getAttribute('height')!;
    if (pngCanvas.width > 32767 || pngCanvas.height > 32767) {
      const msg = `Failed to convert SVG to PNG. The maximum supported resolution is 32767 per dimension, but the given graph is ${pngCanvas.width}x${pngCanvas.height}. Please download SVG instead and convert it to png in another way`;
      alert(msg);
      return;
    }
    const image = new Image();
    image.onload = () => {
      ctx?.drawImage(image, 0, 0);
      setTimeout(() => {
        pngCanvas.toBlob(function (blob) {
          const downloadAnchor = document.createElement('a');
          downloadAnchor.download = 'diagram.png';
          downloadAnchor.href = URL.createObjectURL(blob!);
          downloadAnchor.click();
          document.body.removeChild(pngCanvas);
          document.body.removeChild(rawSVG);
        });
      }, 0);
    };
    image.onerror = (event, s, l, e) => {
      console.error(event, s, l, e);
    };
    image.src =
      'data:image/svg+xml;base64,' +
      btoa(new XMLSerializer().serializeToString(rawSVG));
  }
}
