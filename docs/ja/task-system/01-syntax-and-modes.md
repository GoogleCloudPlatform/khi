# KHI タスクシステムの基本文法と実行モード

[< インデックスへ戻る](../khi-task-system-concept.md) | [次へ: ログ解析のためのタスク実装パターン >](./02-log-processing-cookbook.md)

---

本ドキュメントでは、KHI のタスクアーキテクチャを理解し開発するために必須となる**タスクシステムの基本文法、実行ライフサイクルモード（`Run` と `DryRun`）、およびテスト手法**について解説します。

## 1. DAG で使用される基本的な形式

DAG (Directed Acyclic Graph: 有向非巡回グラフ) は、サイクルを持たずに一方向に流れるグラフです。KHI の文脈では、これはタスクが依存関係に基づいて特定の順序で実行されるワークフローを表します。グラフ内の各ノードはタスクであり、エッジはタスク間の依存関係を表します。

## 2. タスクの型 (`Task[T]`)

KHI のすべてのタスクには、その出力に関連付けられた「型」があります。これらは Go のジェネリクス型を使用して記述され、コンパイル時に検証されます。
以下は `int` 値を返すタスクを宣言する例です:

```go
var IntGeneratorTask = task.NewTask(
    IntGeneratorTaskID,
    []taskid.UntypedTaskReference{},
    func(ctx context.Context) (int, error) {
        return 1, nil
    },
)
```

この例では、以下の重要な要素を宣言しています:

1. **`IntGeneratorTask` 型**: Go コンパイラはジェネリクス推論によりこのタスクの型を `task.Task[int]` と推論します。
2. **第一引数 (`IntGeneratorTaskID`)**: タスクグラフにおけるそのタスク実装の ID を示します。これは `taskid.TaskImplementationID[int]` 型である必要があります。この ID を使用して、他のタスクからそのタスクへの参照を取得できます。
3. **第二引数 (`[]taskid.UntypedTaskReference`)**: このタスクが依存するタスク参照のリストです。タスクグラフの順序付けや、タスクグラフ内の他のタスクから値を読み取る際に使用します。
4. **第三引数 (実行関数)**: この関数の戻り値はタスクの型パラメータ（この場合は `int`）に準拠している必要があり、かつ戻り値の第二引数として常にエラーを返す必要があります。

## 3. タスク内部からの値の取得

タスクから値を読み取るには、読み取り先のタスクへのタスク参照を依存関係リストに含める必要があります。
これにより、タスク実行関数から `coretask.GetTaskResult(ctx, dependencyTaskRef)` を使用して依存タスクからの戻り値を安全に取得できます:

```go
var DoubleIntTask = task.NewTask(
    DoubleIntTaskID,
    []taskid.UntypedTaskReference{IntGeneratorTaskID.Ref()}, // 依存するタスク参照を指定
    func(ctx context.Context) (int, error) {
        // コンテキストと参照IDを渡して戻り値を取得
        value := coretask.GetTaskResult(ctx, IntGeneratorTaskID.Ref())
        return value * 2, nil
    },
)
```

> [!IMPORTANT]
> **未宣言の依存関係へのアクセス禁止**
> 依存関係リストに宣言していないタスクに対して `coretask.GetTaskResult` を呼び出すと、実行時パニック (`panic`) が発生します。必ず第二引数の依存関係リストにアクセス対象のタスク参照を含めてください。

## 4. タスク内でのログ出力 (`slog`)

タスクのデバッグやエラー解析を行う際は、標準の `fmt.Println` ではなく、コンテキスト付きの構造化ロガーである `slog` (例: `slog.InfoContext`, `slog.ErrorContext` など) を使用してログを出力します。

```go
slog.InfoContext(ctx, "processing int value", "intValue", value)
```

これにより、ログメッセージにインスペクションのトレース ID や実行時コンテキスト情報が自動的に付与され、Cloud Logging やローカルデバッグログからの追跡が容易になります。

## 5. タスクのパッケージ構造とパッケージ名規約

KHI のすべての検査タスクは、**単一機能ごとに専用のパッケージとして分離する** アーキテクチャ原則を採用しています。
各タスクは `pkg/task/inspection/<パッケージ名>/` 配下の固有のフォルダー内に定義し、さらにその配下で以下の 2 つのディレクトリへと明確に分離する必要があります:

```text
pkg/task/inspection/<パッケージ名>/
├── contract/  # 公開インターフェース・タスク ID・型定義のみを置く (実処理は書かない)
└── impl/      # 実際のタスク定義やログ処理ロジック (実装) を置く
```

### 1. `contract` フォルダーの責務

- その機能が他のタスクへと公開する **タスク ID**、**インターフェース**、および **公開データ構造（構造体や列挙型など）** のみを定義します。
- **実処理を含む関数やタスク実装自体（`task.NewTask(...)`）を記述してはなりません。**
- 他のあらゆるタスクパッケージからインポート可能な唯一の公開レイヤーです。

### 2. `impl` フォルダーの責務

