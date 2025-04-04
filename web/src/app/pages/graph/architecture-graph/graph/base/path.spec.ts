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

import { $alignedGroup, $sized_rect } from '../builder-alias';
import { graphRootIt } from '../test/graph-test-utiility';
import { AnchorPoints, Direction, ElementStyle } from './base-containers';
import { ArchRect } from './element';
import { PathPipe } from './path';

const POINT_STYLE: ElementStyle = {
  fill: 'orange',
};

describe('Path related class specs', () => {
  graphRootIt('with layout elements', (root) => {
    root
      .withChildren([
        $alignedGroup(Direction.Vertical)
          .withGap(30)
          .withChildren([
            $alignedGroup(Direction.Horizontal)
              .withGap(30)
              .withChildren([
                new PathPipe(Direction.Vertical, 'v-pipe-1'),
                $sized_rect(100, 150).withChildren([
                  $sized_rect(30, 30).withStyle(POINT_STYLE).withId('v1-p1'),
                  $sized_rect(30, 30)
                    .withStyle(POINT_STYLE)
                    .withMargin(50, 0, 0, 0)
                    .withId('v1-p2'),
                ]),
                new PathPipe(Direction.Vertical, 'v-pipe-2'),
                $sized_rect(100, 150).withChildren([
                  $sized_rect(30, 30).withStyle(POINT_STYLE).withId('v2-p1'),
                  $sized_rect(30, 30)
                    .withStyle(POINT_STYLE)
                    .withMargin(50, 0, 0, 0)
                    .withId('v2-p2'),
                ]),
              ]),
            new PathPipe(Direction.Horizontal, 'h-pipe'),
            $sized_rect(500, 50).withChildren([
              $sized_rect(30, 30).withStyle(POINT_STYLE).withId('h1-p1'),
              $sized_rect(30, 30)
                .withStyle(POINT_STYLE)
                .withMargin(0, 0, 0, 100)
                .withId('h1-p2'),
            ]),
          ]),
      ])
      .registerLayoutStep(0, () => {
        const v1Pipe1 = root.find('v-pipe-1') as PathPipe;
        const v2Pipe2 = root.find('v-pipe-2') as PathPipe;
        const hPipe = root.find('h-pipe') as PathPipe;

        v1Pipe1.connectPipe(hPipe);
        v2Pipe2.connectPipe(hPipe);

        const v1p1 = root.find('v1-p1') as ArchRect;
        const v1p2 = root.find('v1-p2') as ArchRect;
        const v2p1 = root.find('v2-p1') as ArchRect;
        const v2p2 = root.find('v2-p2') as ArchRect;
        const h1p1 = root.find('h1-p1') as ArchRect;
        const h1p2 = root.find('h1-p2') as ArchRect;
        hPipe.connectPoint(h1p1, AnchorPoints.TOP);
        v1Pipe1.connectPoint(v1p1, AnchorPoints.CENTER_LEFT);
        v1Pipe1.connectPoint(v1p2, AnchorPoints.CENTER_LEFT);
        v2Pipe2.connectPoint(v2p1, AnchorPoints.CENTER_LEFT);
        v2Pipe2.connectPoint(v2p2, AnchorPoints.CENTER_LEFT);
        hPipe.connectPoint(h1p2, AnchorPoints.TOP);
        hPipe.registerPath('h1-p2/h-pipe/v-pipe-1/v1-p1', 'arrow', 10, 90);
        hPipe.registerPath('h1-p2/h-pipe/v-pipe-1/v1-p2', 'arrow', 10, 90);
        hPipe.registerPath('h1-p1/h-pipe/v-pipe-2/v2-p2', 'circle', 10, 90);
      })
      .render();
  });
});
