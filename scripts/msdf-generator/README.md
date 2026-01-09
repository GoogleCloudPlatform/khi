# Font MSDF generator utility

KHI renders timeline with WebGL and it uses MSDF textures to render texts. This allows the renderer to draw antialiased texts with supporting bold styles.

This directory contains the utility to generate MSDF textures for the fonts used in KHI.

## Setup

The `make setup` command at the project root automatically installs required dependency and build the font texture.
Or you can run `make generate-font-atlas` at the project root to generate the font texture manually.

## Generated files and its input

This MSDF generator generates the following textures:

- Number textures (Characters 0-9) and its atlas config JSON
  - web/src/assets/roboto-number-msdf.json
  - web/src/assets/roboto-number-msdf.png
- Material symbols MSDF texture and its atlas config JSON
  - web/src/assets/material-icons-msdf.json
  - web/src/assets/material-icons-msdf.png
  - The index.js reads `zzz_generated_used_icons.json` to know which icons are used in `pkg/model/enum` and generates the material symbols MSDF texture with only the used icons. This `zzz_generated_used_icons.json` is generated from the backend code generation.
  - The MSDF texture generation requires code points of the used icons, but frontend CSS codes usually requires icon names. This script also creates `icon-codepoints.json` to map icon names to code points from frontend.
