// Maximum number of distinct log types supported.
#define MAX_LOG_TYPE_COUNT 128
// Maximum number of distinct log severities supported.
#define MAX_LOG_SEVERITY_COUNT 8

// EventStyles contains the color definitions for log types and severities.
// This uniform block is shared across all event instances.
layout(std140) uniform EventStyles{
  vec4 logTypeColors[MAX_LOG_TYPE_COUNT];
  vec4 logSeverityColors[MAX_LOG_SEVERITY_COUNT];
} es;

// EventLayerStyles defines the visual style properties that are specific to a timeline layer.
// These properties control the layout and appearance of events within a layer.
layout(std140) uniform EventLayerStyles{
  float timelineHeight;        // The height of the timeline track.
  float verticalPadding;       // Vertical padding inside the timeline track.
  float severityColorRatio;    // The ratio of the event height dedicated to the severity color.
  float borderThickness;       // Thickness of the event border.
  float borderAntialias;       // Antialiasing factor for the border.
  vec3 hoverBorderColor;     // Color of the border when the event is hovered.
  float hoverBorderThickness;  // Thickness of the border when hovered.
  vec3 selectionBorderColor;   // Color of the border when the event is selected.
  float selectionBorderThickness; // Thickness of the border when selected.
} els;

// EventModel acts as an interface to pass event-specific data from the vertex shader to the fragment shader.
struct EventModel{
  // styles
  vec4 logTypeColor;      // Color corresponding to the log type.
  vec4 logSeverityColor;  // Color corresponding to the log severity.
  // states
  uint eventIndex;        // Unique index of the event.
  uint selectionStatus;   // Selection state: 0=None, 1=Hover, 2=Selected.
  uint filterStatus;      // Filter state: 0 = filtered out, 1 = non filtered.
};
