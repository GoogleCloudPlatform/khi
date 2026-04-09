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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { HeaderV2Component } from './header-v2.component';
import {
  MenuItemType,
  MenuGroupViewModel,
  MenuItemViewModel,
} from '../../services/menu/menu-manager.service';
import { MatIconTestingModule } from '@angular/material/icon/testing';
import { signal } from '@angular/core';
import { By } from '@angular/platform-browser';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { MatMenuHarness } from '@angular/material/menu/testing';

describe('HeaderV2Component', () => {
  let component: HeaderV2Component;
  let fixture: ComponentFixture<HeaderV2Component>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HeaderV2Component, MatIconTestingModule],
    }).compileComponents();

    fixture = TestBed.createComponent(HeaderV2Component);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render version badge', () => {
    fixture.componentRef.setInput('version', '1.2.3');
    fixture.detectChanges();

    const badgeEl = fixture.debugElement.query(By.css('.version-badge'));
    expect(badgeEl).toBeTruthy();
    expect(badgeEl.nativeElement.textContent).toContain('v1.2.3');
  });

  it('should render viewer mode badge when viewerMode is true', () => {
    fixture.componentRef.setInput('viewerMode', true);
    fixture.detectChanges();

    const badgeEls = fixture.debugElement.queryAll(By.css('.version-badge'));
    const viewerBadge = badgeEls.find((el) =>
      el.nativeElement.classList.contains('viewer'),
    );

    expect(viewerBadge).toBeTruthy();
    expect(viewerBadge!.nativeElement.textContent).toContain('Viewer Mode');
  });

  it('should emit menuItemClick when a menu item is clicked', async () => {
    const mockItem: MenuItemViewModel = {
      id: 'test-item',
      label: 'Test Item',
      type: MenuItemType.Button,
      icon: '',
      tooltip: '',
      action: () => {},
      checked: signal(false),
      disabled: signal(false),
      priority: 1,
    };

    const mockGroups: MenuGroupViewModel[] = [
      {
        id: 'test',
        label: 'Test Group',
        priority: 1,
        icon: '',
        items: [mockItem],
      },
    ];

    fixture.componentRef.setInput('menuGroups', mockGroups);
    fixture.detectChanges();

    let emittedItem: MenuItemViewModel | null = null;
    component.menuItemClick.subscribe((item) => (emittedItem = item));

    const loader = TestbedHarnessEnvironment.loader(fixture);
    const menu = await loader.getHarness(MatMenuHarness);
    await menu.open();

    const items = await menu.getItems();
    expect(items.length).toBe(1);

    await items[0].click();

    expect(emittedItem as MenuItemViewModel | null).toBe(mockItem);
  });
});