- `contract` で定義されたタスク ID に紐づく実際のタスク（`var SomeTask = task.NewTask(...)` 等）や、パーサー・マッパーロジックの実装コードを記述します。
- `init()` や初期化関数 (`Register(...)` 等) を配置します。
- **他の機能パッケージの `impl` をインポートすることは禁止されています。** 別の機能に依存する場合は、必ずその機能の `contract` のみをインポートし、タスク依存関係としてタスクグラフ上で連携してください。

### 3. パッケージ名 (`package` 宣言) の命名規約

Go では、単に `contract` や `impl` というパッケージ名で宣言すると、異なる機能間でインポート時の識別名が衝突したりコード上で出処が分かりにくくなります。
そのため、`contract` ディレクトリおよび `impl` ディレクトリ内の Go ソースファイルでは、**親の機能パッケージ名に `_contract` または `_impl` のサフィックスを結合したパッケージ名で宣言する** 規約を採用しています:

- **`contract/` 配下のファイル**: `package <機能名>_contract` (例: `package example_contract`)
- **`impl/` 配下のファイル**: `package <機能名>_impl` (例: `package example_impl`)

他のタスクから型や ID を参照する際は、必ずこの `<機能名>_contract` パッケージのみをインポートしてください。

## 6. インスペクションタスクの実行モード (`Run` と `DryRun`)

KHI のインスペクションタスクは、実行されるシチュエーションに応じて **`Run` モード** と **`DryRun` モード** のいずれかで呼び出されます。これはタスクグラフ実行における根本的なライフサイクル機能であり、重い解析処理と UI の軽量なインタラクションを明確に分離するために設計されています。

- **`Run` モード (`TaskModeRun`)**: ユーザーが「Start Inspection」ボタンをクリックして分析を開始した際に選択される通常実行モードです。実際のログクエリ、構文解析、履歴ファイル（KHI ファイル）の生成・シリアライズを実行します。
- **`DryRun` モード (`TaskModeDryRun`)**: ユーザーが「New Inspection」画面で入力パラメータを変更したり、フォーム項目やオートコンプリート候補を動的に取得する際に実行される軽量なモードです。UI の応答性を維持するため、時間のかかるログ取得や解析処理はこのモードではスキップされます。

### 実行モードを判定する具体的なコード例

すべてのインスペクションタスク（または低レベルタスクユーティリティ）は、引数として渡される `inspectioncore_contract.InspectionTaskModeType` (`taskMode`) を評価し、現在の実行モードに応じて処理を切り替える実装にします。
以下は、`DryRun` 時には重い処理を行わずに軽量な空結果や必要な UI メタデータのみを返し、`Run` モード時にのみ実際の解析処理を実行する標準的な Go 実装例です:

```go
var ExampleInspectionTask = inspectiontaskbase.NewProgressReportableInspectionTask(
    ExampleInspectionTaskID,
    []taskid.UntypedTaskReference{SourceLogsTaskID.Ref()},
    func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (ResultType, error) {
        // 1. DryRun モードの判定: フォーム設定用や軽量実行時は、重いログ取得や解析をスキップして即座に返す
        if taskMode == inspectioncore_contract.TaskModeDryRun {
            return ResultType{}, nil
        }

        // 2. Run モード時: 実際のログ取得および時間のかかる解析・計算処理を実行する
        logs := coretask.GetTaskResult(ctx, SourceLogsTaskID.Ref())
        result, err := doHeavyAnalysis(ctx, logs, progress)
        if err != nil {
            return ResultType{}, err
        }
        return result, nil
    },
)
```

この「モードによる早期リターン（Early Return）」パターンをすべてのタスクで一貫して適用することにより、KHI は複雑なログ分析タスクグラフを構成している場合でも、「New Inspection」画面での快適かつ高速なインタラクションを実現しています。

## 7. タスクのテスト

作成された個々のタスクやタスクグラフは、KHI が提供するテストユーティリティを使用して独立してテストできます。
ここでは `tasktest` パッケージを利用したテストについて説明します。

### 7.1 `tasktest.RunTask`

最もシンプルに単一タスクの振る舞いを検証する場合、`tasktest.RunTask` を呼び出すことができます。

```go
func TestIntGeneratorTask(t *testing.T) {
    res, err := tasktest.RunTask(t.Context(), IntGeneratorTask, map[taskid.UntypedTaskImplementationID]any{})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 1 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 1, res)
    }
}
```

第三引数には、依存タスクからの戻り値をモックアップするマップを渡すことができます。
依存関係を持たないタスクの場合は、空のマップを指定します。

### 7.2 `tasktest.RunTaskWithDependency`

依存関係をモックするのではなく、依存先を含むグラフ全体の実行を検証したい場合は、`tasktest.RunTaskWithDependency` を使用します。
これにより、依存タスクを含むミニタスクグラフが自動で組み立てられ、トポロジカルソート・実行されます。

```go
func TestDoubleIntTaskWithDependency(t *testing.T) {
    res, err := tasktest.RunTaskWithDependency(t.Context(), DoubleIntTask, []coretask.UntypedTask{
        IntGeneratorTask,
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 2 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 2, res)
    }
}
```

---

[< インデックスへ戻る](../khi-task-system-concept.md) | [次へ: ログ解析のためのタスク実装パターン >](./02-log-processing-cookbook.md)
