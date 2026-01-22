#version 300 es
precision highp float;
precision highp int;

#define MAX_REVISION_INDEX_DIGITS 5

#include "v2.shared.glsl"
#include "revision-v2.shared.glsl"

flat in RevisionModel revisionModel;
in vec2 uv;
in vec2 revisionScreenSize;
flat in float leftEdgeTimeMS;

layout(location = 0) out vec4 fragColor;

// Computes the median of RGB values, used for MSDF (Multi-channel Signed Distance Field) text rendering.
float median(vec3 c){
   return max(min(c.r, c.g), min(max(c.r, c.g), c.b));
}

// Samples the number SDF texture to get the distance value for a specific digit.
float numberSDF(vec2 uv, int digit){
  vec4 digitUV = numberMSDFAtlasParam.digitUVs[digit];
  vec4 digitOffsets = numberMSDFAtlasParam.digitOffsets[digit];
  uv.y = 1. - uv.y; // Flip Y coordinate for texture sampling.
  vec2 uvInDigit = digitOffsets.xy + digitUV.xy + uv * digitUV.zw;
  vec4 msdf = texture(numbersMSDFTexture, uvInDigit);
  return median(msdf.rgb);
}

// Samples the icon SDF texture.
float iconSDF(vec2 uv, vec4 offsetSizes){
  uv.y = 1. - uv.y; // Flip Y coordinate.
  vec2 uvInIcon = offsetSizes.xy + uv * offsetSizes.zw;
  vec4 msdf = texture(iconsMSDFTexture, uvInIcon);
  return median(msdf.rgb);
}

// Converts a raw SDF value to an alpha value with antialiasing.
float sdfToAlpha(float sdf, float threshold,float antialias){
  return smoothstep(threshold-antialias, threshold+antialias, sdf);
}

// Checks if the current pixel is within the border region.
float isBorder(float borderThickness){
  vec2 distFromEdge = min(uv, 1.0 - uv);
  vec2 borderThresholdUV = vec2(borderThickness/2.0, borderThickness) / revisionScreenSize;
  vec2 isBorderByAxis = step(distFromEdge, borderThresholdUV);
  return max(isBorderByAxis.x, isBorderByAxis.y);
}

// Generates a boolean value for a diagonal stripe pattern.
float stripePattern(float pitch){
  vec2 revisionRelativeUV = uv * revisionScreenSize;
  float diagonalVal = revisionRelativeUV.x + revisionRelativeUV.y;
  float stripeVal = mod(diagonalVal, pitch);
  return step(stripeVal, pitch / 2.0);
}

// Generates a boolean value for a straight dashed pattern along the border.
float borderStripePattern(float pitch){
  vec2 revisionRelativeUV = uv * revisionScreenSize;  
  vec2 distFromEdge = min(revisionRelativeUV, revisionScreenSize - revisionRelativeUV);

  float isVerticalEdge = step(distFromEdge.x, distFromEdge.y);

  float posAlongBorder = mix(revisionRelativeUV.x, revisionRelativeUV.y, isVerticalEdge);

  float patternVal = leftEdgeTimeMS + posAlongBorder;
  float stripeVal = mod(patternVal, pitch);
  return step(stripeVal, pitch / 2.0);
}

// maskUVIn01 returns 1.0 if uv is in [0,1]^2, 0.0 otherwise.
// Used to clip font/icon rendering to their specific bounding boxes.
float maskUVIn01(vec2 uv){
  vec2 minMask = step(vec2(0.0), uv);
  vec2 maxMask = step(uv, vec2(1.0));
  return minMask.x * minMask.y * maxMask.x * maxMask.y;
}

