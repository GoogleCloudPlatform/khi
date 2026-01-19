#version 300 es
precision highp float;
precision highp int;

#include "v2.shared.glsl"
#include "event-v2.shared.glsl"

// Helper output for hit testing (picking).
// Stores internal IDs to identify which object is under the mouse cursor.
layout(location = 0) out uvec2 hittestID; // x: event index, y: object type (event = 2)

in vec2 uv;
in vec2 uvAfterRotation;
in float eventScreenSize;

flat in EventModel eventModel;

void main(){
  // Output the event index and the object type identifier (2 for events).
  // This allows the CPU to read the pixel under the mouse and identify the clicked event.
  if(eventModel.filterStatus == 0u){
    discard;
  }
  hittestID = uvec2(eventModel.eventIndex, 2);
}
