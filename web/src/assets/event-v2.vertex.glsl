#version 300 es
precision highp float;
precision highp int;

#include "v2.shared.glsl"
#include "event-v2.shared.glsl"

// Input attributes from the vertex buffer.
// instanced rendering is used, so these attributes are per-instance (per-event).
layout(location = 0) in uvec2 time; // x: start time(s), y: start time(ns). High precision timestamp.
layout(location = 1) in uvec4 intStaticMeta; // x: eventIndex, y: logType, z: logSeverity. Static metadata.
layout(location = 2) in uvec4 intDynamicMeta; // x: selectionState. y: filterState Dynamic metadata that can change frequently.

// Outputs to the fragment shader.
out vec2 uv;                // UV coordinates for the quad (0.0 to 1.0).
out vec2 uvAfterRotation;   // UV coordinates after 45-degree rotation (used for diamond shape).
out float eventScreenSize;  // The size of the event in screen pixels.
flat out EventModel eventModel; // The event data model passed to fragment shader.

// Generates the position of the vertex based on gl_VertexID.
// This creates a standard quad (-1 to 1) without requiring a vertex buffer for position.
vec4 genQuadPosition(){
  float x = float(gl_VertexID & 1);
  float y = float((gl_VertexID >> 1) & 1);

  return vec4(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
}

// Generates a 2D rotation matrix for the given angle.
mat2 genRotationMatrix2D(float angle){
  float c = cos(angle);
  float s = sin(angle);
  return mat2(c, -s, s, c);
}

// Generates a 2D affine transformation matrix for translation and scaling.
mat3 genTranslationScaleMatrixAffineSpace2D(vec2 translation,vec2 scale){
  return mat3(
    scale.x, 0.0, 0.0,
    0.0, scale.y, 0.0,
    translation.x, translation.y, 1.0
  );
}

// Maps selection state (0=None, 1=Hover, 2=Selected) to depth priority.
// Selected items should be rendered on top.
const uint SELECTION_STATE_TO_DEPTH_PRIORITY[3] = uint[](0u,2u,1u);

void main(){
  // 1. Generate local quad geometry
  vec4 pos = genQuadPosition();
  uv = pos.xy * 0.5 + 0.5;

  // 2. Rotate the quad by 45 degrees to create a diamond shape
  pos.xy = genRotationMatrix2D(PI / 4.0) * pos.xy / SQRT2;
  uvAfterRotation = pos.xy * 0.5 + 0.5;

  // 3. Populate the EventModel to pass data to the fragment shader
  eventModel.eventIndex = intStaticMeta.x;
  eventModel.logTypeColor = es.logTypeColors[intStaticMeta.y];
  eventModel.logSeverityColor = es.logSeverityColors[intStaticMeta.z];
  eventModel.selectionStatus = intDynamicMeta.x;
  eventModel.filterStatus = intDynamicMeta.y;

  // 4. Calculate the screen size of the event
  // If the event is not selected/hovered (selectionStatus < 0.5), it is slightly smaller.
  eventScreenSize = els.timelineHeight - 2.0 * els.verticalPadding * mix(1.0,0.7, step(0.5, float(eventModel.selectionStatus)));
  vec2 scale = vec2(eventScreenSize) / vec2(vs.canvasResolution.x,els.timelineHeight);

  // 5. Calculate the position of the event on the timeline
  ivec2 leftEdgeRelativeTime = ivec2(time.xy - vs.leftEdgeTime);
  // Convert time difference to screen pixels.
  float leftEdgeXScreen = (float(leftEdgeRelativeTime.x) * 1e+3 + float(leftEdgeRelativeTime.y) * 1e-6) * vs.pixelsPerMs;
  // Convert screen pixels to Normalized Device Coordinates (NDC).
  vec2 translation = vec2(leftEdgeXScreen / vs.canvasResolution.x * 2.0 - 1.0,0.0);

  // 6. Apply translation and scaling to the rotated quad
  vec3 moved = genTranslationScaleMatrixAffineSpace2D(translation,scale) * vec3(pos.xy,1.0);

  // 7. Calculate final Depth (Z-coordinate) for correct layering
  // Importance criteria 1: active > filtered out
  // Importance criteria 2: Selected > Hover > None
  // Importance criteria 3: severity (higher severity -> closer to camera)
  // baseOffset ensures these events are rendered behind other UI elements (like revisions on Z=0).
  float baseOffset = -0.5;
  float filterPriority = - 0.1 * float(eventModel.filterStatus);
  float selectionPriority = - 0.01 * float(SELECTION_STATE_TO_DEPTH_PRIORITY[eventModel.selectionStatus]);
  float importancePriority = - 0.001 / float(MAX_LOG_SEVERITY_COUNT) * float(intStaticMeta.z);
  pos.z = filterPriority + selectionPriority + importancePriority + baseOffset;

  gl_Position = vec4(moved.xy,pos.z,1.0);
}
