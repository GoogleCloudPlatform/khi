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

import { downloadBlob } from './download-util';

describe('download-util', () => {
  it('should create object URL, append anchor, click, and clean up', () => {
    const mockBlob = new Blob(['test content'], { type: 'text/plain' });
    const createObjectURLSpy = spyOn(
      window.URL,
      'createObjectURL',
    ).and.returnValue('blob:mock-url');
    const revokeObjectURLSpy = spyOn(window.URL, 'revokeObjectURL');
    let appendedAnchor: HTMLAnchorElement | null = null;
    spyOn(document.body, 'appendChild').and.callFake(
      <T extends Node>(node: T): T => {
        if (node instanceof HTMLAnchorElement) {
          appendedAnchor = node;
        }
        return node;
      },
    );

    downloadBlob(mockBlob, 'test-file.txt');

    expect(createObjectURLSpy).toHaveBeenCalledWith(mockBlob);
    const anchor = appendedAnchor as HTMLAnchorElement | null;
    expect(anchor?.download).toBe('test-file.txt');
    expect(anchor?.href).toBe('blob:mock-url');
    expect(revokeObjectURLSpy).toHaveBeenCalledWith('blob:mock-url');
  });

  it('should clean up DOM and revoke object URL even if anchor click throws an error', () => {
    const mockBlob = new Blob(['test content'], { type: 'text/plain' });
    const createObjectURLSpy = spyOn(
      window.URL,
      'createObjectURL',
    ).and.returnValue('blob:mock-url');
    const revokeObjectURLSpy = spyOn(window.URL, 'revokeObjectURL');
    spyOn(HTMLAnchorElement.prototype, 'click').and.throwError(
      new Error('Click failed'),
    );

    expect(() => {
      downloadBlob(mockBlob, 'test-file.txt');
    }).toThrowError('Click failed');

    expect(createObjectURLSpy).toHaveBeenCalledWith(mockBlob);
    expect(revokeObjectURLSpy).toHaveBeenCalledWith('blob:mock-url');
    expect(
      document.body.querySelector('a[download="test-file.txt"]'),
    ).toBeNull();
  });
});
