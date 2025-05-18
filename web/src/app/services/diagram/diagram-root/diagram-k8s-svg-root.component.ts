/**
 * Copyright 2025 Google LLC
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

import { Component, computed, input } from '@angular/core';
import {
  DiagramModel,
  DiagramPathType,
  PathElement,
} from '../diagram-model-types';
import {
  ArrowShape,
  DiagramSVGArrowComponent,
  LinePattern,
  WayPoint,
  WaypointComplementType,
} from '../diagram-element/diagram-svg-arrow.component';

/**
 * Interface defining the visual style properties for arrows between diagram elements
 * Controls arrow appearance including shapes, sizes, anchor points and line patterns
 */
interface ArrowStyle {
  innerShape: ArrowShape;
  outerShape: ArrowShape;
  innerSize: number;
  outerSize: number;
  innerRotate: number;
  outerRotate: number;
  thickness: number;
  linePattern: LinePattern;
  innerAnchorX: number;
  innerAnchorY: number;
  outerAnchorX: number;
  outerAnchorY: number;
}

/**
 * SVG rendering component for Kubernetes diagram relationships
 * Handles drawing connection arrows between diagram elements based on path data
 */
@Component({
  // eslint-disable-next-line @angular-eslint/component-selector
  selector: '[diagram-k8s-svg-root]',
  templateUrl: './diagram-k8s-svg-root.component.html',
  imports: [DiagramSVGArrowComponent],
})
export class DiagramK8sSVGRootComponent {
  ArrowShape = ArrowShape;
  /**
   * Predefined arrow styles for connections between layers
   * Maps each path type to its visual representation style
   */
  ArrowStylesBetweenLayers: { [type in DiagramPathType]: ArrowStyle } = {
    [DiagramPathType.Invalid]: {
      innerShape: ArrowShape.None,
      outerShape: ArrowShape.None,
      innerSize: 0,
      outerSize: 0,
      innerRotate: 0,
      outerRotate: 0,
      thickness: 0,
      innerAnchorX: 0,
      innerAnchorY: 0,
      outerAnchorX: 0,
      outerAnchorY: 0,
      linePattern: LinePattern.Line,
    },
    [DiagramPathType.Owner]: {
      innerShape: ArrowShape.Circle,
      outerShape: ArrowShape.None,
      innerSize: 5,
      outerSize: 5,
      innerRotate: 0,
      outerRotate: 0,
      thickness: 1,
      innerAnchorX: 0,
      innerAnchorY: 0,
      outerAnchorX: 0.5,
      outerAnchorY: 1,
      linePattern: LinePattern.Dashed,
    },
    [DiagramPathType.Use]: {
      innerShape: ArrowShape.Circle,
      outerShape: ArrowShape.Circle,
      innerSize: 5,
      outerSize: 5,
      innerRotate: 0,
      outerRotate: 0,
      thickness: 1,
      innerAnchorX: 0.5,
      innerAnchorY: 0,
      outerAnchorX: 0.5,
      outerAnchorY: 1,
      linePattern: LinePattern.Line,
    },
    [DiagramPathType.Traffic]: {
      innerShape: ArrowShape.Arrow,
      outerShape: ArrowShape.None,
      innerSize: 5,
      outerSize: 5,
      innerRotate: 180,
      outerRotate: 0,
      thickness: 1,
      innerAnchorX: 0.5,
      innerAnchorY: 0,
      outerAnchorX: 0.5,
      outerAnchorY: 1,
      linePattern: LinePattern.Dotted,
    },
  };
  /**
   * Predefined arrow styles for connections between nodes and layers
   * Maps each path type to its visual representation style
   */
  ArrowStylesBetweenNodeAndLayer: { [type in DiagramPathType]: ArrowStyle } = {
    [DiagramPathType.Invalid]: {
      innerShape: ArrowShape.None,
      outerShape: ArrowShape.None,
      innerSize: 0,
      outerSize: 0,
      innerRotate: 0,
      outerRotate: 0,
      thickness: 0,
      innerAnchorX: 0,
      innerAnchorY: 0,
      outerAnchorX: 0,
      outerAnchorY: 0,
      linePattern: LinePattern.Line,
    },
    [DiagramPathType.Owner]: {
      innerShape: ArrowShape.Circle,
      outerShape: ArrowShape.None,
      innerSize: 5,
      outerSize: 5,
      innerRotate: 0,
      outerRotate: 0,
      thickness: 1,
      outerAnchorY: 1,
      innerAnchorX: 0,
      innerAnchorY: 0,
      outerAnchorX: 0.5,
      linePattern: LinePattern.Dashed,
    },
    [DiagramPathType.Use]: {
      innerShape: ArrowShape.None,
      outerShape: ArrowShape.Circle,
      innerSize: 5,
      outerSize: 5,
      innerRotate: 0,
      outerRotate: 0,
      thickness: 1,
      innerAnchorX: 0,
      innerAnchorY: 0.3,
      outerAnchorX: 0.5,
      outerAnchorY: 1,
      linePattern: LinePattern.Line,
    },
    [DiagramPathType.Traffic]: {
      innerShape: ArrowShape.Arrow,
      outerShape: ArrowShape.None,
      innerSize: 5,
      outerSize: 5,
      innerRotate: 90,
      outerRotate: 0,
      thickness: 1,
      innerAnchorX: 0,
      innerAnchorY: 0.7,
      outerAnchorX: 0.5,
      outerAnchorY: 1,
      linePattern: LinePattern.Dotted,
    },
  };

