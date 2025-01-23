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

import { GraphObjectWithElementType, ElementStyle } from './base-containers';

const DEFAULT_SHAPE_COLOR = '#333';

export class ArchShape<
  T extends SVGGraphicsElement | null,
> extends GraphObjectWithElementType<T> {
  constructor(shapeElement: T) {
    super(shapeElement);
    this.withStyle({
      fill: DEFAULT_SHAPE_COLOR,
    });
    this.transform.onGetChildDomAttachTarget = () => {
      return this.transform.parent!.onGetChildDomAttachTarget();
    };
  }
}

export class ArchRect extends ArchShape<SVGRectElement> {
  constructor() {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'rect'));
  }
}

export class ArchEmpty extends GraphObjectWithElementType<null> {
  constructor() {
    super(null);
    this.transform.onGetChildDomAttachTarget = () => {
      return this.transform.parent!.onGetChildDomAttachTarget();
    };
  }
}

export class ArchCircle extends ArchShape<SVGCircleElement> {
  constructor() {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'circle'));
    this.transform.onSvgElementUpdate = (pos) => {
      this.withStyle({
        cx: pos.x,
        cy: pos.y,
      });
    };
  }
}

export class ArchLabel extends ArchShape<SVGTextElement> {
  DEFAULT_FONT_SIZE = 14;

  constructor(label: string) {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'text'));

    // Default font size is necessary to adjust text location to fit parent box
    this.withStyle({
      'font-size': this.DEFAULT_FONT_SIZE,
      'font-family': 'Roboto',
    });

    this.transform.requireChildSizeToLayout = false;
    this.transform.onSvgElementUpdate = (pos) => {
      this.withStyle({
        x: pos.x,
        y: pos.y + this.fontSize,
      });
    };
    this.transform.onCustomContentSizeCalculate = () => {
      const bbox = this.typedElement.getBoundingClientRect();
      return { width: bbox.width, height: bbox.height };
    };
    this.typedElement.innerHTML = label;
  }

  private get fontSize(): number {
    return this.getAttribute()['font-size'] || this.DEFAULT_FONT_SIZE;
  }

  public override withStyle(style: ElementStyle): this {
    super.withStyle(style);
    this.applyAttribute();
    return this;
  }
}

export class SizedRect extends ArchShape<SVGRectElement> {
  constructor(
    private readonly width: number,
    private readonly height: number,
  ) {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'rect'));
    this.transform.requireChildSizeToLayout = false;
    this.transform.onCustomContentSizeCalculate = () => {
      return { width: this.width, height: this.height };
    };
    this.transform.onSvgElementUpdate = (pos) => {
      this.withStyle({
        x: pos.x,
        y: pos.y,
      });
      this.applyAttribute();
    };
    this.withStyle({});
    this.applyAttribute();
  }

  public override withStyle(style: ElementStyle): this {
    super.withStyle({
      ...style,
      width: this.width,
      height: this.height,
    });
    return this;
  }
}

export class ArrowHead extends ArchShape<SVGPathElement> {
  constructor(arrowHead: number, rotation: number) {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'path'));
    this.transform.onSvgElementUpdate = (p) => {
      this.typedElement.setAttribute(
        'transform',
        `rotate(${rotation + 180})translate(${p.x},${p.y})`,
      );
    };
    const l = arrowHead;
    const path = `M 0,${l} L${-l},${-l} L0,${-l / 2} L${l},${-l}Z`;
    this.typedElement.setAttribute('d', path);
  }
}
