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

import { signal, computed, Signal, Injectable } from '@angular/core';

// Interfaces for users.
/**
 * Defines the type of a menu item.
 */
export enum MenuItemType {
  Button = 'button',
  Checkbox = 'checkbox',
  Separator = 'separator',
}

/**
 * Represents a menu item configuration provided by users.
 */
export interface MenuItem {
  /** Label for the menu item. Not required if type is 'separator'. */
  label?: string;
  /** Type of the menu item. Default is MenuItemType.Button. */
  type?: MenuItemType;
  /** Icon for the menu item. */
  icon?: string;
  /** Tooltip text. */
  tooltip?: string;
  /** Action to perform on click or change. */
  action?: (state?: boolean) => void;
  /** Signal for the checked state (only for Checkbox). */
  checked?: Signal<boolean>;
  /** Signal for the disabled state. */
  disabled?: Signal<boolean>;
  /** Priority for display order. */
  priority: number;
}

/**
 * Represents a menu group configuration.
 */
export interface MenuGroup {
  /** Identifier for the group. */
  id: string;
  /** Label for the group. */
  label: string;
  /** Icon for the group. */
  icon?: string;
  /** Priority for display order. */
  priority: number;
  /** Items belonging to this group. */
  items: MenuItem[];
}

// ViewModel for components like HeaderV2.

/**
 * Represents a normalized menu item for display in components.
 */
export interface MenuItemViewModel {
  label: string;
  type: MenuItemType;
  icon: string;
  tooltip: string;
  action: (state?: boolean) => void;
  checked: Signal<boolean>;
  disabled: Signal<boolean>;
  priority: number;
}

/**
 * Represents a view model for a menu group, containing normalized items.
 */
export interface MenuGroupViewModel {
  id: string;
  label: string;
  icon: string;
  priority: number;
  items: MenuItemViewModel[];
}

/**
 * Creates a MenuItemViewModel from a MenuItem, filling in default values.
 * @param item The source menu item.
 * @returns The constructed view model.
 */
export function createMenuItemViewModel(item: MenuItem): MenuItemViewModel {
  return {
    label: item.label || '',
    type: item.type || MenuItemType.Button,
    icon: item.icon || '',
    tooltip: item.tooltip || '',
    action: item.action || (() => {}),
    checked: item.checked || signal(false),
    disabled: item.disabled || signal(false),
    priority: item.priority,
  };
}

/**
 * Creates a MenuGroupViewModel from a MenuGroup.
 * @param group The source menu group.
 * @returns The constructed view model.
 */
export function createMenuGroupViewModel(group: MenuGroup): MenuGroupViewModel {
  return {
    id: group.id,
    label: group.label,
    icon: group.icon || '',
    priority: group.priority,
    items: group.items.map((item) => createMenuItemViewModel(item)),
  };
}

/**
 * Manages the application menu state.
 * Centralizes the registration and retrieval of menu groups and items.
 */
@Injectable()
export class MenuManager {
  private readonly groupsSignal = signal<Map<string, MenuGroup>>(new Map());

  /**
   * Provides the list of groups sorted by priority and converted to ViewModel.
   */
  readonly groups = computed<MenuGroupViewModel[]>(() => {
    return Array.from(this.groupsSignal().values())
      .sort((a, b) => a.priority - b.priority)
      .map((group) => createMenuGroupViewModel(group));
  });

  /**
   * Adds a root menu (group).
   * @param id Identifier for the group.
   * @param label Display label for the group.
   * @param priority Priority for display order.
   * @param icon Optional icon for the group.
   */
  addGroup(id: string, label: string, priority: number, icon?: string): void {
    const current = this.groupsSignal();
    if (!current.has(id)) {
      current.set(id, { id, label, priority, icon, items: [] });
      this.groupsSignal.set(new Map(current));
    }
  }

  /**
   * Adds a menu item to the specified group.
   * After adding, items are sorted by priority.
   * @param groupId The ID of the group to add the item to.
   * @param item The menu item to add.
   */
  addItem(groupId: string, item: MenuItem): void {
    const current = this.groupsSignal();
    const group = current.get(groupId);
    if (!group) {
      console.warn(`[MenuManager] MenuGroup "${groupId}" not found.`);
      return;
    }
    group.items.push(item);
    group.items.sort((a, b) => a.priority - b.priority);
    this.groupsSignal.set(new Map(current));
  }
}
