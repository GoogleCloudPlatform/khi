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

import { Component, input, output } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatMenuModule } from '@angular/material/menu';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { MatTooltipModule } from '@angular/material/tooltip';
import {
  MenuGroupViewModel,
  MenuItemViewModel,
  MenuItemType,
} from '../../services/menu/menu-manager.service';
import { KHIIconRegistrationModule } from '../../shared/module/icon-registration.module';
import { BackendConnectionStatus } from '../../services/api/backend-sync-interface';

/**
 * Header component version 2 (Dumb component).
 */

@Component({
  selector: 'khi-header-v2',
  templateUrl: './header-v2.component.html',
  styleUrls: ['./header-v2.component.scss'],
  imports: [
    MatButtonModule,
    MatMenuModule,
    MatIconModule,
    MatDividerModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
})
export class HeaderV2Component {
  /** Expose MenuItemType to template. */
  protected readonly MenuItemType = MenuItemType;

  /** Expose BackendConnectionStatus to template. */
  protected readonly BackendConnectionStatus = BackendConnectionStatus;

  /** Current version of the application. */
  readonly version = input<string>('');

  /** Whether the application is in viewer mode. */
  readonly viewerMode = input<boolean>(false);

  /** Menu groups to display. */
  readonly menuGroups = input<MenuGroupViewModel[]>([]);

  /** Status of the server connection. */
  readonly serverStatus = input<BackendConnectionStatus>(
    BackendConnectionStatus.Connecting,
  );

  /** Server current memory usage string. */
  readonly serverMemory = input<string>('');

  /** Server maximum memory limit string. */
  readonly serverMaxMemory = input<string>('');

  /** Session ID for multi-window identification. */
  readonly sessionId = input<string>('');

  /** Event emitted when a menu item is clicked. */
  readonly menuItemClick = output<MenuItemViewModel>();
}
