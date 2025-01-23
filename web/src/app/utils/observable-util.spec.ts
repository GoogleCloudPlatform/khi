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

import { map, take, timer } from 'rxjs';
import { monitorElementHeight } from './observable-util';

describe('observable-util', () => {
  describe('monitorElementHeight', () => {
    it('emits heights on resize events', (done) => {
      const element = document.createElement('div');
      document.body.appendChild(element);
      element.style.height = '100px';

      const sizes = [200, 200, 100];
      const gotValues: number[] = [];
      const observable = monitorElementHeight(element);
      observable.subscribe((v) => {
        gotValues.push(v);
      });
      timer(0, 100)
        .pipe(
          take(3),
          map((v) => `${sizes[v]}px`),
        )
        .subscribe((size) => {
          element.style.height = size;
        });

      timer(1000).subscribe(() => {
        expect(gotValues).toEqual([100, 200, 100]);
        done();
      });
    });
  });
});
