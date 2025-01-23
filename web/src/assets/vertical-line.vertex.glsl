#version 300 es
precision highp float;
precision highp int;

layout(location = 0) in vec3 position;

layout(std140) uniform ViewState {
    // Resolution of the canvas (not viewport)
    vec2 resolution;
    // How many pixels are used for 1ms distance
    float pixelPerTime;

    float pixelScale;
    // The time offset from the minimum log time to the left most visible time.
    float offsetToLeft;
    // The count of log types.
    float logTypeCount;
} vs; // UBO index is 0 for vs

layout(std140) uniform LineState {
    // The offset to line in time from the minimum log time.
    float lineOffsetFromLeft;

    // Thickness of the line in pixels.
    float lineThickness;

    // The color of line.
    vec4 lineColor;
} ls;

out vec2 originalPosition;

void main() {
    float centerOfLineTimeOffset = ls.lineOffsetFromLeft - vs.offsetToLeft;
    float centerOfLinePixelOffset = centerOfLineTimeOffset * vs.pixelPerTime;
    float clipSpaceX = centerOfLinePixelOffset / vs.resolution.x * 2.f - 1.f;
    float lineHalfWidth = ls.lineThickness / vs.resolution.x / 2.f;

    originalPosition = position.xy;
    gl_Position = vec4(clipSpaceX + lineHalfWidth * position.x, position.y, 0, 1);
}