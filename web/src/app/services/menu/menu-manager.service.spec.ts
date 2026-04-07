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

import { TestBed } from '@angular/core/testing';
import { MenuManager, MenuItem, MenuItemType } from './menu-manager.service';

describe('MenuManager', () => {
  let service: MenuManager;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [MenuManager],
    });
    service = TestBed.inject(MenuManager);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should add group and sort by priority', () => {
    service.addGroup('b', 'B', 10);
    service.addGroup('a', 'A', 5);

    const groups = service.groups();
    expect(groups.length).toBe(2);
    expect(groups[0].id).toBe('a');
    expect(groups[1].id).toBe('b');
  });

  it('should add item and sort by priority', () => {
    service.addGroup('a', 'A', 1);

    const item1: MenuItem = { label: 'Item 1', priority: 10 };
    const item2: MenuItem = { label: 'Item 2', priority: 5 };

    service.addItem('a', item1);
    service.addItem('a', item2);

    const groups = service.groups();
    const items = groups[0].items;
    expect(items.length).toBe(2);
    expect(items[0].label).toBe('Item 2');
    expect(items[1].label).toBe('Item 1');
  });

  it('should fallback to default values in ViewModel', () => {
    service.addGroup('a', 'A', 1);

    const item: MenuItem = { priority: 1 }; // label, type などを省略
    service.addItem('a', item);

    const groups = service.groups();
    const vm = groups[0].items[0];

    expect(vm.label).toBe('');
    expect(vm.type).toBe(MenuItemType.Button);
    expect(vm.disabled()).toBe(false);
  });
});
