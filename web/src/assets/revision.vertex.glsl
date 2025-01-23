#version 300 es
precision highp float;
precision highp int;

layout(location = 0) in vec3 position;
layout(location = 1) in vec2 duration; // duration.x is revision beginning point, and duration.y is the end point.
layout(location = 2) in vec2 meta; // meta.x: revision state
layout(location = 3) in ivec2 intInstanceInfo; // intInstanceInfo.x = revisionIndex; intInstanceInfo.y = (0:filtered, 1: default, 2: highlight, 3:selected)

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
uniform float revisionStateCount;

out vec2 originalPosition;
out vec2 actualSize;
out vec3 revisionBaseColor;
flat out int revisionIndex;
flat out int selectionStatus;
const vec2 padding = vec2(0.f, 2.f);

uniform sampler2D revisionColorPalette;

void main() {
    highp vec2 timeFromLeft = duration - vs.offsetToLeft;
    highp vec2 pixelFromLeft = timeFromLeft * vs.pixelPerTime;
    highp vec2 clipSpaceFromLeft = pixelFromLeft / vs.resolution.x;
    highp vec2 clipSpaceCoordinate = clipSpaceFromLeft * 2.f - 1.f;

    originalPosition = position.xy;
    actualSize = vec2((clipSpaceCoordinate.y - clipSpaceCoordinate.x) / 2.f * vs.resolution.x, timelineHeight) - padding;
    highp vec2 clipSpacePadding = 2.f * padding / vec2(vs.resolution.x, timelineHeight);

    revisionBaseColor = texture(revisionColorPalette, vec2((meta.x * 2.f + 0.5f) / (2.f * revisionStateCount), 0.5f)).xyz;
    revisionIndex = intInstanceInfo.x;
    selectionStatus = intInstanceInfo.y;
    gl_Position = vec4(mix(clipSpaceCoordinate.x, clipSpaceCoordinate.y, (position.x + 1.f) / 2.f) - position.x * clipSpacePadding.x, position.y - position.y * clipSpacePadding.y, 0, 1.f);
}
