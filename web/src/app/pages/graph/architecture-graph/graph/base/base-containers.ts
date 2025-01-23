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

import { GraphRoot } from '../graph-root';

export interface ElementStyle {
  width?: number | string;
  height?: number | string;
  viewBox?: string;
  fill?: string;
  'stroke-width'?: string;
  'stroke-dasharray'?: string;
  stroke?: string;
  rx?: number;
  ry?: number;
  r?: number;
  'font-size'?: number;
  'font-weight'?: number;
  [key: string]: unknown;
}

export enum Direction {
  Horizontal = 1 << 0,
  Vertical = 1 << 1,
}

export interface Vector2 {
  x: number;
  y: number;
}

export interface MarginLike {
  left: number;
  right: number;
  top: number;
  bottom: number;
}

export function ZeroVec2(): Vector2 {
  return { x: 0, y: 0 };
}

export function ZeroSize2(): Size {
  return { width: 0, height: 0 };
}

export function SameVector2(v1: Vector2, v2: Vector2): boolean {
  return v1.x == v2.x && v1.y == v2.y;
}

export function SameSize2(v1: Size, v2: Size): boolean {
  return v1.width == v2.width && v1.height == v2.height;
}

export const AnchorPoints = {
  TOP_LEFT: {
    x: 0,
    y: 0,
  },
  TOP: {
    x: 0.5,
    y: 0,
  },
  TOP_RIGHT: {
    x: 1,
    y: 0,
  },
  CENTER_LEFT: {
    x: 0,
    y: 0.5,
  },
  CENTER: {
    x: 0.5,
    y: 0.5,
  },
  CENTER_RIGHT: {
    x: 1,
    y: 0.5,
  },
  BOTTOM_LEFT: {
    x: 0,
    y: 1,
  },
  BOTTOM: {
    x: 0.5,
    y: 1,
  },
  BOTTOM_RIGHT: {
    x: 1,
    y: 1,
  },
};

export interface Size {
  width: number;
  height: number;
}

type OnSvgElementTransformUpdate = (position: Vector2, size: Size) => void;

type OnCalculateChildrenOffsets = (originalOffset: Vector2) => Vector2[];

type OnCalculateContentSize = () => Size;

type OnGetChildDomAttachTarget = () => SVGElement | null;

/**
 * Provide and wrap all of the functionality to calculate Svg elements
 *
 */
export class GraphTransform {
  public requireChildSizeToLayout = true;

  private _parent: GraphTransform | null = null;

  private _children: GraphTransform[] = [];

  private _anchor: Vector2 = AnchorPoints.TOP_LEFT;

  private _pivot: Vector2 = AnchorPoints.TOP_LEFT;

  private _minSize: Size = ZeroSize2();

  private _lastSizeWithMargin: Size = ZeroSize2();

  private _lastContentSize: Size = ZeroSize2();

  private _margin: MarginLike = { left: 0, right: 0, top: 0, bottom: 0 };

  private _lastPosition: Vector2 = ZeroVec2();

  private _influenceParentContentSize = true;

  private _sizeDecided = false;

  private _locationDecided = false;

  public onCustomContentSizeCalculate: OnCalculateContentSize | null = null;

  public onSvgElementUpdate: OnSvgElementTransformUpdate | null = null;

  public onGetChildDomAttachTarget: OnGetChildDomAttachTarget = () => {
    return this._graphObject.element;
  };

  public onOverrideChildrenOffsets: OnCalculateChildrenOffsets = (pos) => {
    return this.children.map(() => pos);
  };

  constructor(public readonly _graphObject: GraphObject) {}

  public get contentSize(): Size {
    if (!this._sizeDecided) {
      throw new Error('Content size is not yet decided!');
    }
    return this._lastContentSize;
  }

  public get sizeWithMargin(): Size {
    if (!this._sizeDecided) {
      throw new Error('Size is not yet decided!');
    }
    return this._lastSizeWithMargin;
  }

  public get anchor(): Vector2 {
    return this._anchor;
  }

  public set anchor(v: Vector2) {
    if (SameVector2(v, this.anchor)) return;
    this._verifyInvalidMarginAnchorPivotCombination();
    this._anchor = {
      ...v,
    };
  }

  public get pivot(): Vector2 {
    return this._pivot;
  }

  public set pivot(v: Vector2) {
    if (SameVector2(v, this.pivot)) return;
    this._verifyInvalidMarginAnchorPivotCombination();
    this._pivot = {
      ...v,
    };
  }

