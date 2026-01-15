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

import { DestroyRef, Injectable, NgZone } from '@angular/core';

/**
 * Manages the WebGL rendering loop using requestAnimationFrame.
 *
 * This service is responsible for:
 * - running the render loop outside of the Angular zone for performance.
 * - broadcasting render events to registered renderers.
 * - managing one-time pre-render tasks (e.g. for synchronizing CSS and Canvas updates).
 */
@Injectable({ providedIn: 'root' })
export class RenderingLoopManager {
  private readonly onceBeforeRenderHandlers = new Set<() => void>();
  private readonly renderHandlers = new Set<() => void>();
  private animationFrameId: number | null = null;
  private _loopIndex = 0;

  /**
   * Returns the current loop index.
   * This index is incremented every time the render method is called.
   */
  get loopIndex() {
    return this._loopIndex;
  }

  /**
   * Register a callback to be called once before the next render.
   * This is typically used for applying CSS properties related to the canvas to avoid flickering effects caused by timing differences between CSS updates and canvas rendering.
   */
  registerOnceBeforeRenderHandler(callback: () => void): void {
    this.onceBeforeRenderHandlers.add(callback);
  }

  /**
   * Register a callback to be called before every render frame.
   * @param destroyRef The DestroyRef to automatically unregister the handler when the component/service is destroyed.
   * @param callback The function to call before each render.
   */
  registerRenderHandler(destroyRef: DestroyRef, callback: () => void): void {
    this.renderHandlers.add(callback);
    destroyRef.onDestroy(() => {
      this.renderHandlers.delete(callback);
    });
  }

  /**
   * Starts the rendering loop.
   * @param ngZone The NgZone to run the loop outside of Angular to improve performance.
   * @param destroyRef The DestroyRef to automatically stop the loop when the context is destroyed.
   * @throws Error if the loop is already started.
   */
  start(ngZone: NgZone, destroyRef: DestroyRef): void {
    if (this.animationFrameId === null) {
      this.requestAnimationFrame(ngZone);
    } else {
      throw new Error('Rendering loop is already started');
    }
    destroyRef.onDestroy(() => {
      this.stop();
    });
  }

  /**
   * Stops the rendering loop.
   * Safe to call even if the loop is not running.
   */
  stop(): void {
    if (this.animationFrameId !== null) {
      cancelAnimationFrame(this.animationFrameId);
      this.animationFrameId = null;
    }
  }

  private render(): void {
    this._loopIndex++;
    this.animationFrameId = null;
    this.onceBeforeRenderHandlers.forEach((handler) => handler());
    this.onceBeforeRenderHandlers.clear();
    this.renderHandlers.forEach((handler) => handler());
  }

  private requestAnimationFrame(ngZone: NgZone): void {
    this.animationFrameId = ngZone.runOutsideAngular(() =>
      requestAnimationFrame(() => {
        this.render();
        this.requestAnimationFrame(ngZone);
      }),
    );
  }
}
