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

import { Meta, StoryObj } from '@storybook/angular';
import { HeaderV2Component } from './header-v2.component';
import {
  MenuGroupViewModel,
  MenuItemType,
} from '../../services/menu/menu-manager.service';
import { signal } from '@angular/core';
import { BackendConnectionStatus } from '../../services/api/backend-sync-interface';

const mockMenuGroups: MenuGroupViewModel[] = [
  {
    id: 'file',
    label: 'File',
    priority: 1,
    icon: '',
    items: [
      {
        id: 'open',
        label: 'Open',
        type: MenuItemType.Button,
        icon: 'folder_open',
        tooltip: '',
        action: () => {},
        checked: signal(false),
        disabled: signal(false),
        priority: 1,
      },
      {
        id: 'save',
        label: 'Save',
        type: MenuItemType.Button,
        icon: 'save',
        tooltip: '',
        action: () => {},
        checked: signal(false),
        disabled: signal(false),
        priority: 2,
      },
      {
        id: 'separator-1',
        label: '',
        type: MenuItemType.Separator,
        icon: '',
        tooltip: '',
        action: () => {},
        checked: signal(false),
        disabled: signal(false),
        priority: 3,
      },
      {
        id: 'exit',
        label: 'Exit',
        type: MenuItemType.Button,
        icon: 'close',
        tooltip: '',
        action: () => {},
        checked: signal(false),
        disabled: signal(false),
        priority: 4,
      },
    ],
  },
  {
    id: 'view',
    label: 'View',
    priority: 2,
    icon: '',
    items: [
      {
        id: 'toggle-sidebar',
        label: 'Toggle Sidebar',
        type: MenuItemType.Checkbox,
        icon: '',
        tooltip: '',
        action: () => {},
        checked: signal(true),
        disabled: signal(false),
        priority: 1,
      },
      {
        id: 'zoom-in',
        label: 'Zoom In',
        type: MenuItemType.Button,
        icon: 'zoom_in',
        tooltip: '',
        action: () => {},
        checked: signal(false),
        disabled: signal(false),
        priority: 2,
      },
    ],
  },
];

const meta: Meta<HeaderV2Component> = {
  title: 'HeaderV2',
  component: HeaderV2Component,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<HeaderV2Component>;

export const Default: Story = {
  args: {
    version: '1.2.3',
    viewerMode: false,
    menuGroups: mockMenuGroups,
    serverStatus: BackendConnectionStatus.Connecting,
    serverMemory: '120MB',
    serverMaxMemory: '512MB',
    sessionId: '1',
  },
};

export const Viewer: Story = {
  args: {
    version: '1.2.3',
    viewerMode: true,
    menuGroups: mockMenuGroups,
    serverStatus: BackendConnectionStatus.Connected,
    serverMemory: '120MB',
    serverMaxMemory: '512MB',
    sessionId: '1',
  },
};

export const OnlyCurrentMemory: Story = {
  args: {
    version: '1.2.3',
    viewerMode: false,
    menuGroups: mockMenuGroups,
    serverStatus: BackendConnectionStatus.Connected,
    serverMemory: '120MB',
    serverMaxMemory: '',
    sessionId: '1',
  },
};

export const Disconnected: Story = {
  args: {
    version: '1.2.3',
    viewerMode: false,
    menuGroups: mockMenuGroups,
    serverStatus: BackendConnectionStatus.Disconnected,
    serverMemory: '',
    serverMaxMemory: '',
    sessionId: '',
  },
};
