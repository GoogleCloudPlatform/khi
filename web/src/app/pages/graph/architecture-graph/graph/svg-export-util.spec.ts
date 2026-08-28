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

import {
  GraphExportFormat,
  MAX_CANVAS_DIMENSION,
  downloadSvg,
  downloadPng,
  exportSvgToBlob,
  exportSvgToPngBlob,
  generateDefaultGraphFilename,
} from './svg-export-util';

describe('svg-export-util', () => {
  let mockSvg: SVGSVGElement;

  beforeEach(() => {
    mockSvg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    mockSvg.setAttribute('width', '100');
    mockSvg.setAttribute('height', '100');
  });

  describe('GraphExportFormat', () => {
    it('should define Svg and Png format values', () => {
      expect(GraphExportFormat.Svg).toBe('svg');
      expect(GraphExportFormat.Png).toBe('png');
    });
  });

  describe('generateDefaultGraphFilename', () => {
    it('should generate a timestamped filename with the correct prefix and extension', () => {
      const svgFilename = generateDefaultGraphFilename(GraphExportFormat.Svg);
      expect(svgFilename).toMatch(/^khi-graph-\d{8}-\d{6}\.svg$/);

      const pngFilename = generateDefaultGraphFilename(GraphExportFormat.Png);
      expect(pngFilename).toMatch(/^khi-graph-\d{8}-\d{6}\.png$/);
    });
  });

  describe('exportSvgToBlob', () => {
    it('should serialize SVG to Blob', () => {
      const blob = exportSvgToBlob(mockSvg);
      expect(blob).toBeTruthy();
      expect(blob.type).toBe('image/svg+xml');
    });
  });

  describe('exportSvgToPngBlob', () => {
    it('should reject if width exceeds maxDimension', async () => {
      mockSvg.setAttribute('width', '40000');
      mockSvg.setAttribute('height', '100');
      await expectAsync(
        exportSvgToPngBlob(mockSvg, MAX_CANVAS_DIMENSION),
      ).toBeRejectedWithError(/exceed the maximum supported resolution/);
    });

    it('should reject if height exceeds maxDimension', async () => {
      mockSvg.setAttribute('width', '100');
      mockSvg.setAttribute('height', '40000');
      await expectAsync(
        exportSvgToPngBlob(mockSvg, MAX_CANVAS_DIMENSION),
      ).toBeRejectedWithError(/exceed the maximum supported resolution/);
    });

    it('should reject when canvas.toBlob returns null', async () => {
      spyOn(HTMLCanvasElement.prototype, 'toBlob').and.callFake((callback) => {
        callback(null);
      });

      await expectAsync(exportSvgToPngBlob(mockSvg)).toBeRejectedWithError(
        /Failed to generate PNG image from graph/,
      );
    });

    it('should reject when image loading fails and revoke object URL', async () => {
      const revokeSpy = spyOn(window.URL, 'revokeObjectURL').and.callThrough();
      spyOnProperty(Image.prototype, 'src', 'set').and.callFake(function (
        this: HTMLImageElement,
      ) {
        setTimeout(() => {
          this.onerror?.(new Event('error'));
        }, 0);
      });

      await expectAsync(exportSvgToPngBlob(mockSvg)).toBeRejectedWithError(
        /Failed to render SVG graph onto canvas for PNG export/,
      );
      expect(revokeSpy).toHaveBeenCalled();
    });

    it('should handle non-ASCII Unicode characters without error', async () => {
      const textElement = document.createElementNS(
        'http://www.w3.org/2000/svg',
        'text',
      );
      textElement.textContent = '日本語ポッド名・テスト';
      mockSvg.appendChild(textElement);

      const blob = await exportSvgToPngBlob(mockSvg);
      expect(blob).toBeTruthy();
      expect(blob.type).toBe('image/png');
    });

    it('should revoke object URL after successful rendering', async () => {
      const revokeSpy = spyOn(window.URL, 'revokeObjectURL').and.callThrough();
      await exportSvgToPngBlob(mockSvg);
      expect(revokeSpy).toHaveBeenCalled();
    });
  });

  describe('downloadSvg', () => {
    it('should serialize SVG to blob and trigger download', () => {
      let appendedAnchor: HTMLAnchorElement | null = null;
      spyOn(document.body, 'appendChild').and.callFake(
        <T extends Node>(node: T): T => {
          if (node instanceof HTMLAnchorElement) {
            appendedAnchor = node;
          }
          return node;
        },
      );
      spyOn(HTMLAnchorElement.prototype, 'click').and.callFake(() => {});

      downloadSvg(mockSvg, 'diagram.svg');
      const anchor = appendedAnchor as HTMLAnchorElement | null;
      expect(anchor?.download).toBe('diagram.svg');
      expect(anchor?.href).toContain('blob:');
    });
  });

  describe('downloadPng', () => {
    it('should rasterize to PNG blob and trigger download', async () => {
      let appendedAnchor: HTMLAnchorElement | null = null;
      spyOn(document.body, 'appendChild').and.callFake(
        <T extends Node>(node: T): T => {
          if (node instanceof HTMLAnchorElement) {
            appendedAnchor = node;
          }
          return node;
        },
      );
      spyOn(HTMLAnchorElement.prototype, 'click').and.callFake(() => {});

      await downloadPng(mockSvg, 'diagram.png');
      const anchor = appendedAnchor as HTMLAnchorElement | null;
      expect(anchor?.download).toBe('diagram.png');
      expect(anchor?.href).toContain('blob:');
    });
  });
});
