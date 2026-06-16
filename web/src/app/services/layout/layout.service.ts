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

import {
  inject,
  Injectable,
  OnDestroy,
  signal,
  ViewContainerRef,
} from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import {
  GoldenLayout,
  ComponentContainer,
  LayoutConfig,
  Tab,
} from 'golden-layout';
import { TimelineSmartComponent } from '../../timeline/timeline-smart.component';
import { LogSmartComponent } from '../../log/log-smart.component';
import { DiffSmartComponent } from '../../diff/diff-smart.component';
import { GraphSmartComponent } from '../../graph/graph-smart.component';
import { MenuManager, MenuItemType } from '../menu/menu-manager.service';
import { StyleOverrideSmartComponent } from 'src/app/dialogs/style-override/style-override-smart.component';

/**
 * LayoutService manages the GoldenLayout instance and component registration.
 */
@Injectable()
export class LayoutService implements OnDestroy {
  private readonly menuManager = inject(MenuManager);
  private readonly dialog = inject(MatDialog);
  private styleOverrideDialogRef: MatDialogRef<StyleOverrideSmartComponent> | null =
    null;
  /** The GoldenLayout instance. */
  private goldenLayout!: GoldenLayout;

  /** ViewContainerRef for creating Angular components dynamically. */
  private viewContainerRef!: ViewContainerRef;

  /** ResizeObserver to track container size changes. */
  private resizeObserver?: ResizeObserver;

  private readonly disableCreateDiffPane = signal(false);

  private readonly disableCreateLogPane = signal(false);

  private readonly disableCreateGraphPane = signal(false);

  /** The default layout configuration used if no saved state is found. */
  private readonly defaultLayout: LayoutConfig = {
    settings: {
      showPopoutIcon: false,
    },
    dimensions: {
      borderWidth: 5,
    },
    root: {
      type: 'column',
      content: [
        {
          type: 'row',
          content: [
            {
              type: 'component',
              componentType: 'timeline',
              title: 'Timeline',
              size: '70%',
            },
            {
              type: 'component',
              componentType: 'log',
              title: 'Logs',
              size: '15%',
            },
            {
              type: 'component',
              componentType: 'history',
              title: 'History',
              size: '15%',
            },
          ],
        },
      ],
    },
  };

  /**
   * Initializes GoldenLayout.
   */
  public init(hostElement: HTMLElement, vcr: ViewContainerRef) {
    this.viewContainerRef = vcr;
    this.goldenLayout = new GoldenLayout(hostElement);

    this.registerComponents();

    this.resizeObserver = new ResizeObserver(() => {
      this.goldenLayout.setSize(
        hostElement.clientWidth,
        hostElement.clientHeight,
      );
    });
    this.resizeObserver.observe(hostElement);
    this.setupMenu();
  }

  /**
   * Registers components to GoldenLayout.
   */
  private registerComponents() {
    this.goldenLayout.registerComponentFactoryFunction(
      'timeline',
      (container: ComponentContainer) => {
        const componentRef = this.viewContainerRef.createComponent(
          TimelineSmartComponent,
        );
        container.element.appendChild(componentRef.location.nativeElement);
        this.addIconToTab(container, 'view_timeline');
        container.on('destroy', () => componentRef.destroy());
      },
    );

    this.goldenLayout.registerComponentFactoryFunction(
      'log',
      (container: ComponentContainer) => {
        const componentRef =
          this.viewContainerRef.createComponent(LogSmartComponent);
        container.element.appendChild(componentRef.location.nativeElement);
        this.addIconToTab(container, 'cards_stack');
        container.on('destroy', () => {
          componentRef.destroy();
          this.disableCreateLogPane.set(false);
        });
        this.disableCreateLogPane.set(true);
      },
    );

    this.goldenLayout.registerComponentFactoryFunction(
      'history',
      (container: ComponentContainer) => {
        const componentRef =
          this.viewContainerRef.createComponent(DiffSmartComponent);
        container.element.appendChild(componentRef.location.nativeElement);
        this.addIconToTab(container, 'deployed_code_history');
        container.on('destroy', () => {
          componentRef.destroy();
          this.disableCreateDiffPane.set(false);
        });
        this.disableCreateDiffPane.set(true);
      },
    );

    this.goldenLayout.registerComponentFactoryFunction(
      'graph',
      (container: ComponentContainer) => {
        const componentRef =
          this.viewContainerRef.createComponent(GraphSmartComponent);
        container.element.appendChild(componentRef.location.nativeElement);
        this.addIconToTab(container, 'family_history');
        container.on('destroy', () => {
          componentRef.destroy();
          this.disableCreateGraphPane.set(false);
        });
        this.disableCreateGraphPane.set(true);
      },
    );
  }

