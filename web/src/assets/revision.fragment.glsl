#version 300 es
precision highp float;
precision highp int;

#define SELECTION_STATUS_FILTERED_OUT 0
#define SELECTION_STATUS_DEFAULT 1
#define SELECTION_STATUS_HIGHLIGHTED 2
#define SELECTION_STATUS_SELECTED 3

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

out vec4 outColor;

in vec2 originalPosition;

// Revision rectangle size in pixels
in vec2 actualSize;
in vec3 revisionBaseColor;
flat in int revisionIndex;
flat in int selectionStatus;

const vec4 edgeThickness = vec4(2.0f, 1.0f, 2.0f, 1.0f);
// To avoid using large empty space around digit in uv space, the uv will be scaled to use only the center part of this width.
const float fontExtractWidth = 0.58f;
const vec2 fontSize = vec2(fontExtractWidth, 1.0f) * .6f;
// Offset of revision indexes from bottom left
const vec2 fontPadding = vec2(12, 8);

const float loge10 = 2.302585092994046f;
const float epsilon = 0.00001f;

uniform float timelineHeight;
uniform float devicePixelRatio;
uniform sampler2D numberTexture;

float number(vec2 uv, int num) {
    uv.x = uv.x * fontExtractWidth + (1.f - fontExtractWidth) / 2.f;
    vec2 offset = vec2(0.1f * float(num), 0);
    vec2 size = vec2(0.1f, 1.0f);
    vec2 uvFlipY = offset + size * uv;
    return texture(numberTexture, vec2(uvFlipY.x, 1.f - uvFlipY.y)).a;
}

float log10(float x) {
    return log(x) / loge10;
}

vec3 correctGamma(vec3 linearColor) {
    return pow(linearColor, vec3(1.0f / 2.2f));
}

void main() {
    // Draw border of revision rectangle
    // edgeSize is the thickness in uv coordinate.
    vec4 edgeSize = 2.0f * edgeThickness / actualSize.yxyx * devicePixelRatio;
    float edgeScaling = 1.0f;
    vec3 baseColor = revisionBaseColor;

    vec2 uvPadding = fontPadding / actualSize;
    float digitCount = floor(log10(float(max(1, revisionIndex)) + 0.001f) + 1.f); // max is needed to avoid log10(0) (NaN)
    vec2 uvSubtractingPads = (originalPosition - uvPadding) / 2.0f + 0.5f;
    vec2 uvScale = actualSize / (fontSize * actualSize.y);
    vec2 numberUv = max(uvSubtractingPads * uvScale, vec2(0));// Shrink uv to 0 for the points exceeding digit count of the uv
    float originalUvx = numberUv.x;
    int divisor = int(pow(10.f, float(int(digitCount) - int(floor(numberUv.x)) - 1)) + epsilon);
    numberUv.x = fract(numberUv.x);
    int currentDigit = revisionIndex / divisor;
    float isDigit = number(clamp(numberUv, vec2(0), vec2(1)), currentDigit % 10);
    isDigit *= step(originalUvx, digitCount);

    float baseAlpha = 0.75f;
    vec3 digitColor = vec3(0);
    if(selectionStatus == SELECTION_STATUS_FILTERED_OUT) {
        edgeScaling = .5f;
        baseAlpha = 0.6f;
    } else if(selectionStatus == SELECTION_STATUS_HIGHLIGHTED) {
        baseAlpha = 0.85f;
        edgeScaling = 1.5f;
    } else if(selectionStatus == SELECTION_STATUS_SELECTED) {
        baseAlpha = 0.9f;
        digitColor = vec3(1);
        edgeScaling = 2.0f;
    }
    // Border become 1 on fragments on the border.
    edgeSize *= edgeScaling;
    float border = max(max(step(originalPosition.x, -1.0f + edgeSize.w), step(1.f - edgeSize.y, originalPosition.x)), // horizontal edge
    max(step(originalPosition.y, -1.0f + edgeSize.x), step(1.f - edgeSize.z, originalPosition.y)));

    outColor = mix(vec4(baseColor, baseAlpha + border), vec4(digitColor, baseAlpha * 1.3f), isDigit);
    outColor.rgb = correctGamma(outColor.rgb);
}