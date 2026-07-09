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
  Directive,
  ElementRef,
  HostListener,
  inject,
  input,
  ComponentRef,
  OnDestroy,
} from '@angular/core';
import {
  Overlay,
  OverlayRef,
  OverlayPositionBuilder,
} from '@angular/cdk/overlay';
import { ComponentPortal } from '@angular/cdk/portal';
import { YamlFieldAnnotation } from 'src/app/shared/components/yaml-viewer/yaml-annotation';
import { DynamicTooltipListContainerComponent } from 'src/app/shared/components/yaml-viewer/components/dynamic-tooltip-list-container.component';

/**
 * A directive to display a dynamic component as a tooltip.
 */
@Directive({
  selector: '[khiDynamicTooltip]',
})
export class DynamicTooltipDirective implements OnDestroy {
  /** The annotations definition containing the components to render and their inputs. */
  readonly khiDynamicTooltip = input<YamlFieldAnnotation[] | undefined>();

  private _overlayRef: OverlayRef | null = null;
  private readonly _overlay = inject(Overlay);
  private readonly _overlayPositionBuilder = inject(OverlayPositionBuilder);
  private readonly _elementRef = inject(ElementRef);

  @HostListener('mouseenter')
  show() {
    const annotations = this.khiDynamicTooltip();
    if (!annotations || annotations.length === 0) {
      return;
    }

    if (this._overlayRef) {
      return; // Already open
    }

    const positionStrategy = this._overlayPositionBuilder
      .flexibleConnectedTo(this._elementRef.nativeElement)
      .withFlexibleDimensions(false)
      .withViewportMargin(8)
      .withPositions([
        {
          originX: 'start',
          originY: 'center',
          overlayX: 'end',
          overlayY: 'center',
          offsetX: -20,
        },
        {
          originX: 'center',
          originY: 'bottom',
          overlayX: 'center',
          overlayY: 'top',
          offsetY: 20,
        },
      ]);

    this._overlayRef = this._overlay.create({
      positionStrategy,
      scrollStrategy: this._overlay.scrollStrategies.close(),
    });

    const portal = new ComponentPortal(DynamicTooltipListContainerComponent);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const componentRef: ComponentRef<any> = this._overlayRef.attach(portal);

    componentRef.setInput('annotations', annotations);

    // Force change detection so that the component renders its content and acquires actual dimensions.
    // This is required before updating the position, otherwise the overlay is positioned assuming 0x0 size.
    componentRef.changeDetectorRef.detectChanges();
    this._overlayRef.updatePosition();
  }

  @HostListener('mouseleave')
  hide() {
    if (this._overlayRef) {
      this._overlayRef.detach();
      this._overlayRef.dispose();
      this._overlayRef = null;
    }
  }

  ngOnDestroy() {
    this.hide();
  }
}