  public get minSize(): Size {
    return this._minSize;
  }

  public set minSize(v: Size) {
    if (SameSize2(v, this.minSize)) return;
    this._minSize = {
      ...v,
    };
  }

  public get margin(): MarginLike {
    return this._margin;
  }

  public set margin(v: MarginLike) {
    if (
      v.left != this._margin.left ||
      v.right != this._margin.right ||
      v.top != this._margin.top ||
      v.bottom != this._margin.bottom
    ) {
      this._verifyInvalidMarginAnchorPivotCombination();
      this._margin = {
        ...v,
      };
    }
  }

  public get parent(): GraphTransform | null {
    return this._parent;
  }

  public set parentGraphObject(parentObj: GraphObject | null) {
    if (parentObj == null) {
      if (this._parent == null) return;
      if (this._parent._graphObject.element && this._graphObject.element) {
        const index = this._parent._children.indexOf(this);
        this._parent._graphObject.element.removeChild(
          this._graphObject.element,
        );
        if (index > -1) {
          this._parent.children.splice(index, 1);
        }
      }
      this._parent = null;
      return;
    }
    this._parent = parentObj.transform;
    this._parent._children.push(this);
  }

  public get children(): GraphTransform[] {
    return this._children;
  }

  public get influenceParentContentSize(): boolean {
    return this._influenceParentContentSize;
  }

  public set influenceParentContentSize(value: boolean) {
    if (this._influenceParentContentSize == value) return;
    this._influenceParentContentSize = value;
  }

  /**
   * Reset layout calculation status to recalculate
   * @returns Count of transforms require layout recalculation
   */
  public invalidateLayoutRecursively(): number {
    this._sizeDecided = false;
    this._locationDecided = false;
    return this.children.reduce(
      (p, c) => p + c.invalidateLayoutRecursively(),
      1,
    );
  }

  public resolveDomHierarchy(root: GraphRoot): void {
    for (const child of this.children) {
      const addTo = this.onGetChildDomAttachTarget();
      if (!addTo) {
        throw new Error('Failed to get dom parent!');
      }
      if (child._graphObject.element) {
        addTo.appendChild(child._graphObject.element);
      }
      child._graphObject._onAttachedToGraphRoot(root);
    }
    this.children.forEach((c) => c.resolveDomHierarchy(root));
  }

  public updateLayout(x = 0, y = 0, max_layout_iteration = 100): boolean {
    for (let i = 0; i < max_layout_iteration; i++)
      if (this._updateSize()) {
        break;
      }
    if (!this._sizeDecided) return false;
    for (let i = 0; i < max_layout_iteration; i++)
      if (this._updatePositions(x, y)) {
        break;
      }
    return true;
  }

  private _updateSize(): boolean {
    if (this._sizeDecided) return true;
    if (!this.isReadyToCalculateLayout()) {
      this.children.forEach((c) => c._updateSize());
      return false;
    }
    this._lastContentSize = this._calculateContentSize();
    this._lastSizeWithMargin = this.calculateSizeFromContentSize(
      this._lastContentSize,
    );
    this._sizeDecided = true;
    return true;
  }

  private _updatePositions(x: number, y: number): boolean {
    if (this._locationDecided) return true;
    let nextPosition: Vector2 = ZeroVec2();
    if (this._parent) {
      nextPosition = this.calculatePosition(
        this._lastContentSize,
        { x, y },
        this._parent._lastSizeWithMargin,
      );
    } else {
      nextPosition = this.calculatePosition(this._lastContentSize);
    }
    if (this.onSvgElementUpdate)
      this.onSvgElementUpdate(nextPosition, this._lastSizeWithMargin);

    const offsetToChildren = this.onOverrideChildrenOffsets(nextPosition);
    for (let childIndex = 0; childIndex < this.children.length; childIndex++) {
      const child = this.children[childIndex];
      child._updatePositions(
        offsetToChildren[childIndex].x,
        offsetToChildren[childIndex].y,
      );
    }
    this._locationDecided = true;
    return true;
  }

  /**
   * Check if this element can decide layout now or not
   */
  private isReadyToCalculateLayout(): boolean {
    return (
      !this.requireChildSizeToLayout ||
      this.children.reduce((p, c) => p && c._sizeDecided, true)
    );
  }

