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

import { downloadBlob } from 'src/app/utils/download-util';
import { generateTimestampedFilename } from 'src/app/utils/time-format-util';

/**
 * Defines supported export formats for the architecture graph.
 */
export enum GraphExportFormat {
  Svg = 'svg',
  Png = 'png',
}

/**
 * Defines the maximum canvas dimension supported across standard browsers (Safari/Chromium limit).
 */
export const MAX_CANVAS_DIMENSION = 32767;

/**
 * Generates the default timestamped filename for architecture graph exports.
 *
 * @param format - Export file format.
 * @returns Formatted filename string (e.g. 'khi-graph-20260827-153045.svg').
 */
export function generateDefaultGraphFilename(
  format: GraphExportFormat,
): string {
  return generateTimestampedFilename('khi-graph', format);
}

/**
 * Serializes an SVGElement into an SVG Blob.
 *
 * @param svgElement - The SVG element to serialize.
 * @returns Blob containing the SVG XML content.
 */
export function exportSvgToBlob(svgElement: SVGElement): Blob {
  const serializer = new XMLSerializer();
  const svgXml = serializer.serializeToString(svgElement);
  return new Blob([svgXml], { type: 'image/svg+xml' });
}

/**
 * Rasterizes an SVGElement into a PNG Blob using an off-screen HTMLCanvasElement.
 *
 * @param svgElement - The SVG element to rasterize.
 * @param maxDimension - The maximum allowed dimension before throwing an error.
 * @returns Promise resolving to a PNG Blob.
 */
export function exportSvgToPngBlob(
  svgElement: SVGElement,
  maxDimension = MAX_CANVAS_DIMENSION,
): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const width = Number(svgElement.getAttribute('width'));
    const height = Number(svgElement.getAttribute('height'));

    if (width > maxDimension || height > maxDimension) {
      reject(
        new Error(
          `Graph dimensions (${width}x${height}) exceed the maximum supported resolution of ${maxDimension}px per dimension. Please download as SVG instead.`,
        ),
      );
      return;
    }

    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;

    const ctx = canvas.getContext('2d');
    const image = new Image();

    const blob = exportSvgToBlob(svgElement);
    const url = URL.createObjectURL(blob);

    image.onload = () => {
      URL.revokeObjectURL(url);
      try {
        ctx?.drawImage(image, 0, 0);
        canvas.toBlob((blob) => {
          if (blob) {
            resolve(blob);
          } else {
            reject(
              new Error(
                'Failed to generate PNG image from graph. Please download as SVG instead.',
              ),
            );
          }
        }, 'image/png');
      } catch (err) {
        reject(err);
      }
    };

    image.onerror = () => {
      URL.revokeObjectURL(url);
      reject(
        new Error(
          'Failed to render SVG graph onto canvas for PNG export. Please download as SVG instead.',
        ),
      );
    };

    image.src = url;
  });
}

/**
 * Exports and triggers a download of the given SVG element as an SVG file.
 *
 * @param svgElement - The SVG element to download.
 * @param filename - The destination filename.
 */
export function downloadSvg(svgElement: SVGElement, filename: string): void {
  const blob = exportSvgToBlob(svgElement);
  downloadBlob(blob, filename);
}

/**
 * Rasterizes and triggers a download of the given SVG element as a PNG file.
 *
 * @param svgElement - The SVG element to download.
 * @param filename - The destination filename.
 */
export async function downloadPng(
  svgElement: SVGElement,
  filename: string,
): Promise<void> {
  const blob = await exportSvgToPngBlob(svgElement);
  downloadBlob(blob, filename);
}
