# Cloud Service Mesh (CSM) インスペクションタスク

このパッケージ (`googlecloudlogcsm`) は、Cloud Service Mesh (CSM) のトラフィックログを検査するためのタスクを含んでいます。ログのクエリ、Envoy 関連フィールドのパース、および KHI タイムラインイベントへのマッピングを処理します。

## タスクの概要

CSM インスペクションパイプラインは以下を処理します:

1. **入力**: Envoy レスポンスフラグなどのユーザー指定フィルターの取得。
2. **ログ取得**: Cloud Logging から CSM トラフィックログを取得。
3. **インベントリ**: Kubernetes Event ログおよび Audit ログから NEG (Network Endpoint Group) と BackendService のマッピングを抽出。
4. **パース & マッピング**: ログフィールドを読み取り、リソースタイムラインへマッピング。

### インベントリタスク

これらのタスクは、トラフィックログ内に直接存在しないものの、適切なリソースマッピングに必要な関連付けを発見するために使用されます。共通の Google Cloud コンポーネントおよびログプロバイダーパッケージから提供されます。

- **`EventLogNEGDiscoveryTask`** (`googlecloudlogk8sevent` 内): Kubernetes Event ログをパースして NEG と BackendService のマッピングを発見。
- **`AuditLogNEGDiscoveryTask`** (`googlecloudlogk8saudit` 内): Kubernetes Audit ログ (リソースマニフェスト経由) をパースして NEG と BackendService のマッピングを発見。
- **`NEGToBackendServiceInventoryTask`** (`googlecloudk8scommon` 内): 発見結果を統合された 1 つのインベントリマップに集約。

### CSM トラフィックログパイプライン

- **`InputCSMResponseFlagsTask`**: Envoy レスポンスフラグによるログフィルタリング用フォーム入力。
- **`ListLogEntriesTask`**: Cloud Logging から CSM トラフィックログを取得。
- **`LogIngesterTask`**: ログを最終的な KHI 履歴に登録。
- **`LogGrouperTask`**: レポーター Pod ごとにログをグループ化。
- **`LogToTimelineMapperTask`**: 正確なサービス関連付けのために NEG インベントリを利用して、CSM トラフィックログイベントをリソースタイムラインにマッピング。

## タスク関係図

```mermaid
graph TD
    classDef input fill:#e0f7fa,stroke:#006064,stroke-width:2px;
    classDef query fill:#fff3e0,stroke:#e65100,stroke-width:2px;
    classDef pipeline fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px;
    classDef inventory fill:#fff9c4,stroke:#fbc02d,stroke-width:2px;

    classDef external fill:#f5f5f5,stroke:#9e9e9e,stroke-width:2px,stroke-dasharray: 5 5;

    %% 外部タスク
    EventLogs[Event Log Provider]:::external
    ManifestGen[Manifest Generator]:::external

    %% インベントリ
    EventLogs --> EventDiscovery[EventLogNEGDiscoveryTask]:::inventory
    ManifestGen --> AuditDiscovery[AuditLogNEGDiscoveryTask]:::inventory
    EventDiscovery --> Inventory[NEGToBackendServiceInventoryTask]:::inventory
    AuditDiscovery --> Inventory

    %% CSM トラフィックログパイプライン
    FlagsInput[InputCSMResponseFlagsTask]:::input
    FlagsInput --> ListLogs[ListLogEntriesTask]:::query
    ListLogs --> Ingester[LogIngesterTask]:::pipeline
    ListLogs --> Grouper[LogGrouperTask]:::pipeline
    Grouper --> Mapper[LogToTimelineMapperTask]:::pipeline
    Ingester --> Mapper
    Inventory --> Mapper
```
