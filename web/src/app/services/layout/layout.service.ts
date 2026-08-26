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
  ComponentRef,
  inject,
  Injectable,
  OnDestroy,
  signal,
  Type,
  ViewContainerRef,
  WritableSignal,
} from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import {
  GoldenLayout,
  ComponentContainer,
  LayoutConfig,
  Tab,
} from 'golden-layout';
import { TimelineSmartComponent } from 'src/app/timeline/timeline-smart.component';
import { LogSmartComponent } from 'src/app/log/log-smart.component';
import { DiffSmartComponent } from 'src/app/diff/diff-smart.component';
import { GraphSmartComponent } from 'src/app/graph/graph-smart.component';
import {
  MenuManager,
  MenuItemType,
} from 'src/app/services/menu/menu-manager.service';
import { StyleOverrideSmartComponent } from 'src/app/dialogs/style-override/style-override-smart.component';
import { isEventFromOverlay } from 'src/app/common/dom-util';

/**
 * Type identifiers for components managed by LayoutService.
 */
export enum LayoutComponentType {
  Timeline = 'timeline',
  Log = 'log',
  History = 'history',
  Graph = 'graph',
}

/**
 * Configuration for registering a GoldenLayout component.
 */
interface ComponentRegistrationConfig {
  readonly type: LayoutComponentType;
  readonly componentClass: Type<unknown>;
  readonly tabIcon: string;
  readonly disabledSignal?: WritableSignal<boolean>;
}

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

  /** Flag indicating whether a layout switch is in progress. */
  private isSwitchingLayout = false;

  /** Pool of detached component instances waiting to be reused. */
  private readonly componentPool = new Map<
    LayoutComponentType,
    ComponentRef<unknown>[]
  >();

  /** Active component instances mapped by their GoldenLayout container. */
  private readonly activeComponentRefs = new Map<
    ComponentContainer,
    ComponentRef<unknown>
  >();

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
              componentType: LayoutComponentType.Timeline,
              title: 'Timeline',
              size: '70%',
            },
            {
              type: 'component',
              componentType: LayoutComponentType.Log,
              title: 'Logs',
              size: '15%',
            },
            {
              type: 'component',
              componentType: LayoutComponentType.History,
              title: 'History',
              size: '15%',
            },
          ],
        },
      ],
    },
  };

  /** The layout configuration for state analysis. */
  private readonly stateAnalysisLayout: LayoutConfig = {
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
              componentType: LayoutComponentType.Timeline,
              title: 'Timeline',
              size: '60%',
            },
            {
              type: 'component',
              componentType: LayoutComponentType.History,
              title: 'History',
              size: '40%',
            },
          ],
        },
      ],
    },
  };

  /** The layout configuration for topology analysis. */
  private readonly topologyAnalysisLayout: LayoutConfig = {
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
              componentType: LayoutComponentType.Timeline,
              title: 'Timeline',
              size: '50%',
            },
            {
              type: 'component',
              componentType: LayoutComponentType.Graph,
              title: 'Graph',
              size: '50%',
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
    window.addEventListener('keydown', this.handleKeyDown);
  }

  /**
   * Registers components to GoldenLayout.
   */
  private registerComponents(): void {
    this.registerComponent({
      type: LayoutComponentType.Timeline,
      componentClass: TimelineSmartComponent,
      tabIcon: 'view_timeline',
    });

    this.registerComponent({
      type: LayoutComponentType.Log,
      componentClass: LogSmartComponent,
      tabIcon: 'cards_stack',
      disabledSignal: this.disableCreateLogPane,
    });

    this.registerComponent({
      type: LayoutComponentType.History,
      componentClass: DiffSmartComponent,
      tabIcon: 'deployed_code_history',
      disabledSignal: this.disableCreateDiffPane,
    });

    this.registerComponent({
      type: LayoutComponentType.Graph,
      componentClass: GraphSmartComponent,
      tabIcon: 'family_history',
      disabledSignal: this.disableCreateGraphPane,
    });
  }

  /**
   * Registers a single component type with pooling and caching support.
   */
  private registerComponent(config: ComponentRegistrationConfig): void {
    this.goldenLayout.registerComponentFactoryFunction(
      config.type,
      (container: ComponentContainer) => {
        const pool = this.componentPool.get(config.type);
        const componentRef =
          pool && pool.length > 0
            ? pool.pop()!
            : this.viewContainerRef.createComponent(config.componentClass);

        componentRef.changeDetectorRef.reattach();

        this.activeComponentRefs.set(container, componentRef);
        container.element.appendChild(componentRef.location.nativeElement);
        this.addIconToTab(container, config.tabIcon);
        config.disabledSignal?.set(true);

        container.on('destroy', () => {
          this.activeComponentRefs.delete(container);
          if (this.isSwitchingLayout) {
            componentRef.location.nativeElement.remove();
            componentRef.changeDetectorRef.detach();
            let currentPool = this.componentPool.get(config.type);
            if (!currentPool) {
              currentPool = [];
              this.componentPool.set(config.type, currentPool);
            }
            currentPool.push(componentRef);
          } else {
            componentRef.destroy();
          }
          config.disabledSignal?.set(false);
        });
      },
    );
  }

  /**
   * Adds icon to tab.
   */
  private addIconToTab(container: ComponentContainer, iconName: string): void {
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
   * Executes a layout switch while preserving cached components.
   */
  private executeLayoutSwitch(layoutConfig: LayoutConfig): void {
    this.isSwitchingLayout = true;
    try {
      this.goldenLayout.loadLayout(layoutConfig);

      const timelinePool = this.componentPool.get(LayoutComponentType.Timeline);
      if (timelinePool) {
        while (timelinePool.length > 0) {
          const surplus = timelinePool.pop();
          surplus?.destroy();
        }
      }
    } finally {
      this.isSwitchingLayout = false;
    }
  }

  /**
   * Loads default layout configuration.
   */
  public loadDefaultLayout(): void {
    this.executeLayoutSwitch(this.defaultLayout);
  }

  /**
   * Loads state analysis layout configuration.
   */
  public loadStateAnalysisLayout(): void {
    this.executeLayoutSwitch(this.stateAnalysisLayout);
  }

  /**
   * Loads topology analysis layout configuration.
   */
  public loadTopologyAnalysisLayout(): void {
    this.executeLayoutSwitch(this.topologyAnalysisLayout);
  }

  /**
   * Handles keyboard shortcuts for switching layout mode.
   */
  private readonly handleKeyDown = (event: KeyboardEvent) => {
    if (isEventFromOverlay(event)) {
      return;
    }
    if ((event.ctrlKey || event.metaKey) && !event.altKey && !event.shiftKey) {
      if (event.key === '1') {
        event.preventDefault();
        this.loadDefaultLayout();
      } else if (event.key === '2') {
        event.preventDefault();
        this.loadStateAnalysisLayout();
      } else if (event.key === '3') {
        event.preventDefault();
        this.loadTopologyAnalysisLayout();
      }
    }
  };

  private setupMenu() {
    this.menuManager.addGroup('view', 'View', 2, 'dashboard_customize');
    this.menuManager.addItem('view', {
      id: 'open-timeline',
      label: 'Open timeline',
      type: MenuItemType.Button,
      icon: 'timeline',
      priority: 1,
      action: () => {
        this.addPane(LayoutComponentType.Timeline, 'Timeline');
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
        this.addPane(LayoutComponentType.Log, 'Logs');
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
        this.addPane(LayoutComponentType.History, 'History');
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
        this.addPane(LayoutComponentType.Graph, 'Graph');
      },
    });
    this.menuManager.addItem('view', {
      id: 'view-separator',
      type: MenuItemType.Separator,
      priority: 5,
    });
    this.menuManager.addItem('view', {
      id: 'default-layout',
      label: 'Default layout',
      type: MenuItemType.Button,
      icon: 'dashboard',
      shortcut: 'Ctrl+1',
      priority: 6,
      action: () => {
        this.loadDefaultLayout();
      },
    });
    this.menuManager.addItem('view', {
      id: 'state-view-layout',
      label: 'State view layout',
      type: MenuItemType.Button,
      icon: 'conditions',
      shortcut: 'Ctrl+2',
      priority: 7,
      action: () => {
        this.loadStateAnalysisLayout();
      },
    });
    this.menuManager.addItem('view', {
      id: 'topology-view-layout',
      label: 'Topology view layout',
      type: MenuItemType.Button,
      icon: 'hub',
      shortcut: 'Ctrl+3',
      priority: 8,
      action: () => {
        this.loadTopologyAnalysisLayout();
      },
    });
    this.menuManager.addItem('view', {
      id: 'view-separator-2',
      type: MenuItemType.Separator,
      priority: 9,
    });
    this.menuManager.addItem('view', {
      id: 'style-override',
      label: 'Style override Settings',
      type: MenuItemType.Button,
      icon: 'palette',
      priority: 10,
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
  private addPane(componentType: LayoutComponentType, title: string): void {
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

  public ngOnDestroy(): void {
    window.removeEventListener('keydown', this.handleKeyDown);
    this.resizeObserver?.disconnect();
    this.goldenLayout?.destroy();

    for (const ref of this.activeComponentRefs.values()) {
      ref.destroy();
    }
    this.activeComponentRefs.clear();

    for (const pool of this.componentPool.values()) {
      for (const ref of pool) {
        ref.destroy();
      }
    }
    this.componentPool.clear();
  }
}
