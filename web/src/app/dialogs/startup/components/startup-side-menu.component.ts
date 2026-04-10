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
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { SidebarLink } from '../types/startup-side-menu.types';

/**
 * Sidebar component for the startup dialog.
 * Displays the application logo, title, main action buttons, and footer links.
 */
@Component({
  selector: 'khi-startup-side-menu',
  imports: [
    MatTooltipModule,
    MatIconModule,
    MatButtonModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './startup-side-menu.component.html',
  styleUrls: ['./startup-side-menu.component.scss'],
})
export class StartupSideMenuComponent {
  /** The current version of the application to be displayed. */
  public readonly version = input.required<string>();

  /** The list of links to be displayed in the footer. */
  public readonly links = input.required<SidebarLink[]>();

  /** Emitted when the user clicks the 'New Investigation' button. */
  public readonly newInvestigation = output<void>();

  /** Emitted when the user clicks the 'Open .khi file' button. */
  public readonly openKhiFile = output<void>();
}
