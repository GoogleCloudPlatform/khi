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

import { Component, inject, computed } from '@angular/core';
import { BACKEND_API } from '../services/api/backend-api-interface';
import {
  MenuManager,
  MenuItemViewModel,
  MenuItemType,
} from '../services/menu/menu-manager.service';
import { VERSION } from 'src/environments/version';
import { map } from 'rxjs';
import { toSignal } from '@angular/core/rxjs-interop';
import { HeaderV2Component } from './components/header-v2.component';
import { BACKEND_SYNC } from '../services/api/backend-sync.service';
import { BackendSyncService } from '../services/api/backend-sync-interface';
import { WindowConnectorService } from '../services/frame-connection/window-connector.service';

/**
 * Smart component for Header version 2.
 * Manages dependencies and provides data to the dumb component.
 */
@Component({
  selector: 'khi-header-v2-smart',
  imports: [HeaderV2Component],
  templateUrl: './header-v2-smart.component.html',
  styleUrls: ['./header-v2-smart.component.scss'],
})
export class HeaderV2SmartComponent {
  /** Menu manager service. */
  protected readonly menuManager = inject(MenuManager);

  private readonly backendAPI = inject(BACKEND_API);
  private readonly backendSync = inject<BackendSyncService>(BACKEND_SYNC);
  private readonly windowConnector = inject(WindowConnectorService);

  /** Divisor to convert bytes to GB. */
  private readonly BYTES_TO_GB = 1024 * 1024 * 1024;

  /** Current version of the application. */
  protected readonly version = VERSION;

  /** Whether the application is in viewer mode. */
  protected readonly viewerMode = toSignal(
    this.backendAPI.getConfig().pipe(map((config) => config.viewerMode)),
    { initialValue: false },
  );

  /** Session ID for multi-window identification. */
  protected readonly sessionId = toSignal(
    this.windowConnector.sessionEstablished.pipe(
      map(() => String(this.windowConnector.sessionId)),
    ),
    { initialValue: '' },
  );

  /** Server statistics. */
  private readonly serverStat = computed(
    () => this.backendSync.tasks.value()?.serverStat,
  );

  /** Server connection status from BackendSyncService. */
  protected readonly serverStatus = computed(() =>
    this.backendSync.connectionStatus(),
  );

  /** Server current memory usage string. */
  protected readonly serverMemory = computed(() => {
    const stat = this.serverStat();
    if (!stat) return '';
    return (stat.currentMemoryUsage / this.BYTES_TO_GB).toFixed(2);
  });

  /** Server maximum memory limit string. */
  protected readonly serverMaxMemory = computed(() => {
    const stat = this.serverStat();
    if (!stat || stat.totalMemory === 0) return '';
    return (stat.totalMemory / this.BYTES_TO_GB).toFixed(2);
  });

  /**
   * Handles menu item click events from the dumb component.
   * Resolves actions and toggles state.
   */
  protected handleMenuItemClick(item: MenuItemViewModel) {
    if (item.type === MenuItemType.Checkbox) {
      const currentState = item.checked();
      item.action(!currentState);
    } else {
      item.action();
    }
  }
}
