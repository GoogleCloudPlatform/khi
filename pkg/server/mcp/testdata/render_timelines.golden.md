# Rendered timeline view

Rendered 8 of 8 matched timelines (2026-09-24T01:00:00Z to 2026-09-24T02:00:00Z).
Annotations: 1 marker [1], 1 deck excerpt [E1].

- Scene ID: `scn_abcdef1234567890` (derived from `scn_parent00000000`)
- Image URL: `http://127.0.0.1:8080/api/v1/mcp-assets/ABCDEF.png`
- Interactive URL: `http://127.0.0.1:8080/render/mcp-timeline?sceneId=scn_abcdef1234567890`
- Expires at: 2026-09-24T02:30:00Z

Pass `parentSceneId: "scn_abcdef1234567890"` to `render_timelines` to refine or branch this scene, or open the Interactive URL in a browser to jump directly to pinned diffs and logs in KHI.
Embed the static PNG in Markdown (`![Timeline](http://127.0.0.1:8080/api/v1/mcp-assets/ABCDEF.png)`) or download it before it expires:

```sh
curl --fail-with-body -o timeline.png 'http://127.0.0.1:8080/api/v1/mcp-assets/ABCDEF.png'
```

## Applied filter
Enter these values in the KHI Web UI filter to see the same data.

```yaml
timelineQuery: |-
  timelineType == "Pod"
timelineExclusionQuery: ""
logQuery: |-
  severity >= WARNING
startTime: 2026-09-24T01:00:00Z
endTime: 2026-09-24T02:00:00Z
excludeTimelinesWithoutLogs: true
```
