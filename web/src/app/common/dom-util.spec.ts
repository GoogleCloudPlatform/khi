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
  isMac,
  isSearchShortcut,
  isEventFromOverlay,
} from 'src/app/common/dom-util';

describe('dom-util', () => {
  describe('isMac', () => {
    let originalUserAgent: string;
    beforeEach(() => {
      originalUserAgent = navigator.userAgent;
    });

    afterEach(() => {
      Object.defineProperty(navigator, 'userAgent', {
        value: originalUserAgent,
        configurable: true,
      });
    });

    it('should return true if userAgent contains Mac OS X', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
        configurable: true,
      });
      expect(isMac()).toBeTrue();
    });

    it('should return false if userAgent is Windows', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
        configurable: true,
      });
      expect(isMac()).toBeFalse();
    });

    it('should return false if navigator is undefined', () => {
      const originalNavigator = globalThis.navigator;
      try {
        Object.defineProperty(globalThis, 'navigator', {
          value: undefined,
          configurable: true,
        });
        expect(isMac()).toBeFalse();
      } finally {
        Object.defineProperty(globalThis, 'navigator', {
          value: originalNavigator,
          configurable: true,
        });
      }
    });
  });

  describe('isSearchShortcut', () => {
    let originalUserAgent: string;
    beforeEach(() => {
      originalUserAgent = navigator.userAgent;
    });

    afterEach(() => {
      Object.defineProperty(navigator, 'userAgent', {
        value: originalUserAgent,
        configurable: true,
      });
    });

    it('should return true for Cmd+F on Mac', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mac OS X',
        configurable: true,
      });
      expect(
        isSearchShortcut(
          new KeyboardEvent('keydown', { key: 'f', metaKey: true }),
        ),
      ).toBeTrue();
    });

    it('should return false for Ctrl+F on Mac', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mac OS X',
        configurable: true,
      });
      expect(
        isSearchShortcut(
          new KeyboardEvent('keydown', { key: 'f', ctrlKey: true }),
        ),
      ).toBeFalse();
    });

    it('should return true for Ctrl+F on Windows', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Windows NT',
        configurable: true,
      });
      expect(
        isSearchShortcut(
          new KeyboardEvent('keydown', { key: 'f', ctrlKey: true }),
        ),
      ).toBeTrue();
    });

    it('should return false for Cmd+F on Windows', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Windows NT',
        configurable: true,
      });
      expect(
        isSearchShortcut(
          new KeyboardEvent('keydown', { key: 'f', metaKey: true }),
        ),
      ).toBeFalse();
    });

    it('should return false for non-F keys', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Windows NT',
        configurable: true,
      });
      expect(
        isSearchShortcut(
          new KeyboardEvent('keydown', { key: 'g', ctrlKey: true }),
        ),
      ).toBeFalse();
    });
  });

  describe('isEventFromOverlay', () => {
    it('should return true if target is inside an overlay pane', () => {
      const target = document.createElement('div');
      const pane = document.createElement('div');
      pane.classList.add('cdk-overlay-pane');
      pane.appendChild(target);
      expect(isEventFromOverlay({ target } as unknown as Event)).toBeTrue();
    });

    it('should return false if target is not inside an overlay pane', () => {
      const target = document.createElement('div');
      expect(isEventFromOverlay({ target } as unknown as Event)).toBeFalse();
    });

    it('should return false if target is null', () => {
      expect(
        isEventFromOverlay({ target: null } as unknown as Event),
      ).toBeFalse();
    });
  });
});
