// Mathematical constants.
#define PI 3.141592653589793
#define SQRT2 1.414213562373095

// Bitset texture constant
#define BITSET_TEXTURE_WIDTH 2048u

// Sentinel value indicating that no log is currently selected.
#define NO_LOG_INDEX_SELECTED 0xFFFFFFFFu

// ViewState containing the global viewport and time state.
// This uniform block is shared across all renderers.
layout(std140) uniform ViewState {
    vec2 canvasResolution; // The logical resolution of the canvas (width, height).
    float devicePixelRatio; // The ratio of physical pixels to logical pixels.
    float pixelsPerMs; // The current zoom level: pixels per millisecond.
    uvec2 leftEdgeTime; // The timestamp at the left edge of the viewport. x: seconds, y: nanoseconds.
    uint selectedLogIndex; // The index of the selected log, or NO_LOG_INDEX_SELECTED (0xFFFFFFFFu) if none.
    uint _padding;
} vs;

// Checks whether a given bit is set in an R32UI bitset texture.
// id >> 5u is equivalent to integer division by 32 to determine the uint32 word index.
// id & 31u is equivalent to modulo 32 to determine the bit offset within that word.
// Texture coordinate is computed via bitwise operations: wordIndex & 2047u for x (mod 2048), wordIndex >> 11u for y (div 2048).
bool checkBitset(highp usampler2D bitsetTexture, uint id) {
    uint wordIndex = id >> 5u;
    uint bitOffset = id & 31u;
    ivec2 coord = ivec2(int(wordIndex & 2047u), int(wordIndex >> 11u));
    uint word = texelFetch(bitsetTexture, coord, 0).r;
    return (word & (1u << bitOffset)) != 0u;
}