  /**
   * Adds icon to tab.
   */
  private addIconToTab(container: ComponentContainer, iconName: string) {
    container.on('tab', (tab: Tab) => {
      const iconSpan = document.createElement('span');
      iconSpan.className = 'material-symbols-outlined khi-tab-icon';
      iconSpan.innerText = iconName;

      const titleEl = tab.titleElement as HTMLElement;
      if (titleEl) {
        titleEl.insertBefore(iconSpan, titleEl.firstChild);
      }
    });
  }

  /**
   * Loads default layout configuration.
   */
  public loadDefaultLayout() {
    this.goldenLayout.loadLayout(this.defaultLayout);
  }

  private setupMenu() {
    this.menuManager.addGroup('view', 'View', 2, 'dashboard_customize');
    this.menuManager.addItem('view', {
      id: 'open-timeline',
      label: 'Open timeline',
      type: MenuItemType.Button,
      icon: 'timeline',
      priority: 1,
      action: () => {
        this.addPane('timeline', 'Timeline');
      },
    });
    this.menuManager.addItem('view', {
      id: 'open-log',
      label: 'Open log view',
      type: MenuItemType.Button,
      icon: 'cards_stack',
      priority: 2,
      disabled: this.disableCreateLogPane,
      action: () => {
        this.addPane('log', 'Logs');
      },
    });
    this.menuManager.addItem('view', {
      id: 'open-history',
      label: 'Open history view',
      type: MenuItemType.Button,
      icon: 'difference',
      disabled: this.disableCreateDiffPane,
      priority: 3,
      action: () => {
        this.addPane('history', 'History');
      },
    });
    this.menuManager.addItem('view', {
      id: 'open-graph',
      label: 'Open graph view',
      type: MenuItemType.Button,
      icon: 'family_history',
      disabled: this.disableCreateGraphPane,
      priority: 4,
      action: () => {
        this.addPane('graph', 'Graph');
      },
    });
    this.menuManager.addItem('view', {
      id: 'view-separator',
      type: MenuItemType.Separator,
      priority: 5,
    });
    this.menuManager.addItem('view', {
      id: 'reset-layout',
      label: 'Reset layout',
      type: MenuItemType.Button,
      icon: 'refresh',
      priority: 6,
      action: () => {
        this.loadDefaultLayout();
      },
    });
    this.menuManager.addItem('view', {
      id: 'style-override',
      label: 'Style override Settings',
      type: MenuItemType.Button,
      icon: 'palette',
      priority: 7,
      action: () => {
        this.openStyleOverrideDialog();
      },
    });
  }

  private openStyleOverrideDialog() {
    if (this.styleOverrideDialogRef) {
      return;
    }
    this.styleOverrideDialogRef = this.dialog.open(
      StyleOverrideSmartComponent,
      {
        width: '400px',
        height: '100vh',
        maxHeight: '100vh',
        position: {
          right: '0px',
          top: '0px',
        },
        hasBackdrop: false,
      },
    );
    this.styleOverrideDialogRef.afterClosed().subscribe(() => {
      this.styleOverrideDialogRef = null;
    });
  }

  /**
   * Adds a new pane to the layout.
   */
  private addPane(componentType: string, title: string) {
    try {
      this.goldenLayout.addItem({
        type: 'component',
        componentType: componentType,
        title: title,
      });
    } catch (e) {
      console.error(
        `[LayoutService] Failed to add pane "${componentType}":`,
        e,
      );
    }
  }

  ngOnDestroy(): void {
    this.resizeObserver?.disconnect();
    this.goldenLayout?.destroy();
  }
}
