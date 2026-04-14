---
trigger: glob
globs: **/*.scss
---

# SCSS Coding rules

1. Do not use color literal directly specified to CSS field like `background-color: #FF00FF;`.
   Define semantically meaningful color variables at the top of the SCSS file to use the color like background-color: $dialog-background-color;.
2. Use color palette from Material with mat.m2-get-color-from-palette rather than specifying color codes.
3. Prefer display: grid rather than display: flex. Use grid-template field rather than specifying grid-template-areas, grid-template-columns or grid-template-rows separately. **When using `grid-template`, you MUST define area names for all grid tracks.**
   grid-template field must be formatted with table like format:
   grid-template: 'foo bar' 1fr
   'qux quux' 1fr
   / 1fr 1fr; // <- the column widths must be after a line break of the last grid area row.
4. **DO NOT** use `repeat()` in grid layouts unless the number of elements is dynamic or unknown.
5. KHI's color scheme is light theme.
