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

import { FirstOrUndefined } from './first-or-null.pipe';

describe('FirstOrNullPipe', () => {
  it('create an instance', () => {
    const pipe = new FirstOrUndefined();
    expect(pipe).toBeTruthy();
  });
  it('returns null when null was passed', () => {
    const pipe = new FirstOrUndefined();
    expect(pipe.transform(null)).toBeUndefined();
  });
  it('returns null when empty array was passed', () => {
    const pipe = new FirstOrUndefined();
    expect(pipe.transform([])).toBeUndefined();
  });
  it('returns first element', () => {
    const pipe = new FirstOrUndefined();
    expect(pipe.transform([1, 2, 3])).toBe(1);
  });
});
