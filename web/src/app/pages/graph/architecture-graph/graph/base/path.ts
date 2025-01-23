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
import {
  AnchorPoints,
  Direction,
  ElementStyle,
  GraphObject,
  GraphObjectWithElementType,
  Vector2,
} from './base-containers';

interface CalculatedPath {
  source: string;
  path: number[];
  pipes: PathPipe[];
  definition: PathDefinition;
}

export type EdgeShape = 'arrow' | 'circle';

interface PathDefinition {
  route: string;
  headSize: number;
  headShape: EdgeShape;
  headDirection: number;
  pathStyle: ElementStyle;
}

export class PathPipe extends GraphObjectWithElementType<SVGGElement> {
  private static OFFSET_PER_LINE = 30;

  private _owningPathDefinitions: PathDefinition[] = [];

  private _connectedPoints: { [id: string]: GraphObject } = {};

  private _connectedAnchors: { [id: string]: Vector2 } = {};

  private _connectedPipe: { [id: string]: PathPipe } = {};

  private _connectedSourceNameToLineIndex: Map<string, number> = new Map<
    string,
    number
  >();

  private _minMaxCoordinateForLine: Vector2[] = [];

  private _connectedLineCount = 0;

  constructor(
    public readonly direction: Direction,
    public readonly routeId: string,
  ) {
    super(document.createElementNS('http://www.w3.org/2000/svg', 'g'));
    this.withId(routeId);
    this.transform.onCustomContentSizeCalculate = () => {
      switch (this.direction) {
        case Direction.Horizontal:
          return {
            height:
              this._connectedLineCount * PathPipe.OFFSET_PER_LINE +
              PathPipe.OFFSET_PER_LINE,
            width: 0,
          };
        case Direction.Vertical:
          return {
            width:
              this._connectedLineCount * PathPipe.OFFSET_PER_LINE +
              PathPipe.OFFSET_PER_LINE,
            height: 0,
          };
      }
      throw new Error('unreachable');
    };
  }

  public override _onAttachedToGraphRoot(root: GraphRoot): void {
    super._onAttachedToGraphRoot(root);
    root.registerLayoutStep(1, () => {
      this._owningPathDefinitions.forEach((p) =>
        this.prepareConnection(p.route),
      );
    });
    root.registerLayoutStep(2, () => {
      const paths = this._owningPathDefinitions.map((p) =>
        this.calculatePath(p),
      );
      paths.forEach((cp) => this.drawCalculatedPath(cp));
    });
  }

  public connectPipe(nextPipe: PathPipe): void {
    if (this.direction == nextPipe.direction)
      throw new Error(
        'Invalid connection! Pipe direction must not be same when it was connected to.',
      );
    this._connectedPipe[nextPipe.routeId] = nextPipe;
    nextPipe._connectedPipe[this.routeId] = this;
  }

  public connectPoint(point: GraphObject, anchor: Vector2) {
    if (point.getId() == null)
      throw new Error('Connected point must have an id!');
    const id = point.getId()!;
    this._connectedPoints[id] = point;
    this._connectedAnchors[id] = anchor;
  }

  public registerPath(
    route: string,
    headShape: EdgeShape,
    headSize: number,
    headDirection: number,
    pathStyle: ElementStyle = {},
  ): void {
    this._owningPathDefinitions.push({
      route,
      headShape,
      headSize,
      headDirection,
      pathStyle,
    });
  }

  public prepareConnection(route: string) {
    const splittedRoute = route.split('/').reverse();
    const source = splittedRoute[splittedRoute.length - 1];
    splittedRoute.pop();
    splittedRoute.pop();
    this._prepareConnection(splittedRoute, source);
  }

  private _prepareConnection(route: string[], source: string) {
    if (!this._connectedSourceNameToLineIndex.has(source)) {
      this._connectedSourceNameToLineIndex.set(
        source,
        this._connectedLineCount,
      );
      this._connectedLineCount += 1;
      this._minMaxCoordinateForLine.push({ x: 10000000, y: -10000000 });
    }
    const next = route.pop();
    if (route.length > 0) {
      const pipe = this._connectedPipe[next!];
      pipe._prepareConnection(route, source);
    }
  }

