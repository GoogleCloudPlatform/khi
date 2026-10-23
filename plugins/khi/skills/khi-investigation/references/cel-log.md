# Log CEL Reference

Log CEL expressions filter individual log entries within timelines. Logs matching the expression remain visible, while non-matching logs are hidden or dimmed. Expressions must evaluate to a boolean value (`bool`).

## Variables

The following variables are available in the log evaluation scope:

- `logType` (`string`): The type label of the log entry (e.g. `"k8s audit"`, `"container"`, `"k8s event"`). Available values for the current inspection are listed in the `Filter vocabulary` section of `search_timelines` and `search_logs` responses.
- `severity` (`int`): The integer severity level of the log entry (`INFO`, `WARNING`, `ERROR`, `FATAL`).
- `l` (`map<string, dyn>`): The root log object, exposing `l.logType` and `l.severity`.

## Severity Constants

Severity levels can be compared using standard comparison operators (`==`, `!=`, `<`, `<=`, `>`, `>=`):

| Constant | Value | Description |
| --- | --- | --- |
| `UNKNOWN` | 0 | UNKNOWN severity |
| `INFO` | 1 | INFO severity |
| `WARNING` | 2 | WARNING severity |
| `ERROR` | 3 | ERROR severity |
| `FATAL` | 4 | FATAL severity |

## Functions

### `body` / `B`

Matches contents within the log entry body against substrings or regular expressions.

- `body(value string) bool` / `B(value string) bool`:
  Checks if any field in the log body contains `value` (case-insensitive substring or regex).
- `body(values list<string>) bool` / `B(values list<string>) bool`:
  Checks if any field in the log body matches any pattern in `values`.
- `body(fieldPath string, value string) bool` / `B(fieldPath string, value string) bool`:
  Checks if the single dot-separated field `fieldPath` in the log body matches `value` (combine multiple field paths with `||`).
- `body(fieldPath string, values list<string>) bool` / `B(fieldPath string, values list<string>) bool`:
  Checks if the single dot-separated field `fieldPath` in the log body matches any pattern in `values`.

## Common Patterns

Filter by minimum severity:

```cel
severity >= WARNING
```

Filter only errors and fatal logs:

```cel
severity >= ERROR
```

Filter by specific log type:

```cel
logType == "k8s audit"
```

Search for keywords in log message:

```cel
body("OOMKilled")
```

Search within a specific JSON/structured field:

```cel
body("protoPayload.methodName", "delete")
```

Combine severity and content filters:

```cel
severity >= ERROR && body("connection refused")
```
