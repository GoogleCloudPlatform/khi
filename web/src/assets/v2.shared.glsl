// Mathematical constants.
#define PI 3.141592653589793
#define SQRT2 1.414213562373095

// ViewState containing the global viewport and time state.
// This uniform block is shared across all V2 renderers.
layout(std140) uniform ViewState {
    vec2 canvasResolution; // The logical resolution of the canvas (width, height).
    float devicePixelRatio; // The ratio of physical pixels to logical pixels.
    float pixelsPerMs; // The current zoom level: pixels per millisecond.
    uvec2 leftEdgeTime; // The timestamp at the left edge of the viewport. x: seconds, y: nanoseconds.
} vs;
