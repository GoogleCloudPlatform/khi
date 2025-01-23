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

import { BreaklinePipe } from './breakline.pipe';

describe('BreaklinePipe', () => {
  it('create an instance', () => {
    const pipe = new BreaklinePipe();
    expect(pipe).toBeTruthy();
  });
  it('convert string not containing breakline', () => {
    const pipe = new BreaklinePipe();
    expect(pipe.transform('foo')).toBe('foo');
  });
  it('convert string containing breaklines', () => {
    const pipe = new BreaklinePipe();
    expect(pipe.transform('a\nb\nc')).toBe('a<br/>b<br/>c');
  });
});
