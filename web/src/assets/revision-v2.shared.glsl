// Maximum number of distinct revision state types supported.
#define MAX_REVISION_STATE_TYPE 128

// Texture containing MSDF font data for numbers 0-9.
uniform sampler2D numbersMSDFTexture;

// NumberMSDFAtlasParam contains UV coordinates and layout data for the number font atlas.
layout(std140) uniform NumberMSDFAtlasParam {
  vec4 digitUVs[10]; // UV bounds for each digit: [x, y, width, height]
  vec4 digitOffsets[10]; // Rendering offsets: [x_offset, y_offset, x_advance, (padding)]
} numberMSDFAtlasParam;

// Texture containing MSDF icon data.
uniform sampler2D iconsMSDFTexture;

// RevisionStyles defines visual properties for each specific revision state type.
layout(std140) uniform RevisionStyles {
    vec4 baseColors[MAX_REVISION_STATE_TYPE]; // The base color of the revision state: [r, g, b, (padding)]
    vec4 iconUVSize[MAX_REVISION_STATE_TYPE]; // UV bounds for the state's icon: [x, y, width, height]
    vec4 revisionStyles[MAX_REVISION_STATE_TYPE]; // Style flags and values: [alpha, borderStripePattern, bodyStripePattern, padding]
} rs;

// RevisionLayerStyles defines the visual style properties specific to a timeline layer.
layout(std140) uniform RevisionLayerStyles {
    float timelineHeight;              // Height of the timeline track.
    float verticalPadding;             // Vertical spacing inside the timeline track.
    float borderThickness;             // Thickness of the revision box border.
    vec2 fontPaddingInPx;              // Padding for the revision index text.
    float fontSizeInPx;                // Font size for the revision index.
    float fontThicknessBySelectionType[3]; // SDF font thickness/weight adjustment: [None, Selected, Highlighted]
    float fontAntialias;               // Antialiasing factor for the font.
    float fontStepInPx;                // Horizontal spacing per digit character.
    vec3 selectionBorderColor;         // Color of the border when selected.
    float selectionBorderThickness;    // Thickness of the border when selected.
    vec3 highlightBorderColor;         // Color of the border when highlighted (hovered).
    float highlightBorderThickness;    // Thickness of the border when highlighted.
    float iconSizeInPx;                // Size of the state icon in pixels.
    vec2 iconPaddingInPx;              // Padding for the state icon.
    float iconThicknessBySelectionType[3]; // SDF icon thickness/weight adjustment: [None, Selected, Highlighted]
    float iconAntialias;               // Antialiasing factor for the icon.
    float borderStripePitch;           // Pitch of the stripe pattern on the border.
    float bodyStripePitch;             // Pitch of the stripe pattern on the body.
} rls;

struct RevisionModel {
  // styles
  vec4 baseColor;
  vec4 iconUVSize;
  float alphaTransparency;
  float borderStripePatten;
  float bodyStripePattern;
  // states
  uint revisionIndex;
  uint revisionState;
  uint selectionStatus;
  uint filterStatus;
  uint logIndex;
};