  public drawCalculatedPath(calculatedPath: CalculatedPath) {
    const path = calculatedPath.path;
    let result = '';
    for (let i = 0; i < path.length; i += 2) {
      if (i == 0) {
        result += `M ${path[i]},${path[i + 1]}`;
      } else {
        result += ` L ${path[i]},${path[i + 1]}`;
      }
    }
    const pathElement = document.createElementNS(
      'http://www.w3.org/2000/svg',
      'path',
    );
    pathElement.setAttribute('stroke', 'black');
    pathElement.setAttribute('fill', 'none');
    pathElement.setAttribute('d', result);
    pathElement.setAttribute('stroke-width', '3');
    for (const styleKey in calculatedPath.definition.pathStyle) {
      pathElement.setAttribute(
        styleKey,
        calculatedPath.definition.pathStyle[styleKey] as string,
      );
    }
    this._root?.element!.appendChild(pathElement);
    for (let i = 2; i < path.length - 2; i += 2) {
      const pointIndex = i / 2 - 1;
      const prevPipe = pointIndex - 1;

      const connectedPipes = [];
      if (pointIndex < calculatedPath.pipes.length) {
        connectedPipes.push(calculatedPath.pipes[pointIndex]);
      }
      if (prevPipe >= 0) {
        connectedPipes.push(calculatedPath.pipes[prevPipe]);
      }

      let jointVisible = false;
      for (const pipe of connectedPipes) {
        jointVisible =
          jointVisible ||
          pipe.isVisibleJointForLine(calculatedPath.source, {
            x: path[i],
            y: path[i + 1],
          });
      }
      if (!jointVisible) continue;
      const jointElement = document.createElementNS(
        'http://www.w3.org/2000/svg',
        'circle',
      );
      jointElement.setAttribute('r', '5');
      jointElement.setAttribute('cx', '' + path[i]);
      jointElement.setAttribute('cy', '' + path[i + 1]);
      this._root?.element!.appendChild(jointElement);
    }
    const pathEdge = this.createHeadEdge(
      calculatedPath.definition,
      path[path.length - 2],
      path[path.length - 1],
    );
    if (pathEdge) {
      this._root?.element!.appendChild(pathEdge);
    }
    return result;
  }

  public calculatePath(definition: PathDefinition): CalculatedPath {
    const splittedRoute = definition.route.split('/').reverse();
    const beginObject =
      this._connectedPoints[splittedRoute[splittedRoute.length - 1]];
    const beginAnchor =
      this._connectedAnchors[splittedRoute[splittedRoute.length - 1]];
    if (beginObject == undefined) {
      throw new Error(
        `object ${splittedRoute[splittedRoute.length - 1]} not found`,
      );
    }
    const fp = beginObject.transform.anchorToPoint(beginAnchor);
    const path = [fp.x, fp.y];
    const source = splittedRoute.pop();
    splittedRoute.pop();
    this._calculatePath(splittedRoute, path, fp, null, source!);
    return {
      source: source!,
      path,
      pipes: this.getRoutePipes(definition.route),
      definition,
    };
  }

  private _calculatePath(
    route: string[],
    currentPath: number[],
    from: Vector2,
    fromPipe: PathPipe | null,
    source: string,
  ) {
    if (route.length == 1) {
      // Destination should be connected to this pipe
      const pipeCenter = this.transform.anchorToPoint(AnchorPoints.CENTER);
      const middle = from;
      switch (this.direction) {
        case Direction.Horizontal:
          middle.y = pipeCenter.y + this.getOffsetFromSourceName(source);
          break;
        case Direction.Vertical:
          middle.x = pipeCenter.x + this.getOffsetFromSourceName(source);
          break;
      }
      currentPath.push(middle.x, middle.y);
      if (fromPipe) {
        fromPipe.includeJointForLine(source, middle);
      }
      this.includeJointForLine(source, middle);
      const pipeEndPoint = from;
      const endObject = this._connectedPoints[route[route.length - 1]];
      const endAnchor = this._connectedAnchors[route[route.length - 1]];
      const ep = endObject.transform.anchorToPoint(endAnchor);
      switch (this.direction) {
        case Direction.Horizontal:
          pipeEndPoint.x = ep.x;
          break;
        case Direction.Vertical:
          pipeEndPoint.y = ep.y;
          break;
      }
      this.includeJointForLine(source, pipeEndPoint);
      currentPath.push(pipeEndPoint.x, pipeEndPoint.y, ep.x, ep.y);
    } else {
      const pipeCenter = this.transform.anchorToPoint(AnchorPoints.CENTER);
      const pipeFirstPoint = from;
      switch (this.direction) {
        case Direction.Horizontal:
          pipeFirstPoint.y =
            pipeCenter.y + this.getOffsetFromSourceName(source);
          break;
        case Direction.Vertical:
          pipeFirstPoint.x =
            pipeCenter.x + this.getOffsetFromSourceName(source);
          break;
      }
      currentPath.push(pipeFirstPoint.x, pipeFirstPoint.y);
      const nextPipeId = route.pop()!;
      const nextPipe = this._connectedPipe[nextPipeId];
      if (fromPipe) {
        fromPipe.includeJointForLine(source, pipeFirstPoint);
      }
      this.includeJointForLine(source, pipeFirstPoint);
      nextPipe._calculatePath(route, currentPath, pipeFirstPoint, this, source);
    }
  }

