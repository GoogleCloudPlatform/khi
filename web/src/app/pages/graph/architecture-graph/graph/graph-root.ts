/**
 * Copyright 2024 Google LLC
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
  GraphObjectWithElementType,
  Size,
  ZeroSize2,
  SameSize2,
  GraphObject,
} from './base/base-containers';

/**
 * Root <svg> element used in the graph
 */
export class GraphRoot extends GraphObjectWithElementType<SVGSVGElement> {
  /**
   * Registers a GraphObject under a specific string identifier.
   *
   * @param obj - The graph object to register.
   * @param id - The unique identifier.
   */
  public registerGraphObjectWithId(obj: GraphObject, id: string): void {
    this._elementDict[id] = obj;
  }
  /**
   * Maximum updateLayout cycle count to reach the convergence of the layout
   */
  static readonly MAX_LAYOUT_ITERATION = 30;

  static readonly SCALE_SPEED = 0.0001;

  static readonly MAXIMUM_SIZE = 10;

  static readonly MINIMUM_SIZE = 0.1;

  /**
   * Root <svg> element always have a <defs> element to contain referenced patterns used in the other elements
   */
  private defsElement!: SVGDefsElement;

  private _currentSize: Size = ZeroSize2();

  private _viewBoxX = -100;

  private _viewBoxY = -100;

  private _currentScale = 1;

  private _elementDict: { [id: string]: GraphObject } = {};

  private _layoutSteps: (() => void)[][] = [[]];

  constructor() {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'svg'));
    this.transform.onSvgElementUpdate = () => {
      this.applyAttribute();
    };
    this.createDefNode();
    this.transform.onCustomContentSizeCalculate = () => {
      return this._currentSize;
    };

    // Add event listeners for moving canvases
    this.typedElement.addEventListener('mousedown', () => {
      window.addEventListener('mouseup', () => {
        window.removeEventListener('mousemove', this._canvasMove);
      });
      window.addEventListener('mousemove', this._canvasMove);
    });
    this.typedElement.addEventListener('wheel', (e: WheelEvent) => {
      this._currentScale = Math.min(
        GraphRoot.MAXIMUM_SIZE,
        Math.max(
          GraphRoot.MINIMUM_SIZE,
          this._currentScale + e.deltaY * GraphRoot.SCALE_SPEED,
        ),
      );
      this._updateCanvas();
      e.preventDefault();
    });
  }

  private _canvasMove = (e: MouseEvent) => {
    this._viewBoxX -= e.movementX * this._currentScale;
    this._viewBoxY -= e.movementY * this._currentScale;
    this._updateCanvas();
  };

  private _updateCanvas() {
    this.withStyle({
      viewBox: `${this._viewBoxX},${this._viewBoxY},${
        this._currentSize.width * this._currentScale
      },${this._currentSize.height * this._currentScale}`,
    });
    this.applyAttribute();
  }

  /**
   * Fits and centers the specified bounding box within the container viewport.
   * Scales down to fit large graphs, centers small graphs with a maximum scale of 1.0.
   *
   * @param x - Bounding box X origin in graph space.
   * @param y - Bounding box Y origin in graph space.
   * @param width - Bounding box width.
   * @param height - Bounding box height.
   * @param padding - Optional padding in pixels around the bounding box (default 20).
   */
  public fitBounds(
    x: number,
    y: number,
    width: number,
    height: number,
    padding = 20,
  ): void {
    let containerWidth = this._currentSize.width;
    let containerHeight = this._currentSize.height;

    if (containerWidth <= 0 || containerHeight <= 0) {
      const parent = this.typedElement.parentElement;
      if (parent) {
        const bbox = parent.getBoundingClientRect();
        containerWidth = bbox.width;
        containerHeight = bbox.height;
        this._currentSize = { width: bbox.width, height: bbox.height };
      }
    }

    if (
      containerWidth <= 0 ||
      containerHeight <= 0 ||
      width <= 0 ||
      height <= 0
    ) {
      return;
    }

    const requiredWidth = width + padding * 2;
    const requiredHeight = height + padding * 2;

    const scaleX = requiredWidth / containerWidth;
    const scaleY = requiredHeight / containerHeight;

    // ViewBox scale factor: targetScale >= 1.0 means zoom level <= 1.0 (no excessive magnification).
    const targetScale = Math.min(
      GraphRoot.MAXIMUM_SIZE,
      Math.max(1.0, Math.max(scaleX, scaleY)),
    );

    this._currentScale = targetScale;
    const viewWidth = containerWidth * targetScale;
    const viewHeight = containerHeight * targetScale;

    this._viewBoxX = x - padding + (requiredWidth - viewWidth) / 2;
    this._viewBoxY = y - padding + (requiredHeight - viewHeight) / 2;

    this._updateCanvas();
  }

  /**
   * Attach <svg> element to the parent DOM and follow the size of the parent.
   * @param parentDom A parent element that this <svg> element should be appened to.
   */
  public attach(parentDom: HTMLElement): void {
    parentDom.appendChild(this.typedElement);
    const _parentSizeMonitorLoop = () => {
      const bbox = parentDom.getBoundingClientRect();
      if (!SameSize2(bbox, this._currentSize)) {
        this._currentSize = {
          width: bbox.width,
          height: bbox.height,
        };
        this.withStyle({
          ...this._currentSize,
        });
        this.updateLayout();
      }
      requestAnimationFrame(_parentSizeMonitorLoop);
    };
    _parentSizeMonitorLoop();
  }

  public withDefs(defs: { [key: string]: GraphObject }): this {
    for (const key in defs) {
      const child = defs[key];
      if (!child.element) {
        throw new Error('Root element of def must not be an empty');
      }
      child.element.id = key;
      this.defsElement.appendChild(child.element);
      child.applyAttributeRecursively();
      child.transform.resolveDomHierarchy(this);
      for (let i = 0; i < GraphRoot.MAX_LAYOUT_ITERATION; i++) {
        if (!child.transform.updateLayout(0, 0)) {
          break;
        }
      }
      child.applyAttributeRecursively();
    }
    return this;
  }

  public render(): boolean {
    const scale = this._currentScale;
    this._currentScale = 1; // Calculate everything in scale=1 for easy calculation
    this._updateCanvas();
    this._elementDict = {};
    this.transform.resolveDomHierarchy(this);

    let layoutResult = true;
    for (let i = 0; i < this._layoutSteps.length; i++) {
      this.transform.invalidateLayoutRecursively();
      layoutResult = layoutResult && this.updateLayout();
      this._layoutSteps[i].forEach((action) => action());
    }
    this._currentScale = scale;
    this._updateCanvas();
    return layoutResult;
  }

  public updateLayout(): boolean {
    return this.transform.updateLayout(0, 0);
  }

  public find(id: string): GraphObject | null {
    return this._elementDict[id] ?? null;
  }

  public clearChildren() {
    this.getChildren().forEach((c) => c.detach());
    this._layoutSteps = [[]];
  }

  private createDefNode(): void {
    this.defsElement = document.createElementNS(
      'http://www.w3.org/2000/svg',
      'defs',
    );
    this.typedElement.appendChild(this.defsElement);
  }

  public registerLayoutStep(step: number, action: () => void): this {
    if (this._layoutSteps.length <= step) {
      this._layoutSteps.push([]);
    }
    this._layoutSteps[step].push(action);
    return this;
  }
}
