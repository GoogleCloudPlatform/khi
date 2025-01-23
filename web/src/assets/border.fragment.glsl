#version 300 es
precision highp float;
// Fragment shader for drawing a border on top/bottom of a timeline row.

layout(std140) uniform ViewState {
    // Resolution of the canvas (not viewport)
    vec2 resolution;
    // How many pixels are used for 1ms distance
    float pixelPerTime;

    float pixelScale;
    // Offset unixtime to the left most edge.
    float offsetToLeft;

    float logTypeCount;

} vs; // UBO index is 0 for vs

uniform float timelineHeight;

// model space vertex position of the rect.
in vec2 originalPosition;
out vec4 outColor;

// thickness for border in order of top,right,bottom,left.
const vec4 thickness = vec4(0.f, 0.f, 1.0f, 0.f);
const vec4 color = vec4(0, 0, 0, 0.4f);

void main() {
    vec2 viewportResolution = vec2(vs.resolution.x, timelineHeight);
    // convert edge size scale from screen space to viewport space.
    vec4 edgeSize = 2.f / viewportResolution.yxyx * thickness;

    float border = max(max(step(originalPosition.x, -1.0f + edgeSize.w), step(1.f - edgeSize.y, originalPosition.x)), // horizontal edge
    max(step(originalPosition.y, -1.0f + edgeSize.x), step(1.f - edgeSize.z, originalPosition.y)));

    outColor = mix(vec4(0), color, border);
}
