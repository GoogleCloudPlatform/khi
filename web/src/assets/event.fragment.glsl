#version 300 es
precision highp float;
// Fragment shader for drawing events

#define SELECTION_STATUS_FILTERED_OUT 0
#define SELECTION_STATUS_DEFAULT 1
#define SELECTION_STATUS_HIGHLIGHTED 2
#define SELECTION_STATUS_SELECTED 3

in vec4 eventColor;
in vec4 borderColor;
in vec4 severityColor;
in vec4 severityBorderColor;

in vec2 originalPosition;
out vec4 outColor;
flat in int selectionStatus;

const float edgeThickness = 0.35f;
const float highlightedEdgeThickness = 0.4f;
const float selectedEdgeThickness = 0.5f;

const float severityBorderHeight = 0.2f;
const float borderThicknessBetweenSeverityAndLogType = 0.4f;

vec3 correctGamma(vec3 linearColor) {
    return pow(linearColor, vec3(1.0f / 2.2f));
}

void main() {
    float alpha = 1.0f;
    float bodySize = 1.0f - edgeThickness;
    if(selectionStatus == SELECTION_STATUS_FILTERED_OUT) {
        alpha = 0.4f;
    } else if(selectionStatus == SELECTION_STATUS_HIGHLIGHTED) {
        bodySize = 1.f - highlightedEdgeThickness;
    } else if(selectionStatus == SELECTION_STATUS_SELECTED) {
        bodySize = 1.f - selectedEdgeThickness;
    }
    float severityBorder = severityBorderHeight - originalPosition.x;
    float severityToEventRatio = step(originalPosition.y, severityBorder);
    float borderToBodyRatio = step(max(abs(originalPosition.x), abs(originalPosition.y)), bodySize);
    outColor = mix(mix(severityBorderColor, severityColor, borderToBodyRatio), mix(borderColor, eventColor, borderToBodyRatio), severityToEventRatio);
    outColor.rgb = correctGamma(outColor.rgb);
    outColor.a = alpha;
}