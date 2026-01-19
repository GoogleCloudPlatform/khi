#version 300 es
precision highp float;
precision highp int;

#include "v2.shared.glsl"
#include "event-v2.shared.glsl"

layout(location = 0) out vec4 fragColor;

in vec2 uv;
in vec2 uvAfterRotation;
in float eventScreenSize;

flat in EventModel eventModel;

// Calculates the border intensity based on UV coordinates and screen size.
// Returns a value between 0.0 (center) and 1.0 (edge).
float isBorder(float borderThickness,float antialias){
  vec2 distFromEdge = min(uv, 1.0 - uv);
  vec2 borderThresholdUV = vec2(borderThickness, borderThickness) / eventScreenSize;
  // Use smoothstep for antialiased border edges.
  vec2 isBorderByAxis = vec2(1.0) - smoothstep(borderThresholdUV - vec2(antialias), borderThresholdUV + vec2(antialias), distFromEdge);
  return max(isBorderByAxis.x, isBorderByAxis.y);
}

void main(){
  // Calculate border factors for different states.
  float hoverBorderAlpha = isBorder(els.hoverBorderThickness,els.borderAntialias);
  float selectionBorderAlpha = isBorder(els.selectionBorderThickness,els.borderAntialias);
  float borderAlpha = isBorder(els.borderThickness,els.borderAntialias);

  // split colors to logType / logSeverity.
  // The bottom part of diamond is logTypeColor, and the top part is logSeverityColor.
  vec3 baseColor = mix(
    eventModel.logTypeColor.rgb,
    eventModel.logSeverityColor.rgb,
    step(els.severityColorRatio, uvAfterRotation.y)
  );

  // Apply borders based on selection status (Hover, Selected, or None).
  vec3 colorWithBorder = mix(
    mix(
      mix(
        baseColor,
        mix(baseColor, vec3(1.0), 0.5), // Standard border is a lighter version of the base color.
        borderAlpha
      ),
      els.hoverBorderColor,
      hoverBorderAlpha * step(0.5, float(eventModel.selectionStatus)) // Apply hover border if Hover or Selected.
    ),
    els.selectionBorderColor,
    selectionBorderAlpha * step(1.5, float(eventModel.selectionStatus)) // Apply selection border if Selected.
  );

  fragColor.rgb = colorWithBorder;
  fragColor.a = mix(0.9,1.0,borderAlpha) * mix(0.2,1.0,float(eventModel.filterStatus));
  fragColor.rgb *= fragColor.a; // Canvas expects premultiplied alpha.
}