  /**
   * The diagram model to render
   */
  model = input.required<DiagramModel>();

  /**
   * Creates waypoints array for upper path elements
   * @param path The path element from upperPaths
   * @param layerIndex Index of the layer in upperPaths array
   * @param pathIndexOfLayer Index of the path within the current layer
   * @param pathCountOfLayer Total number of paths in the current layer
   * @param nodeIDPathCount Map of node IDs to the count of paths connected to each node
   * @param currentNodeIDPathIndex Map to track the current path index for each node ID
   * @returns Array of waypoints for SVG arrow component
   */
  createUpperPathWaypoints(
    path: PathElement,
    layerIndex: number,
    pathIndexOfLayer: number,
    pathCountOfLayer: number,
    nodeIDPathCount: { [nodeID: string]: number },
    currentNodeIDPathIndex: { [nodeID: string]: number },
  ): WayPoint[] {
    const style = this.getStyleOfPath(path, layerIndex);
    const result: WayPoint[] = [
      {
        areaID: path.outerID,
        anchorX: style.outerAnchorX,
        anchorY: style.outerAnchorY,
      },
    ];

    if (layerIndex === 0) {
      // For upperPaths[0] - connect to node layer
      // Extract node ID from pod ID (e.g., "pod-xxx" -> find which node it belongs to)
      const podID = path.innerID;
      const nodeID = this.podIDToNodeID()[podID];
      result.push({
        areaID: 'upper-spacer-0',
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Previous,
      });
      result.push({
        areaID: 'upper-spacer-0',
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Next,
      });

      if (nodeID) {
        currentNodeIDPathIndex[nodeID] =
          (currentNodeIDPathIndex[nodeID] ?? 0) + 1;
        // Add node spacer as middle point
        result.push({
          areaID: `node-${nodeID}-l`,
          anchorY: undefined,
          anchorX:
            (1 / (nodeIDPathCount[nodeID] + 1)) *
            (currentNodeIDPathIndex[nodeID] + 1),
          complementType: WaypointComplementType.Next,
        });
      }
      result.push({
        areaID: path.innerID,
        anchorX: style.innerAnchorX,
        anchorY: style.innerAnchorY,
      });
    } else {
      // For upperPaths[i] (i>0) - connect upper layers
      // Note: reversed order in display, so calculate correct spacer index
      const upperLength = this.model().upper.length;
      result.push({
        areaID: `upper-spacer-${upperLength - layerIndex}`,
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Previous,
      });
      result.push({
        areaID: `upper-spacer-${upperLength - layerIndex}`,
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Next,
      });
      result.push({
        areaID: path.innerID,
        anchorX: style.innerAnchorX,
        anchorY: style.innerAnchorY,
      });
    }

    return result;
  }

