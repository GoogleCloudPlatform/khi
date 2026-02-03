#version 300 es
precision highp float;
precision highp int;

#define MAX_REVISION_INDEX_DIGITS 5
#define MIN_LEFT_REVISION_LOCATION -300.0

#include "v2.shared.glsl"
#include "revision-v2.shared.glsl"

// Input attributes from the vertex buffer.
layout(location = 0) in uvec4 time; // x: start time(s), y: start time(ns), z: end time(s), w: end time(ns)
layout(location = 1) in uvec4 intStaticMeta; // x: revisionIndex, y: revisionState, z: logIndex. Static metadata.
layout(location = 2) in uvec4 intDynamicMeta; // x: selectionState, y: filterStatus. Dynamic metadata.

// Outputs to the fragment shader.
out vec2 uv;                // UV coordinates for the quad (0.0 to 1.0).
out vec2 revisionScreenSize; // The dimensions of the revision box in pixels.
flat out RevisionModel revisionModel; // The revision data model passed to fragment shader.
flat out float leftEdgeTimeMS; // The timestamp at the left edge of the revision in ms (used for stripe patterns).

// Generates the position of the vertex based on gl_VertexID.
vec4 genQuadPosition(){
  float x = float(gl_VertexID & 1);
  float y = float((gl_VertexID >> 1) & 1);

  return vec4(x * 2.0 - 1.0, y * 2.0 - 1.0, 0.0, 1.0);
}

void main(){
  vec4 pos = genQuadPosition();
  
  // Populate the RevisionModel to pass data to the fragment shader.
  revisionModel.baseColor = rs.baseColors[intStaticMeta.y];
  revisionModel.iconUVSize = rs.iconUVSize[intStaticMeta.y];
  revisionModel.alphaTransparency = rs.revisionStyles[intStaticMeta.y].x;
  revisionModel.borderStripePatten = rs.revisionStyles[intStaticMeta.y].y;
  revisionModel.bodyStripePattern = rs.revisionStyles[intStaticMeta.y].z;
  revisionModel.revisionIndex = intStaticMeta.x;
  revisionModel.revisionState = intStaticMeta.y;
  revisionModel.logIndex = intStaticMeta.z;
  revisionModel.selectionStatus = intDynamicMeta.x;
  revisionModel.filterStatus = intDynamicMeta.y;

  uv = pos.xy * 0.5 + 0.5;

  // Calculate the Y scale regarding the padding of the timeline.
  // Shrink height slightly if not selected/hovered to create visual separation.
  float targetHeight = rls.timelineHeight - 2.0 * rls.verticalPadding * mix(1.0,0.0, step(0.5, float(revisionModel.selectionStatus)));
  float scaleY = targetHeight / rls.timelineHeight;
  pos.y *= scaleY;

  // Calculate the X position and width regarding the left edge time and the duration.
  // 1. Calculate relative time difference from the viewport left edge.
  // Caps the time.xy with the time calculated from the screen space value not to subtract very small value and cause float precision issue on uv.
  // We can ignore the edge case that minTime could be negative because vs.leftEdgeTime should be enough larger than 0.(It's unix time)
  uvec2 minTime = vs.leftEdgeTime - uvec2(MIN_LEFT_REVISION_LOCATION / float(vs.pixelsPerMs) / 1000.0, 1e10-1.0); // y component should always larger than time.y
  uvec2 cappedTime = max(time.xy, minTime);
  ivec2 leftEdgeRelativeTime = ivec2(cappedTime.xy - vs.leftEdgeTime);
  ivec2 durationTime = ivec2(time.zw - cappedTime.xy);
  
  // 2. Convert time to screen coordinates (pixels).
  float leftEdgeXScreen = (float(leftEdgeRelativeTime.x)* 1e+3 + float(leftEdgeRelativeTime.y) * 1e-6) * vs.pixelsPerMs;
  float durationXScreen = (float(durationTime.x)* 1e+3 + float(durationTime.y) * 1e-6) * vs.pixelsPerMs;
  
  // 3. Convert screen coordinates to Normalized Device Coordinates (NDC).
  float leftEdgeClip = leftEdgeXScreen / vs.canvasResolution.x * 2.0 - 1.0;
  float durationClip = durationXScreen / vs.canvasResolution.x * 2.0;
  
  // 4. Transform the vertex position.
  pos.x = leftEdgeClip + durationClip * (pos.x * 0.5 + 0.5);
  
  // 5. Pass screen size and time information to fragment shader.
  revisionScreenSize = vec2(durationXScreen, targetHeight);
  leftEdgeTimeMS = float(leftEdgeRelativeTime.x)* 1e+3 + float(leftEdgeRelativeTime.y) * 1e-6;
  
  // Z=0.0 is the default depth for revisions.
  gl_Position = pos;
}
