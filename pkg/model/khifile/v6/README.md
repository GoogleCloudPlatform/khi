# KHI File v6 Serializer Architecture

This directory (`pkg/model/khifile/v6`) contains the core serialization logic for generating the `khifile v6` binary format. The design strictly isolates the internal KHI data models from the generated Protobuf schema, ensuring type safety, high concurrency, and minimal memory footprint.

## Processing Flow

```text
+-------------------------------------------------------------+
|                     Inspection Tasks                        |
| (Concurrent generation of Logs, Timelines, Metadata, etc.)  |
+-------------------------------------------------------------+
          |                      |                      |
          v                      v                      v
+------------------+   +-------------------+  +-------------------+
|  LogAccumulator  |   |MetadataAccumulator|  |TimelineAccumulator|
|  (sync.RWMutex)  |   |  (sync.RWMutex)   |  | (Facade wrapper)  |
+------------------+   +-------------------+  +-------------------+
          |                      |                      |
          |                      v                      |
          |            [Type Switch & Fallback]         |
          v                                             v
+-------------------------------------------------------------+
|                 InternPool & IDGenerator                    |
| (Thread-safe ID assignment, Structural/String Deduplication)|
+-------------------------------------------------------------+
          |                      |                      |
          |  [Convert to pb.*]   | [Convert to pb.*]    |
          v                      v                      v
+------------------+   +-------------------+  +-------------------+
|    []*pb.Log     |   | []*pb.MetadataItem|  | []*pb.Timeline... |
+------------------+   +-------------------+  +-------------------+
          |                      |                      |
          +----------------------+----------------------+
                                 |
                                 v
                 +-------------------------------+
                 |       File Generator          |
                 | (Writes Protobuf Binary Chunks|
                 +-------------------------------+
```

## Core Components

### 1. Accumulators (`LogAccumulator`, `MetadataAccumulator`, `TimelineAccumulator`)

These act as thread-safe builders that gather data produced concurrently by multiple inspection tasks.

- **Concurrency Optimization:** They use synchronization primitives (`sync.RWMutex` or `sync.Mutex`) combined with dynamically expanding slices to guarantee efficient insertions and fast lookups. We intentionally avoid `sync.Map` to prevent severe GC overhead from interface boxing (`any`) and to avoid expensive `O(N log N)` sorting phases at the end of the run.
- **Responsibility:** They are responsible for taking an internal domain model (e.g., `log.Log`, `inspectionmetadata.Metadata`), converting it to its corresponding Protobuf message (e.g., `pb.Log`), and storing it safely until the final file generation phase. The `TimelineAccumulator` acts as a unified facade, internally orchestrating a `TimelineRegistry` and `TimelinePathPool` to properly manage aliasing and deep hierarchical paths.

### 2. `InternPool` and `IDGenerator`

To minimize the file size and memory footprint during serialization, structural data (like JSON trees in logs) and repeating strings are deduplicated using the `InternPool`.

- **`IDGenerator`:** Provides atomic, strictly sequential `uint32` IDs across different namespaces (`IDLog`, `IDString`, `IDFieldSet`). Because IDs are guaranteed to be sequential starting from 1, Accumulators can use ultra-fast slice indexing (`slice[id-1]`) instead of hash maps.
- **`InternPool`:** Reduces duplicated complex data structures (`structured.Node`) into `pb.InternedStruct` by mapping identical field structures and strings to shared `uint32` reference IDs.

### 3. Metadata Filtering and Fallback

The `MetadataAccumulator` utilizes Go's type-switching alongside the `oneof` feature in Protobuf to balance type safety and extensibility.

- **Label Filtering:** Only internal metadata flagged with `IncludeInResultBinary()` is processed and serialized into the file.
- **Strong Typing vs. Extensibility:** Known metadata types (e.g., `HeaderMetadata`, `QueryMetadata`) are strictly mapped to their dedicated Protobuf messages. If an unknown metadata type is received (e.g., from an experimental or third-party task), it safely falls back to standard JSON serialization and is stored in a generic `json_payload` string field. This ensures backward compatibility, prevents crashes, and allows the frontend to easily parse and display unrecognized metadata.
