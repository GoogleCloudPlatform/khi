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

import { GraphRoot } from '../graph-root';
import { GraphRenderer } from '../renderer';

export type SvgTestCallback<T> = (r: T) => void;

/**
 * Create a new graph root element for this spec
 * @param title Title of the spec
 * @param size Size of the wrapper div element. Size will be a square.
 * @param callback Test callback
 */
export function graphRootIt(
  title: string,
  callback: SvgTestCallback<GraphRoot>,
) {
  it(title, () => {
    const gr = generateRoot();
    callback(gr);
  });
}

export function rendererIt(
  title: string,
  callback: SvgTestCallback<GraphRenderer>,
) {
  it(title, () => {
    const gr = generateRenderer();
    callback(gr);
  });
}

function generateRoot() {
  const gr = new GraphRoot();
  const svgWrap = document.createElement('div');
  gr.attach(svgWrap);
  return gr;
}

function generateRenderer() {
  const svgWrap = document.createElement('div');
  return new GraphRenderer(svgWrap);
}
