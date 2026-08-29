// Mathematical constants.
#define PI 3.141592653589793
#define SQRT2 1.414213562373095

// Bitset texture constant
#define BITSET_TEXTURE_WIDTH 2048u

// ViewState containing the global viewport and time state.
// This uniform block is shared across all renderers.
layout(std140) uniform ViewState {
    vec2 canvasResolution; // The logical resolution of the canvas (width, height).
    float devicePixelRatio; // The ratio of physical pixels to logical pixels.
    float pixelsPerMs; // The current zoom level: pixels per millisecond.
    uvec2 leftEdgeTime; // The timestamp at the left edge of the viewport. x: seconds, y: nanoseconds.
    uint selectedLogIndex; // The index of the selected log, or 0xFFFFFFFFu if none.
    uint _padding;
} vs;

// Checks whether a given bit is set in an R32UI bitset texture.
bool checkBitset(highp usampler2D bitsetTexture, uint id) {
    uint wordIndex = id >> 5u;
    uint bitOffset = id & 31u;
    ivec2 coord = ivec2(int(wordIndex % BITSET_TEXTURE_WIDTH), int(wordIndex / BITSET_TEXTURE_WIDTH));
    uint word = texelFetch(bitsetTexture, coord, 0).r;
    return (word & (1u << bitOffset)) != 0u;
}
