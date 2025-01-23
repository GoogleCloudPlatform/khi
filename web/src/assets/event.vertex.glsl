#version 300 es
precision highp float;
// Vertex shader for drawing events

#define SELECTION_STATUS_FILTERED_OUT 0
#define SELECTION_STATUS_DEFAULT 1
#define SELECTION_STATUS_HIGHLIGHTED 2
#define SELECTION_STATUS_SELECTED 3

layout(location = 0) in vec3 position;
layout(location = 1) in vec3 event; // event.x = time offset from left, event.y = event type index,event.z = log severity index / log severity count
layout(location = 2) in int status; // interaction status 0 = filtered,1 = default, 2 = hover, 3 = selected

const int MAX_STATUS = 3;

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

const vec2 size = vec2(0.42f);

uniform sampler2D colorPaletteTexture;
uniform sampler2D logSeverityColorPaletteTexture;
uniform float timelineHeight;

out vec4 eventColor;
out vec4 borderColor;
out vec4 severityColor;
out vec4 severityBorderColor;
out vec2 originalPosition;
flat out int selectionStatus;

vec2 rotation2D(vec2 source, float angle) {
    float c = cos(angle);
    float s = sin(angle);
    return vec2(c * source.x - s * source.y, s * source.x + c * source.y);
}

void main() {
    // Calculate the center of event rectangle
    vec2 viewportSize = vec2(vs.resolution.x, timelineHeight);
    float screenSpaceFromLeft = (event.x - vs.offsetToLeft) * vs.pixelPerTime;
    float clipSpaceCoordinate = screenSpaceFromLeft / viewportSize.x * 2.0f - 1.0f;

    float sizeScale = 1.3f;
    if(status <= SELECTION_STATUS_DEFAULT) {
        sizeScale = 1.f;
    }
    vec2 clipSpaceHalfSize = sizeScale * size * timelineHeight / viewportSize;

    eventColor = texture(colorPaletteTexture, vec2(((event.y * 2.0f + 0.5f)) / (vs.logTypeCount * 2.0f), 0.5f));
    borderColor = texture(colorPaletteTexture, vec2((event.y * 2.0f + 1.5f) / (vs.logTypeCount * 2.0f), 0.5f));
    severityColor = texture(logSeverityColorPaletteTexture, vec2(event.z, 0.25f));
    severityBorderColor = texture(logSeverityColorPaletteTexture, vec2(event.z, 0.75f));
    originalPosition = position.xy;
    selectionStatus = status;

    // Calculate the depth of event from the importance of it.
    // Importance criteria 1: Selected > Hover > None
    // Importance criteria 2: severity.
    // This depth is calculated in the z coordinate.
    float depth = (float(MAX_STATUS - status) - event.z) / 2.0f - 1.0f;
    // Convert [-1,1] depth space to [0,1] depth space not to put points behind of timeline background.
    depth = (depth + 1.0f) / 2.0f;
    gl_Position = vec4(vec2(clipSpaceCoordinate, 0.f) + clipSpaceHalfSize * rotation2D(position.xy, 3.1415f / 4.f), depth, 1);
}