  public calculatePosition(
    contentSize: Size = ZeroSize2(),
    parentOffset: Vector2 = ZeroVec2(),
    parentSize: Size | null = null,
  ): Vector2 {
    const newLocation: Vector2 = { x: 0, y: 0 };
    const { x, y } = parentOffset;
    if (parentSize) {
      const { width, height } = parentSize;
      const ax = x + width * this.anchor.x;
      const ay = y + height * this.anchor.y;
      const cx = -contentSize.width * this.pivot.x;
      const cy = -contentSize.height * this.pivot.y;
      newLocation.x = ax + this.margin.left + cx;
      newLocation.y = ay + this.margin.top + cy;
    } else {
      newLocation.x = x;
      newLocation.y = y;
    }
    if (!SameVector2(this._lastPosition, newLocation)) {
      this._lastPosition = { ...newLocation };
    }
    return newLocation;
  }

  public calculateSize(): Size {
    return this.calculateSizeFromContentSize(this._calculateContentSize());
  }

  public anchorToPoint(anchor: Vector2): Vector2 {
    const x = this._lastPosition.x + this._lastSizeWithMargin.width * anchor.x;
    const y = this._lastPosition.y + this._lastSizeWithMargin.height * anchor.y;
    return { x, y };
  }

  private calculateSizeFromContentSize(contentSize: Size): Size {
    const size: Size = {
      width: contentSize.width,
      height: contentSize.height,
    };
    size.width = Math.max(size.width, this._minSize.width);
    size.height = Math.max(size.height, this._minSize.height);
    return size;
  }

  private _calculateContentSize(): Size {
    if (this.onCustomContentSizeCalculate) {
      return this.onCustomContentSizeCalculate();
    } else {
      // TODO: Add margin in consideration
      const result = ZeroSize2();
      for (const c of this.children) {
        if (c.influenceParentContentSize) {
          const size = c.calculateSize();
          const margin = c.margin;
          result.width = Math.max(
            result.width,
            size.width + margin.left + margin.right,
          );
          result.height = Math.max(
            result.height,
            size.height + margin.top + margin.bottom,
          );
        }
      }
      return result;
    }
  }

  private _verifyInvalidMarginAnchorPivotCombination(): void {
    if (
      this.margin.bottom != 0 ||
      this.margin.top != 0 ||
      this.margin.left != 0 ||
      this.margin.right != 0
    ) {
      if (
        !SameVector2(this._anchor, AnchorPoints.TOP_LEFT) ||
        !SameVector2(this._anchor, AnchorPoints.TOP_LEFT)
      ) {
        throw new Error(
          'Invalid combination of anchor and margin. Margin is only supports TOP_LEFT anchor and pivot',
        );
      }
    }
  }
}

export class GraphObject {
  protected _transform: GraphTransform = new GraphTransform(this);

  protected _style: object = {};

  protected _lastParentPosition: Vector2 = { x: 0, y: 0 };

  protected _id: string | null = null;

  protected _root: GraphRoot | null = null;

  constructor(public element: SVGElement | null) {
    this.transform.onSvgElementUpdate = (pos, size) => {
      this.withStyle({
        x: pos.x,
        y: pos.y,
        width: size.width,
        height: size.height,
      });
      this.applyAttribute();
    };
  }

  public get transform(): GraphTransform {
    return this._transform;
  }

  public getId(): string | null {
    return this._id;
  }

  public getAttribute(): ElementStyle {
    return this._style as ElementStyle;
  }

  public getParent<T extends GraphObject = GraphObject>(): T | null {
    return this.transform.parent?._graphObject as T | null;
  }

  public getChildren(): GraphObject[] {
    return this.transform.children.map((c) => c._graphObject);
  }

  public __getDomStructureElementToAddChildren(): SVGElement | null {
    return this.element;
  }

  public __updateParentPosition(x: number, y: number): void {
    this._lastParentPosition = { x, y };
  }

  public applyAttribute(): void {
    if (!this.element) return;
    const attributes = this.getAttribute();
    for (const key in attributes) {
      this.element.setAttribute(key, attributes[key] as string);
    }
  }

  public applyAttributeRecursively(): void {
    this.applyAttribute();
    for (const child of this.getChildren()) {
      child.applyAttributeRecursively();
    }
  }

  public withAnchor(anchor: Vector2): this {
    this._transform.anchor = anchor;
    return this;
  }

