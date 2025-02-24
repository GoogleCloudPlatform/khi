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

import { AnchorPoints, Direction, ElementStyle } from './base-containers';
import { $alignedGroup, $label, $rect } from '../builder-alias';
import { TRANSPARENT_BOX } from '../styles';
import { graphRootIt } from '../test/graph-test-utiility';

const BOX_STYLE: ElementStyle = {
  fill: 'transparent',
  stroke: 'green',
  'stroke-width': '3px',
};

const DEBUG_STYLE: ElementStyle = {
  fill: 'transparent',
  stroke: 'red',
  'stroke-width': '2px',
};

describe('Layout', () => {
  graphRootIt('Box containing a label', (root) => {
    const nonExceededLayoutLoop = root
      .withChildren([
        $rect()
          .withStyle(BOX_STYLE)
          .withChildren([$label('foo')]),
      ])
      .render();
    expect(nonExceededLayoutLoop).toBeTrue();
    const rect = root.getChildren()[0];
    expect(rect.element!.nodeName).toBe('rect');
    const label = rect.getChildren()[0];
    expect(label.element!.nodeName).toBe('text');

    const bbox = label.element!.getBoundingClientRect();
    expect(rect.element!.getAttribute('width')).toBe(bbox.width + '');
    expect(rect.element!.getAttribute('height')).toBe(bbox.height + '');
  });

  graphRootIt('Box containing a label with margins', (root) => {
    const nonExceededLayoutLoop = root
      .withChildren([
        $rect()
          .withStyle(BOX_STYLE)
          .withChildren([$label('foo').withMargin(10, 10, 10, 10)]),
      ])
      .render();
    expect(nonExceededLayoutLoop).toBeTrue();
    const rect = root.getChildren()[0];
    expect(rect.element!.nodeName).toBe('rect');
    const label = rect.getChildren()[0];
    expect(label.element!.nodeName).toBe('text');

    const bbox = label.element!.getBoundingClientRect();
    expect(rect.element!.getAttribute('width')).toBe(bbox.width + 20 + '');
    expect(rect.element!.getAttribute('height')).toBe(bbox.height + 20 + '');
  });

  graphRootIt('AlignedGroup should expand children', (root) => {
    const nonExceededLayoutLoop = root
      .withChildren([
        $rect()
          .withStyle(BOX_STYLE)
          .withChildren([
            $alignedGroup(Direction.Vertical)
              .withMargin(10, 10, 10, 10)
              .withGap(10)
              .withChildren([
                $rect()
                  .withStyle(BOX_STYLE)
                  .withChildren([$label('HOGE')]),
                $rect()
                  .withStyle(BOX_STYLE)
                  .withChildren([$label('HOGEHOGEHOGE')]),
              ]),
          ]),
      ])
      .render();

    expect(nonExceededLayoutLoop).toBeTrue();
    // TODO: Is this really needed feature?
    // expect(labelBox1.element.getAttribute("width")).toBe(labelBox2.element.getAttribute("width"))
  });

  graphRootIt('Nested aligned group', (root) => {
    const nonExceededLayoutLoop = root
      .withChildren([
        $rect()
          .withStyle(BOX_STYLE)
          .withChildren([
            $alignedGroup(Direction.Vertical)
              .withMargin(30, 30, 30, 30)
              .withGap(10)
              .withChildren([
                $rect()
                  .withStyle(BOX_STYLE)
                  .withChildren([$label('HOGE')]),
                $alignedGroup(Direction.Vertical).withChildren([
                  $rect()
                    .withStyle(BOX_STYLE)
                    .withChildren([$label('HOGE')]),
                  $rect()
                    .withStyle(BOX_STYLE)
                    .withChildren([$label('HOGEHOGEHOGE')]),
                ]),
              ]),
          ]),
      ])
      .render();

    expect(nonExceededLayoutLoop).toBeTrue();
  });

  graphRootIt('Aligned group with margin', (root) => {
    const nonExceededLayoutLoop = root
      .withChildren([
        $rect()
          .withStyle(DEBUG_STYLE)
          .withChildren([
            $alignedGroup(Direction.Vertical).withChildren([
              $rect()
                .withStyle(BOX_STYLE)
                .withChildren([
                  $rect()
                    .withStyle(TRANSPARENT_BOX)
                    .withAnchor(AnchorPoints.CENTER)
                    .withPivot(AnchorPoints.CENTER)
                    .withChildren([$label('foo').withMargin(0, 20, 0, 20)]),
                ]),
            ]),
          ]),
      ])
      .render();

    expect(nonExceededLayoutLoop).toBeTrue();
    const label = root
      .getChildren()[0]
      .getChildren()[0]
      .getChildren()[0]
      .getChildren()[0]
      .getChildren()[0];
    expect(label.element!.nodeName).toBe('text');
    expect(label.element!.getAttribute('x')).toBe('20');
  });
});