  private getOffsetFromSourceName(sourceName: string): number {
    const lineIndex = this._connectedSourceNameToLineIndex.get(sourceName);
    if (lineIndex === undefined) {
      throw new Error(`Unknown source name ${sourceName}`);
    }
    return (
      lineIndex * PathPipe.OFFSET_PER_LINE -
      (PathPipe.OFFSET_PER_LINE * this._connectedSourceNameToLineIndex.size) / 2
    );
  }

  private includeJointForLine(sourceName: string, point: Vector2) {
    const lineIndex = this._connectedSourceNameToLineIndex.get(sourceName);
    if (lineIndex === undefined) {
      throw new Error(`Unknown source name ${sourceName}`);
    }
    let coordinateInTheDirection = 0;
    switch (this.direction) {
      case Direction.Horizontal:
        coordinateInTheDirection = point.x;
        break;
      case Direction.Vertical:
        coordinateInTheDirection = point.y;
        break;
    }
    const minMax = this._minMaxCoordinateForLine[lineIndex];
    minMax.x = Math.min(minMax.x, coordinateInTheDirection);
    minMax.y = Math.max(minMax.y, coordinateInTheDirection);
  }

  /**
   * Get the list of PathPipe from route string
   * @param route
   * @returns
   */
  private getRoutePipes(route: string): PathPipe[] {
    let splittedRoute = route.split('/');
    // Remove first and last route name to filter pipe names
    splittedRoute.pop();
    splittedRoute = splittedRoute.reverse();
    splittedRoute.pop();

    /*  eslint-disable-next-line @typescript-eslint/no-this-alias */
    let currentPipe: PathPipe = this;
    const result: PathPipe[] = [this];
    splittedRoute.pop();
    while (splittedRoute.length > 0) {
      const nextPipe = splittedRoute.pop();
      currentPipe = this._connectedPipe[nextPipe!];
      result.push(currentPipe);
    }
    return result;
  }

  private isVisibleJointForLine(sourceName: string, point: Vector2): boolean {
    const lineIndex = this._connectedSourceNameToLineIndex.get(sourceName);
    if (lineIndex === undefined) {
      throw new Error(`Unknown source name ${sourceName}`);
    }
    const minmax = this._minMaxCoordinateForLine[lineIndex];
    switch (this.direction) {
      case Direction.Horizontal:
        return minmax.x != point.x && minmax.y != point.x;
      case Direction.Vertical:
        return minmax.x != point.y && minmax.y != point.y;
    }
    throw new Error('unreachable');
  }

  private createHeadEdge(
    definition: PathDefinition,
    x: number,
    y: number,
  ): SVGElement | null {
    switch (definition.headShape) {
      case 'arrow':
        return this.createArrowHead(
          x,
          y,
          definition.headSize,
          definition.headDirection,
        );
      case 'circle':
        return this.createCircleHead(x, y, definition.headSize);
    }
    return null;
  }

  private createArrowHead(
    x: number,
    y: number,
    size: number,
    rotation: number,
  ): SVGElement {
    const arrowHead = document.createElementNS(
      'http://www.w3.org/2000/svg',
      'path',
    );
    const l = size;
    const path = `M 0,${l} L${-l},${-l} L0,${-l / 2} L${l},${-l}Z`;
    arrowHead.setAttribute('d', path);
    arrowHead.setAttribute(
      'transform',
      `translate(${x},${y})rotate(${rotation + 180})`,
    );
    return arrowHead;
  }

  private createCircleHead(
    x: number,
    y: number,
    size: number,
  ): SVGCircleElement {
    const arrowHead = document.createElementNS(
      'http://www.w3.org/2000/svg',
      'circle',
    );
    arrowHead.setAttribute('r', '' + size);
    arrowHead.setAttribute('transform', `translate(${x},${y})`);
    return arrowHead;
  }
}