  public withPivot(pivot: Vector2): this {
    this._transform.pivot = pivot;
    return this;
  }

  public withChildren(children: GraphObject[]): this {
    children.forEach((c) => (c.transform.parentGraphObject = this));
    children.forEach((c) => c.applyAttribute());
    return this;
  }

  public withStyle(style: ElementStyle): this {
    this._style = {
      ...this._style,
      ...style,
    };
    return this;
  }

  public withIgnoredFromParentSizing(): this {
    this.transform.influenceParentContentSize = false;
    return this;
  }

  public withMinSize(width: number, height: number): this {
    this.transform.minSize = { width, height };
    return this;
  }

  public withMargin(
    top: number,
    right: number,
    bottom: number,
    left: number,
  ): this {
    this.transform.margin = { top, right, bottom, left };
    return this;
  }

  public withId(id: string): this {
    if (this.element) this.element.id = id;
    else
      console.warn(
        `ID ${id} was attempted to assign to non element svg object`,
      );
    this._id = id;
    return this;
  }

  public detach(): void {
    this.transform.parentGraphObject = null;
  }

  public _onAttachedToGraphRoot(root: GraphRoot) {
    if (this._id != null) root.registerGraphObjectWithId(this, this._id);
    this._root = root;
  }
}

export class GraphObjectWithElementType<
  T extends SVGElement | null,
> extends GraphObject {
  constructor(protected typedElement: T) {
    super(typedElement);
  }
}

export class AlighedGraphObjectContainer extends GraphObjectWithElementType<SVGGElement> {
  private _gap = 0;

  private _expandChildren = true;

  constructor(public readonly direction: Direction) {
    super(
      document.createElementNS(
        'http://www.w3.org/2000/svg',
        'g',
      ) as SVGGElement,
    );
    this.transform.onOverrideChildrenOffsets = (originalPos) => {
      const offsets = [];
      let changingSideOffset = 0;
      let maxChildSize = 0;
      for (const child of this.transform.children) {
        const childSize = child.calculateSize();
        const childMargin = child.margin;
        switch (this.direction) {
          case Direction.Horizontal:
            offsets.push({
              x: childMargin.left + originalPos.x + changingSideOffset,
              y: originalPos.y + childMargin.top,
            });
            changingSideOffset +=
              childSize.width + childMargin.left + childMargin.right;
            changingSideOffset += this._gap;
            maxChildSize = Math.max(
              maxChildSize,
              childSize.height + childMargin.top + childMargin.bottom,
            );
            break;
          case Direction.Vertical:
            offsets.push({
              y: childMargin.top + originalPos.y + changingSideOffset,
              x: originalPos.x + childMargin.left,
            });
            changingSideOffset +=
              childSize.height + childMargin.top + childMargin.bottom;
            changingSideOffset += this._gap;
            maxChildSize = Math.max(
              maxChildSize,
              childSize.width + childMargin.left + childMargin.right,
            );
            break;
        }
      }
      if (this._expandChildren) {
        for (const child of this.transform.children) {
          switch (this.direction) {
            case Direction.Horizontal:
              child.minSize = { height: maxChildSize, width: 0 };
              break;
            case Direction.Vertical:
              child.minSize = { height: 0, width: maxChildSize };
              break;
          }
        }
      }
      return offsets;
    };
    this.transform.onSvgElementUpdate = () => {
      return;
    };
    this.transform.onCustomContentSizeCalculate = () => {
      const childTransforms = this.transform.children;
      switch (this.direction) {
        case Direction.Horizontal:
          return {
            width:
              childTransforms.reduce((p, c) => p + c.sizeWithMargin.width, 0) +
              this._gap * Math.max(0, childTransforms.length - 1),
            height: childTransforms.reduce(
              (p, c) => Math.max(p, c.sizeWithMargin.height),
              0,
            ),
          };
        case Direction.Vertical:
          return {
            height:
              childTransforms.reduce((p, c) => p + c.sizeWithMargin.height, 0) +
              this._gap * Math.max(0, childTransforms.length - 1),
            width: childTransforms.reduce(
              (p, c) => Math.max(p, c.sizeWithMargin.width),
              0,
            ),
          };
      }
      throw new Error('unreachable');
    };
  }

  public withGap(gap: number): this {
    this._gap = gap;
    return this;
  }

  public disableExpandingChildren(): this {
    this._expandChildren = false;
    return this;
  }
}
