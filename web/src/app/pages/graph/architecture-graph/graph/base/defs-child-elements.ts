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

import { GraphObjectWithElementType } from './base-containers';

/**
 * Graph wrapper that can be placed just under <defs>
 */
export class GraphDefsChildItem<
  T extends SVGElement,
> extends GraphObjectWithElementType<T> {}

/**
 * Graph wrapper for <pattern>
 */
export class GraphPattern extends GraphDefsChildItem<SVGPatternElement> {
  constructor(width: number, height: number) {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'pattern'));
    this.transform.onSvgElementUpdate = (pos, size) => {
      this.withMinSize(size.width, size.height);
      return;
    };
    this.withMinSize(width, height);
    this.withStyle({
      width: `${width}px`,
      height: `${height}px`,
      viewBox: `0,0,${width},${height}`,
      patternUnits: 'userSpaceOnUse',
    });
  }
}