  /**
   * Creates waypoints array for lower path elements
   * @param path The path element from lowerPaths
   * @param layerIndex Index of the layer in lowerPaths array
   * @param pathIndexOfLayer Index of the path within the current layer
   * @param pathCountOfLayer Total number of paths in the current layer
   * @param nodeIDPathCount Map of node IDs to the count of paths connected to each node
   * @param currentNodeIDPathIndex Map to track the current path index for each node ID
   * @returns Array of waypoints for SVG arrow component
   */
  createLowerPathWaypoints(
    path: PathElement,
    layerIndex: number,
    pathIndexOfLayer: number,
    pathCountOfLayer: number,
    nodeIDPathCount: { [nodeID: string]: number },
    currentNodeIDPathIndex: { [nodeID: string]: number },
  ): WayPoint[] {
    const style = this.getStyleOfPath(path, layerIndex);
    const result: WayPoint[] = [
      {
        areaID: path.outerID,
        anchorX: 1 - style.outerAnchorX,
        anchorY: 1 - style.outerAnchorY,
      }, // Start point
    ];

    if (layerIndex === 0) {
      // For lowerPaths[0] - connect to node layer
      const podID = path.innerID;
      const nodeID = this.podIDToNodeID()[podID];
      result.push({
        areaID: 'lower-spacer-0',
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Previous,
      });
      result.push({
        areaID: 'lower-spacer-0',
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Next,
      });
      if (nodeID) {
        currentNodeIDPathIndex[nodeID] =
          (currentNodeIDPathIndex[nodeID] ?? 0) + 1;
        // Add node spacer as middle point
        result.push({
          areaID: `node-${nodeID}-r`,
          anchorY: undefined,
          anchorX:
            (1 / (nodeIDPathCount[nodeID] + 1)) *
            (currentNodeIDPathIndex[nodeID] + 1),
          complementType: WaypointComplementType.Next,
        });
      }
      result.push({
        areaID: path.innerID,
        anchorX: 1 - style.innerAnchorX,
        anchorY: 1 - style.innerAnchorY,
      });
    } else {
      // For lowerPaths[i] (i>0) - connect lower layers
      result.push({
        areaID: `lower-spacer-${layerIndex}`,
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Previous,
      });
      result.push({
        areaID: `lower-spacer-${layerIndex}`,
        anchorX: undefined,
        anchorY: (1 / (pathCountOfLayer + 1)) * (pathIndexOfLayer + 1),
        complementType: WaypointComplementType.Next,
      });
      result.push({
        areaID: path.innerID,
        anchorX: 1 - style.innerAnchorX,
        anchorY: 1 - style.innerAnchorY,
      });
    }

    return result;
  }

  /**
   * Computed map of Pod IDs to their parent Node IDs
   * Cached for efficient lookups during waypoint calculations
   */
  readonly podIDToNodeID = computed(() => {
    const podIDToNodeIDMap: { [podID: string]: string } = {};
    for (const node of this.model().nodes) {
      for (const pod of node.pods) {
        podIDToNodeIDMap[pod.id] = node.id;
      }
    }
    return podIDToNodeIDMap;
  });

  /**
   * All upper paths with their calculated waypoints
   */
  readonly upperPathsWithWaypoints = computed(() => {
    const model = this.model();
    const result = [];
    const podIDToNodeID = this.podIDToNodeID();
    const nodeIDPathCount: { [nodeID: string]: number } = {};
    if (model.upperPaths.length > 0) {
      for (let i = 0; i < model.upperPaths[0].length; i++) {
        const path = model.upperPaths[0][i];
        const nodeID = podIDToNodeID[path.innerID];
        nodeIDPathCount[nodeID] = (nodeIDPathCount[nodeID] ?? 0) + 1;
      }
    }
    const currentNodeIDPathIndex: { [nodeID: string]: number } = {};

    for (let i = 0; i < model.upperPaths.length; i++) {
      const layerPaths = model.upperPaths[i];
      const pathsWithWaypoints = layerPaths.map((path, pathIndexOfLayer) => ({
        path,
        waypoints: this.createUpperPathWaypoints(
          path,
          i,
          pathIndexOfLayer,
          layerPaths.length,
          nodeIDPathCount,
          currentNodeIDPathIndex,
        ),
      }));
      result.push(pathsWithWaypoints);
    }

    return result;
  });

  /**
   * All lower paths with their calculated waypoints
   */
  readonly lowerPathsWithWaypoints = computed(() => {
    const model = this.model();
    const result = [];
    const podIDToNodeID = this.podIDToNodeID();
    const nodeIDPathCount: { [nodeID: string]: number } = {};
    if (model.lowerPaths.length > 0) {
      for (let i = 0; i < model.lowerPaths[0].length; i++) {
        const path = model.lowerPaths[0][i];
        const nodeID = podIDToNodeID[path.innerID];
        nodeIDPathCount[nodeID] = (nodeIDPathCount[nodeID] ?? 0) + 1;
      }
    }
    const currentNodeIDPathIndex: { [nodeID: string]: number } = {};

    for (let i = 0; i < model.lowerPaths.length; i++) {
      const layerPaths = model.lowerPaths[i];
      const pathsWithWaypoints = layerPaths.map((path, pathIndexOfLayer) => ({
        path,
        waypoints: this.createLowerPathWaypoints(
          path,
          i,
          pathIndexOfLayer,
          layerPaths.length,
          nodeIDPathCount,
          currentNodeIDPathIndex,
        ),
      }));
      result.push(pathsWithWaypoints);
    }

    return result;
  });

  /**
   * Determines the appropriate arrow style for a path based on its type and layer
   * @param path The path element to style
   * @param layerIndex Index of the layer containing the path
   * @returns The arrow style configuration to apply
   */
  getStyleOfPath(path: PathElement, layerIndex: number): ArrowStyle {
    return layerIndex === 0
      ? this.ArrowStylesBetweenNodeAndLayer[path.type]
      : this.ArrowStylesBetweenLayers[path.type];
  }
}
