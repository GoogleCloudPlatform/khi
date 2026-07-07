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

/**
 * A directive to display a dynamic component as a tooltip.
 */
@Directive({
  selector: '[khiDynamicTooltip]',
})
export class DynamicTooltipDirective implements OnDestroy {
  /** The annotation definition containing the component to render and its inputs. */
  readonly khiDynamicTooltip = input<YamlFieldAnnotation | undefined>();

  private _overlayRef: OverlayRef | null = null;
  private readonly _overlay = inject(Overlay);
  private readonly _overlayPositionBuilder = inject(OverlayPositionBuilder);
  private readonly _elementRef = inject(ElementRef);

  @HostListener('mouseenter')
  show() {
    const annotation = this.khiDynamicTooltip();
    if (!annotation) {
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
      ]);

    this._overlayRef = this._overlay.create({
      positionStrategy,
      scrollStrategy: this._overlay.scrollStrategies.close(),
    });

    const portal = new ComponentPortal(annotation.component);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const componentRef: ComponentRef<any> = this._overlayRef.attach(portal);

    if (annotation.inputs) {
      for (const [key, value] of Object.entries(annotation.inputs)) {
        componentRef.setInput(key, value);
      }
    }

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
