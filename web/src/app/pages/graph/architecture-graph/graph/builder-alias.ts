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

import {
  AlighedGraphObjectContainer,
  Direction,
  ElementStyle,
  GraphObject,
} from './base/base-containers';
import { GraphPattern } from './base/defs-child-elements';
import {
  ArchCircle,
  ArchEmpty,
  ArchLabel,
  ArchRect,
  SizedRect,
} from './base/element';
import { ContentMetadataPairLayout } from './base/layout';
import { PathPipe } from './base/path';

export function $alignedGroup(
  direction: Direction,
): AlighedGraphObjectContainer {
  return new AlighedGraphObjectContainer(direction);
}

export function $alignedBox(
  direction: Direction,
  backgroundStyle: ElementStyle,
  gap: number,
  padding: number[],
  children: GraphObject[],
): ArchRect {
  return $rect()
    .withStyle(backgroundStyle)
    .withChildren([
      $alignedGroup(direction)
        .withMargin(padding[0], padding[1], padding[2], padding[3])
        .withStyle(backgroundStyle)
        .withGap(gap)
        .withChildren(children),
    ]);
}

export function $label(txt: string): ArchLabel {
  return new ArchLabel(txt);
}

export function $boxed_label(
  txt: string,
  boxStyle: ElementStyle,
  txtStyle: ElementStyle,
  padding: number[],
): ArchRect {
  return $rect()
    .withStyle(boxStyle)
    .withChildren([
      $label(txt)
        .withStyle(txtStyle)
        .withMargin(padding[0], padding[1], padding[2], padding[3]),
    ]);
}

export function $pattern(width: number, height: number): GraphPattern {
  return new GraphPattern(width, height);
}

export function $empty(): ArchEmpty {
  return new ArchEmpty();
}

export function $rect(): ArchRect {
  return new ArchRect();
}

export function $sized_rect(w: number, h: number): SizedRect {
  return new SizedRect(w, h);
}

export function $circle(): ArchCircle {
  return new ArchCircle();
}

export function $hpair(
  left: GraphObject,
  right: GraphObject,
): ContentMetadataPairLayout {
  return new ContentMetadataPairLayout(left, right);
}

export function $pathPipe(direction: Direction, id: string): PathPipe {
  return new PathPipe(direction, id);
}
