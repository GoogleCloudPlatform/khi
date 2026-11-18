/**
 * Copyright 2025 Google LLC
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

import type { Preview } from '@storybook/angular';
import { setCompodocJson } from '@storybook/addon-docs/angular';
import { addons } from 'storybook/preview-api';
import docJson from '../documentation.json';
setCompodocJson(docJson);

const isVrtMode =
  typeof window !== 'undefined' &&
  new URLSearchParams(window.location.search).get('vrt') === '1';

let vrtStoryId = '';
let vrtStatus = 'pending';
let vrtErrorMessage = '';
let vrtParameters: KhiVrtParameters = {};
let vrtIsFullscreen = false;

function formatChannelError(payload: unknown): string {
  if (typeof payload === 'string') {
    return payload;
  }
  if (payload && typeof payload === 'object') {
    const rec = payload as Record<string, unknown>;
    if (typeof rec['message'] === 'string') {
      return rec['message'];
    }
    if (typeof rec['description'] === 'string') {
      return rec['description'];
    }
    if (typeof rec['title'] === 'string') {
      return rec['title'];
    }
  }
  return 'story failed to render';
}

function nextAnimationFrame(): Promise<void> {
  return new Promise((resolve) => requestAnimationFrame(() => resolve()));
}

function pauseAllAnimations(): void {
  if (typeof document.getAnimations !== 'function') {
    return;
  }
  for (const animation of document.getAnimations()) {
    try {
      animation.pause();
      animation.currentTime = 0;
    } catch {
      // Ignore animations whose timeline does not allow seeking.
    }
  }
}

async function waitForSelectorState(
  selector: string,
  expectPresent: boolean,
  timeoutMs: number,
): Promise<boolean> {
  const deadline = performance.now() + timeoutMs;
  while (performance.now() < deadline) {
    const exists = document.querySelector(selector) !== null;
    if (exists === expectPresent) {
      return true;
    }
    await nextAnimationFrame();
  }
  return (document.querySelector(selector) !== null) === expectPresent;
}

async function waitForImages(): Promise<void> {
  const images = Array.from(document.querySelectorAll('img'));
  await Promise.all(
    images.map(async (img) => {
      if (img.complete) {
        return;
      }
      await new Promise<void>((resolve) => {
        const done = () => resolve();
        img.addEventListener('load', done, { once: true });
        img.addEventListener('error', done, { once: true });
      });
    }),
  );
}

function getCurrentVrtState(): KhiVrtState {
  if (
    vrtStatus === 'pending' &&
    document.body.classList.contains('sb-show-errordisplay')
  ) {
    const msg =
      document.querySelector('#error-message')?.textContent?.trim() ||
      'Storybook error display shown';
    return {
      storyId: vrtStoryId,
      status: 'error',
      errorMessage: msg,
      parameters: vrtParameters,
      isFullscreen: vrtIsFullscreen,
    };
  }
  return {
    storyId: vrtStoryId,
    status: vrtStatus,
    errorMessage: vrtErrorMessage,
    parameters: vrtParameters,
    isFullscreen: vrtIsFullscreen,
  };
}

async function waitForVrtReady(): Promise<KhiVrtState> {
  const timeoutMs = 15_000;
  const deadline = performance.now() + timeoutMs;

  while (performance.now() < deadline) {
    const state = getCurrentVrtState();
    if (state.status === 'error') {
      return state;
    }
    if (
      state.status === 'finished' ||
      (state.status === 'rendered' &&
        document.body.classList.contains('sb-show-main'))
    ) {
      break;
    }
    await nextAnimationFrame();
  }

  if (vrtStatus === 'pending') {
    const fallback = getCurrentVrtState();
    if (fallback.status === 'error') {
      return fallback;
    }
    return {
      storyId: vrtStoryId,
      status: 'error',
      errorMessage: 'timed out waiting for Storybook story to finish rendering',
      parameters: vrtParameters,
      isFullscreen: vrtIsFullscreen,
    };
  }

  if (vrtStatus === 'error') {
    return getCurrentVrtState();
  }

  await document.fonts.ready;
  await waitForImages();

  if (vrtParameters.readySelector) {
    const matched = await waitForSelectorState(
      vrtParameters.readySelector,
      true,
      10_000,
    );
    if (!matched) {
      return {
        storyId: vrtStoryId,
        status: 'error',
        errorMessage: `timed out waiting for readySelector: ${vrtParameters.readySelector}`,
        parameters: vrtParameters,
        isFullscreen: vrtIsFullscreen,
      };
    }
  }

  if (vrtParameters.waitForHiddenSelector) {
    const hidden = await waitForSelectorState(
      vrtParameters.waitForHiddenSelector,
      false,
      10_000,
    );
    if (!hidden) {
      return {
        storyId: vrtStoryId,
        status: 'error',
        errorMessage: `timed out waiting for selector to hide: ${vrtParameters.waitForHiddenSelector}`,
        parameters: vrtParameters,
        isFullscreen: vrtIsFullscreen,
      };
    }
  }

  if (typeof window.__khiVRTWaitSettled === 'function') {
    await window.__khiVRTWaitSettled(10_000);
  } else {
    pauseAllAnimations();
    await nextAnimationFrame();
  }

  return {
    storyId: vrtStoryId,
    status: 'ready',
    errorMessage: '',
    parameters: vrtParameters,
    isFullscreen: vrtIsFullscreen,
  };
}

if (isVrtMode) {
  const channel = addons.getChannel();
  channel.on('storyRendered', (storyId: unknown) => {
    if (typeof storyId === 'string') {
      vrtStoryId = storyId;
    }
    if (vrtStatus === 'pending') {
      vrtStatus = 'rendered';
    }
  });
  channel.on('storyFinished', (payload: unknown) => {
    if (payload && typeof payload === 'object') {
      const rec = payload as Record<string, unknown>;
      if (typeof rec['storyId'] === 'string') {
        vrtStoryId = rec['storyId'];
      }
      if (rec['status'] === 'error') {
        vrtStatus = 'error';
        vrtErrorMessage = vrtErrorMessage || 'storyFinished reported error';
        return;
      }
    }
    if (vrtStatus !== 'error') {
      vrtStatus = 'finished';
    }
  });
  for (const errEvent of [
    'storyErrored',
    'storyThrewException',
    'storyMissing',
    'playFunctionThrewException',
    'unhandledErrorsWhilePlaying',
  ]) {
    channel.on(errEvent, (payload: unknown) => {
      vrtStatus = 'error';
      vrtErrorMessage = `${errEvent}: ${formatChannelError(payload)}`;
    });
  }

  window.__khiVRT = {
    getState: getCurrentVrtState,
    waitForReady: waitForVrtReady,
  };
}

const preview: Preview = {
  decorators: isVrtMode
    ? [
        (storyFn, context) => {
          vrtStoryId = context.id;
          vrtParameters =
            (context.parameters?.['vrt'] as KhiVrtParameters | undefined) ?? {};
          vrtIsFullscreen = context.parameters?.['layout'] === 'fullscreen';
          return storyFn();
        },
      ]
    : [],
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
};

// Workaround to use BigInt in Storybook.
// Ref: https://github.com/storybookjs/storybook/issues/22452#issuecomment-1538474857
(window.BigInt.prototype as unknown as { toJSON: () => string }).toJSON =
  function () {
    return this.toString();
  };

export default preview;
