# Rendered timeline view

Rendered 8 of 8 matched timelines (1600x460 px, 2026-09-24T01:00:00Z to 2026-09-24T02:00:00Z).

- Image URL: `http://127.0.0.1:8080/api/v1/rendered-timelines/ABCDEF.png`
- Expires at: 2026-09-24T02:30:00Z

Embed the image in Markdown (`![Timeline](http://127.0.0.1:8080/api/v1/rendered-timelines/ABCDEF.png)`) or download the PNG before it expires:

```sh
curl --fail-with-body -o timeline.png 'http://127.0.0.1:8080/api/v1/rendered-timelines/ABCDEF.png'
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
