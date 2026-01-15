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

import { DestroyRef, NgZone } from '@angular/core';
import { RenderingLoopManager } from './rendering-loop-manager';
import { TestBed } from '@angular/core/testing';

function waitNextFrame(): Promise<void> {
  return new Promise((resolve) => {
    requestAnimationFrame(() => resolve());
  });
}

describe('RenderingLoopManager', () => {
  let manager: RenderingLoopManager;
  let ngZone: NgZone;
  let destroyRef: DestroyRef;
  let runOutsideAngularSpy: jasmine.Spy;
  let mockDestroyRef: WeakMap<object, () => void>;

  beforeEach(() => {
    mockDestroyRef = new WeakMap();
    const destroyRefSpy = jasmine.createSpyObj('DestroyRef', ['onDestroy']);
    destroyRefSpy.onDestroy.and.callFake((callback: () => void) => {
      mockDestroyRef.set(destroyRefSpy, callback);
    });

    TestBed.configureTestingModule({
      providers: [
        RenderingLoopManager,
        { provide: DestroyRef, useValue: destroyRefSpy },
      ],
    });

    manager = TestBed.inject(RenderingLoopManager);
    ngZone = TestBed.inject(NgZone);
    destroyRef = TestBed.inject(DestroyRef);
    runOutsideAngularSpy = spyOn(ngZone, 'runOutsideAngular').and.callFake(
      (fn) => fn(),
    );
  });

  afterEach(() => {
    manager.stop();
  });

  it('should be created', () => {
    expect(manager).toBeTruthy();
  });

  describe('start', () => {
    it('should start the loop', async () => {
      const initialIndex = manager.loopIndex;
      manager.start(ngZone, destroyRef);

      await waitNextFrame();

      expect(manager.loopIndex).toBeGreaterThan(initialIndex);
      expect(runOutsideAngularSpy).toHaveBeenCalled();
    });

    it('should throw error if already started', async () => {
      manager.start(ngZone, destroyRef);

      await waitNextFrame();

      expect(() => manager.start(ngZone, destroyRef)).toThrowError(
        'Rendering loop is already started',
      );
    });
  });

  describe('stop', () => {
    it('should stop increments', async () => {
      manager.start(ngZone, destroyRef);

      await waitNextFrame();

      const indexBeforeStop = manager.loopIndex;
      manager.stop();

      await waitNextFrame();

      expect(manager.loopIndex).toBe(indexBeforeStop);
    });

    it('should be safe to call if not started', () => {
      expect(() => manager.stop()).not.toThrow();
    });
  });

  describe('render loop', () => {
    it('should increment loopIndex', async () => {
      manager.start(ngZone, destroyRef);
      const initialIndex = manager.loopIndex;

      await waitNextFrame();

      expect(manager.loopIndex).toBeGreaterThan(initialIndex);
    });

    it('should execute registered render handlers', async () => {
      const handler = jasmine.createSpy('handler');
      manager.registerRenderHandler(destroyRef, handler);
      manager.start(ngZone, destroyRef);

      await waitNextFrame();

      expect(handler).toHaveBeenCalled();
    });

    it('should execute onceBeforeRenderHandlers only once', async () => {
      const handler = jasmine.createSpy('handler');
      manager.registerOnceBeforeRenderHandler(handler);
      manager.start(ngZone, destroyRef);

      await waitNextFrame();
      expect(handler).toHaveBeenCalledTimes(1);

      await waitNextFrame();
      expect(handler).toHaveBeenCalledTimes(1); // Still 1
    });
  });
});
