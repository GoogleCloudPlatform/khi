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
  computeIndexPaneWidth,
  IndexPaneRow,
  TextMeasurer,
} from 'src/app/pages/timeline-render/index-pane-width';

describe('computeIndexPaneWidth', () => {
  // Every character is 10px wide, so expected widths are easy to compute by hand.
  const measureText: TextMeasurer = (text) => text.length * 10;
  // Row chrome (border, padding, toggle and icon columns, gaps, chip padding and margins) is 85px.
  const rowChromePx = 85;

  const testCases: {
    desc: string;
    rows: IndexPaneRow[];
    want: number;
  }[] = [
    {
      desc: 'returns the minimum width when there are no rows',
      rows: [],
      want: 300,
    },
    {
      desc: 'returns the minimum width when every label fits',
      rows: [{ name: 'pod-a', layerName: 'pod', layer: 1 }],
      want: 300,
    },
    {
      desc: 'grows to fit the longest name, chip, and indentation',
      rows: [
        { name: 'short', layerName: 'pod', layer: 1 },
        // 30 chars name (300px) + 3 chars chip (30px + 1.5px letter spacing) + 2 levels of indent (8px)
        { name: 'a'.repeat(30), layerName: 'pod', layer: 3 },
      ],
      want: Math.ceil(rowChromePx + 8 + 300 + 31.5),
    },
    {
      desc: 'is capped at the maximum width',
      rows: [{ name: 'a'.repeat(200), layerName: 'pod', layer: 1 }],
      want: 800,
    },
  ];

  for (const tc of testCases) {
    it(tc.desc, () => {
      expect(
        computeIndexPaneWidth(tc.rows, measureText, 'Roboto', 300, 800),
      ).toBe(tc.want);
    });
  }

  it('passes the font family and sizes to the measurer', () => {
    const fonts: string[] = [];
    computeIndexPaneWidth(
      [{ name: 'pod-a', layerName: 'pod', layer: 1 }],
      (text, font) => {
        fonts.push(font);
        return 0;
      },
      'Roboto',
      300,
      800,
    );

    expect(fonts).toEqual(['400 14px Roboto', '500 10px Roboto']);
  });
});
