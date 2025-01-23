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

import { InjectionToken, Provider } from '@angular/core';
import { ExtensionStore } from './extension-store';
import { TimelineNavigatorExtension } from './extension-types/timeline-navigator';
import { URLDataOpenerExtension } from './extension-types/url-data-opener';
import { LifecycleHookExtension } from './extension-types/lifecycle-hook';

/**
 * The injection token for KHIExtensionBundle
 */
export const KHI_FRONTEND_EXTENSION_BUNDLE =
  new InjectionToken<KHIExtensionBundle>('KHI_FRONTEND_EXTENSION_BUNDLE');

export type KHIExtensionInitHandler = (extension: KHIExtensionBundle) => void;

/**
 * KHIExtensionBundle provides APIs to extend KHI frontend for extension modules.
 * KHI frontend extensions register its extensions through the init function on the constructor.
 */
export class KHIExtensionBundle {
  /**
   * Returns a provider configuration for extension module.
   */
  public static forExtension(init: KHIExtensionInitHandler): Provider {
    return {
      multi: true,
      provide: KHI_FRONTEND_EXTENSION_BUNDLE,
      useValue: new KHIExtensionBundle(init),
    };
  }

  private extensionStore: ExtensionStore | null = null;

  constructor(private readonly init: KHIExtensionInitHandler) {}

  /**
   * This API must only be called from KHI app side.
   */
  public initializeExtension(store: ExtensionStore): void {
    this.extensionStore = store;
    this.init(this);
  }

  /**
   * Register the new extension for TimelineNavigator
   */
  public addTimelineNavigatorExtension(
    extension: TimelineNavigatorExtension,
  ): void {
    if (this.extensionStore === null) {
      throw new Error('KHIExtension is not initialized');
    }
    this.extensionStore.timelineNavigatorExtensions.push(extension);
  }

  /**
   * Register the new extension to load data somewhere else with URL hash.
   */
  public addURLDataOpenerExtension(extension: URLDataOpenerExtension): void {
    if (this.extensionStore === null) {
      throw new Error('KHIExtension is not initialized');
    }
    this.extensionStore.urlDataOenerExtensions.push(extension);
  }

  public addLifecycleHookExtension(extension: LifecycleHookExtension): void {
    if (this.extensionStore === null) {
      throw new Error('KHIExtension is not initialized');
    }
    this.extensionStore.lifecycleHookExtensions.push(extension);
  }
}