// Renders the revision index text.
// Returns a vec2 where:
// x: Accumulate alpha of the text.
// y: Remaining X space ratio (used to determine if there is enough space for the icon).
vec2 revisionIndexFontAlphaAndRemainingXRatio(){
  float alpha = .0;
  // Start rendering from the top-right corner.
  vec2 screenPosition = vec2(1. - uv.x,uv.y) * revisionScreenSize;
  vec2 screenPositionFromFontRightBottom = screenPosition - rls.fontPaddingInPx;
  uint revisionIndex = revisionModel.revisionIndex;
  float remainingX = revisionScreenSize.x - rls.fontPaddingInPx.x;
  
  // Iterate through digits of the revision index.
  for(int i = 0; i < MAX_REVISION_INDEX_DIGITS; i++){
    uint digit = revisionIndex % uint(10);
    float fontAspectRatio = numberMSDFAtlasParam.digitUVs[digit].z / numberMSDFAtlasParam.digitUVs[digit].w;
    vec2 fontOffset = vec2(rls.fontPaddingInPx.x, rls.fontPaddingInPx.y + rls.fontStepInPx * float(i));
    vec2 localFontUV = screenPositionFromFontRightBottom / (vec2(rls.fontSizeInPx * fontAspectRatio, rls.fontSizeInPx));
    localFontUV.x = 1. - localFontUV.x;
    
    // Accumulate alpha for the current digit.
    alpha += sdfToAlpha(numberSDF(localFontUV, int(digit)), rls.fontThicknessBySelectionType[revisionModel.selectionStatus], rls.fontAntialias) * maskUVIn01(localFontUV);
    
    screenPositionFromFontRightBottom.x -= rls.fontStepInPx;
    remainingX -= rls.fontStepInPx;
    revisionIndex /= uint(10);
    if(revisionIndex == uint(0)){
      break;
    }
  }
  return vec2(alpha, remainingX / revisionScreenSize.x);
}

// Renders the revision state icon on the left side.
float revisionIconAlpha(){
  vec2 screenPositionRelativeToLeftRevision = uv * revisionScreenSize;
  vec2 screenPositionRelativeToIconLeft = screenPositionRelativeToLeftRevision - rls.iconPaddingInPx;
  float iconAspectRatio = revisionModel.iconUVSize.z / revisionModel.iconUVSize.w;
  vec2 iconUV = screenPositionRelativeToIconLeft / vec2(rls.iconSizeInPx * iconAspectRatio, rls.iconSizeInPx);
  return sdfToAlpha(iconSDF(iconUV, revisionModel.iconUVSize),rls.iconThicknessBySelectionType[revisionModel.selectionStatus], rls.iconAntialias) * maskUVIn01(iconUV);
}


void main(){
  // 1. Render Text and Icon
  vec2 fontAlphaAndRemainingX = revisionIndexFontAlphaAndRemainingXRatio();
  float fontAlpha = fontAlphaAndRemainingX.x;
  // Only render icon if there is enough space (based on remaining X from text).
  float iconAlpha = revisionIconAlpha() * step(0.5,fontAlphaAndRemainingX.y);
  
  // 2. Compute stripe patterns
  float borderStripeAlpha = mix(1.0, borderStripePattern(rls.borderStripePitch), float(revisionModel.borderStripePatten));
  float bodyStripeAlpha = mix(1.0, stripePattern(rls.bodyStripePitch), float(revisionModel.bodyStripePattern));
  float isSelected = step(1.5,float(revisionModel.selectionStatus));
  float isHovered = step(0.5, float(revisionModel.selectionStatus)) * (1.0 - isSelected);
  float borderScale = 1.0 + isSelected * 0.5 + isHovered * 0.1;
  float borderAlpha = isBorder(rls.borderThickness * borderScale) * borderStripeAlpha;
  float alphaScale = 1.0 + isSelected * 0.3 + isHovered * 0.1;
  float baseAlpha = revisionModel.alphaTransparency * alphaScale * mix(0.5, 1.0, bodyStripeAlpha);

  // 3. Combine Alphas for "Dark" elements (Text, Icon, Border)
  float darkAlpha = max(fontAlpha, max(iconAlpha, borderAlpha));
  
  // 4. Compose Final Color
  vec3 baseColor = revisionModel.baseColor.rgb;
  fragColor.rgb = mix(baseColor, baseColor * 0.6, max(fontAlpha, iconAlpha)); // Darken text and icon
  fragColor.a = mix(baseAlpha, 1.0, darkAlpha) * mix(0.2, 1.0, float(revisionModel.filterStatus));
  fragColor.rgb *= fragColor.a; // Pre-multiplied alpha
}
