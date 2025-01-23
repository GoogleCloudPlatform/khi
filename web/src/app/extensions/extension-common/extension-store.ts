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

import { TimelineEntry } from 'src/app/store/timeline';
import {
  DisplayableTimelineNavigatorExtension,
  TimelineNavigatorExtension,
} from './extension-types/timeline-navigator';
import { InjectionToken, Injector, runInInjectionContext } from '@angular/core';
import { URLDataOpenerExtension } from './extension-types/url-data-opener';
import {
  LifecycleHookExtension,
  PageType,
} from './extension-types/lifecycle-hook';
import { InspectionData } from 'src/app/store/inspection-data';
import { ReferenceResolverStore } from 'src/app/common/loader/reference-resolver';

/**
 * An injectio toke to get the instnce of ExtensionStore.
 */
export const EXTENSION_STORE = new InjectionToken<ExtensionStore>(
  'EXTENSION_STORE',
);

/**
 * ExtensionStore is the type to hold the reference to plugin instances of each extensible points in KHI.
 */
export class ExtensionStore {
  public readonly timelineNavigatorExtensions: TimelineNavigatorExtension[] =
    [];

  public readonly urlDataOenerExtensions: URLDataOpenerExtension[] = [];

  public readonly lifecycleHookExtensions: LifecycleHookExtension[] = [];

  /**
   * Extensions can have dependency to load it from Angular.
   * Extension methods are called with the specified injector context.
   */
  private _injector: Injector | null = null;

  public set injector(value: Injector) {
    if (this._injector !== null)
      throw new Error('environment injector is already set');
    this._injector = value;
  }
  public get injector(): Injector {
    if (this._injector === null)
      throw new Error('environment injector is not set yet');
    return this._injector;
  }
  /**
   * Returns the visible extensions for the given timeline.
   */
  public getVisibleTimelineNavigatorExtensions(
    timeline: TimelineEntry,
  ): DisplayableTimelineNavigatorExtension[] {
    return runInInjectionContext(this.injector, () => {
      return this.timelineNavigatorExtensions
        .filter((extension) => extension.show(timeline))
        .map((extension) => extension.getDisplayable(timeline));
    });
  }

  /**
   * Open files from urlDataLaoderExtensions. Return true when data is loaded.
   */
  public tryOpenDataFromURL(): boolean {
    return runInInjectionContext(this.injector, () => {
      const opened = this.urlDataOenerExtensions.find((extension) =>
        extension.tryOpen(),
      );
      return opened !== undefined;
    });
  }

  /**
   * Call the lifecycle hooks onPageLoad.
   */
  public notifyLifecycleOnPageOpen(page: PageType): void {
    return runInInjectionContext(this.injector, () => {
      this.lifecycleHookExtensions
        .filter((e) => e.onPageLoaded)
        .forEach((e) => e.onPageLoaded!(page));
    });
  }
  /**
   * Call the lifecycle hooks onInspectionDataOpen.
   */
  public notifyLifecycleOnInspectionStart(): void {
    return runInInjectionContext(this.injector, () => {
      this.lifecycleHookExtensions
        .filter((e) => e.onInspectionStart)
        .forEach((e) => e.onInspectionStart!());
    });
  }
  /**
   * Call the lifecycle hooks onInspectionDataOpen.
   */
  public notifyLifecycleOnInspectionDataOpen(
    inspectionData: InspectionData,
    textBufferSource: ReferenceResolverStore,
    rawData: ArrayBuffer,
  ): void {
    return runInInjectionContext(this.injector, () => {
      this.lifecycleHookExtensions
        .filter((e) => e.onInspectionDataOpen)
        .forEach((e) =>
          e.onInspectionDataOpen!(inspectionData, textBufferSource, rawData),
        );
    });
  }
}
