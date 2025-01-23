#version 300 es
precision highp float;

// Vertex shader for drawing a border on top/bottom of a timeline row.

layout(location = 0) in vec3 position;

// model space vertex position of the rect.
out vec2 originalPosition;

void main() {
    originalPosition = position.xy;
    gl_Position = vec4(position.xy, 0, 1.f);
}
