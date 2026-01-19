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


// Output for picking / hit testing.
layout(location = 0) out uvec2 hittestID; // x: revision index, y: object type (1 for revision)

void main(){
  if(revisionModel.filterStatus == 0u){
    discard;
  }
  // Return the revision index and object type 1 (Revision).
  hittestID = uvec2(revisionModel.revisionIndex, 1);
}
