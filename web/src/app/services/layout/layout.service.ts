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

import { Injectable, OnDestroy, ViewContainerRef } from '@angular/core';
import {
  GoldenLayout,
  ComponentContainer,
  LayoutConfig,
  Tab,
} from 'golden-layout';
import { TimelineSmartComponent } from '../../timeline/timeline-smart.component';
import { LogSmartComponent } from '../../log/log-smart.component';
import { DiffSmartComponent } from '../../diff/diff-smart.component';

/**
 * LayoutService manages the GoldenLayout instance and component registration.
 */
@Injectable()
export class LayoutService implements OnDestroy {
  /** The GoldenLayout instance. */
  private goldenLayout!: GoldenLayout;

  /** ViewContainerRef for creating Angular components dynamically. */
  private viewContainerRef!: ViewContainerRef;

  /** ResizeObserver to track container size changes. */
  private resizeObserver?: ResizeObserver;

  /** The default layout configuration used if no saved state is found. */
  private readonly defaultLayout: LayoutConfig = {
    settings: {
      showPopoutIcon: false,
    },
    dimensions: {
      borderWidth: 5,
    },
    root: {
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
          componentType: 'diff',
          title: 'History',
          size: '15%',
        },
      ],
    },
  };

  /**
   * Initialize GoldenLayout.
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
  }

  /**
   * Register components to GoldenLayout.
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
        container.on('destroy', () => componentRef.destroy());
      },
    );

    this.goldenLayout.registerComponentFactoryFunction(
      'diff',
      (container: ComponentContainer) => {
        const componentRef =
          this.viewContainerRef.createComponent(DiffSmartComponent);
        container.element.appendChild(componentRef.location.nativeElement);
        this.addIconToTab(container, 'deployed_code_history');
        container.on('destroy', () => componentRef.destroy());
      },
    );
  }

  /**
   * Add icon to tab.
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
   * Load default layout configuration.
   */
  public loadDefaultLayout() {
    this.goldenLayout.loadLayout(this.defaultLayout);
  }

  ngOnDestroy(): void {
    this.resizeObserver?.disconnect();
    this.goldenLayout?.destroy();
  }
}
