#version 300 es
precision highp float;
precision highp int;

layout(std140) uniform LineState {
    // The offset to line in time from the minimum log time.
    float lineOffsetFromLeft;

    // Thickness of the line in pixels.
    float lineThickness;

    // The color of line.
    vec4 lineColor;
} ls;

in vec2 originalPosition;

out vec4 resultColor;

vec3 correctGamma(vec3 linearColor) {
    return pow(linearColor, vec3(1.0f / 2.2f));
}

void main() {
    resultColor = ls.lineColor;
    resultColor.rgb = correctGamma(resultColor.rgb);
}