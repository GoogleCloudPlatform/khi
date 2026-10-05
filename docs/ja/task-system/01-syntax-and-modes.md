# KHI タスクシステムの基本文法と実行モード

[< インデックスへ戻る](../khi-task-system-concept.md) | [次へ: ログ解析のためのタスク実装パターン >](./02-log-processing-cookbook.md)

---

本ドキュメントでは、KHI のタスクアーキテクチャを理解し開発するために必須となる**タスクシステムの基本文法、`Run` と `DryRun` の実行ライフサイクルモード、およびテスト手法**について解説します。

## 1. DAG で使用される基本的な形式

DAG は、サイクルを持たずに一方向に流れる有向非巡回グラフです。KHI の文脈では、これはタスクが依存関係に基づいて特定の順序で実行されるワークフローを表します。グラフ内の各ノードはタスクであり、エッジはタスク間の依存関係を表します。

## 2. タスクの型 (`Task[T]`)

KHI のすべてのタスクには、その出力に関連付けられた「型」があります。これらは Go のジェネリクス型を使用して記述され、コンパイル時に検証されます。
以下は `int` 値を返すタスクを宣言する例です:

```go
var IntGeneratorTask = coretask.Define(
    IntGeneratorTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (int, error) {
        return func(ctx context.Context) (int, error) {
            return 1, nil
        }
    },
)
```

この例では、以下の重要な要素を宣言しています:

1. **`IntGeneratorTask` 型**: Go コンパイラはジェネリクス推論によりこのタスクの型を `coretask.Task[int]` と推論します。
2. **第一引数 (`IntGeneratorTaskID`)**: タスクグラフにおけるそのタスク実装の ID を示します。これは `taskid.TaskImplementationID[int]` 型である必要があります。この ID を使用して、他のタスクからそのタスクへの参照を取得できます。
3. **第二引数 (`bind` 関数)**: タスクの構築時に `Define` から `*coretask.Binder` が渡され、`coretask.Use`、`coretask.UseOptional`、`coretask.UseTag`、または `coretask.After` を用いて入力の依存関係を宣言します。`bind` 関数は実行クロージャである `func(ctx context.Context) (int, error)` を返し、その戻り値の型はタスクの型パラメータに準拠している必要があります。静的な定数値を返すタスクの場合は、`coretask.DefineConstant(IntGeneratorTaskID, 1)` も利用できます。

## 3. タスク内部からの値の取得

### 3.1 ポイント・ツー・ポイントの依存関係 (`coretask.Use`)

先行タスクから値を読み取るには、`*coretask.Binder` に対して `coretask.Use` でタスク参照 (`taskID.Ref()`) をバインドし、返された `coretask.Input[T]` ハンドルから実行クロージャ内で値を取得します:

```go
var DoubleIntTask = coretask.Define(
    DoubleIntTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (int, error) {
        // Binder に対して必須の入力タスク参照をバインド
        intInput := coretask.Use(b, IntGeneratorTaskID.Ref())
        return func(ctx context.Context) (int, error) {
            // 型付き入力ハンドルから戻り値を取得
            return intInput.Get(ctx) * 2, nil
        }
    },
)
```

> [!IMPORTANT]
> **すべての入力は `bind` 関数が返る前にバインドする**
> `*coretask.Binder` は外側の `bind` 関数が返った時点で封印されます。先行タスクの戻り値は `b` から取得した `Input[T]`、`OptionalInput[T]`、または `TagInput[T]` ハンドル経由でしか読み取れないため、未宣言の依存関係に実行時アクセスしてしまうミスを構造的に防げます。戻り値を読み取らず実行順序のみを制御したい場合は、`coretask.After(b, dep)` で登録します。

#### 依存関係スコープ

`taskID.Ref()` で依存関係を宣言すると、グラフリゾルバが対象タスクをどのように探索してアクティブグラフに組み込むかを制御する**依存関係スコープ**が適用されます。

- **`taskid.ScopeAll` (`coretask.FromAll`)**: 登録された全タスクプールから探索し、対象タスクを実行グラフに引き込みます。通常の**ポイント・ツー・ポイント参照のデフォルト**です。
- **`taskid.ScopeActiveFeatures` (`coretask.FromActiveFeatures`)**: 今回のインスペクションで有効化された機能に属するプロデューサタスクのみを引き込みます。**タグによるファンイン参照のデフォルト**です。
- **`taskid.ScopeActiveGraph` (`coretask.FromActiveGraph`)**: 他の依存関係によってすでにアクティブグラフに含まれているタスクにのみ遅延バインドします。新たな上流タスクを自発的にグラフへ引き込むことはありません。

必要に応じて、依存関係宣言時にスコープを明示的に指定して上書きできます。詳細は [3.4 依存関係スコープの明示指定 (`coretask.From*`)](#34-依存関係スコープの明示指定-coretaskfrom) を参照してください。

### 3.2 オプショナルな依存関係 (`coretask.UseOptional`)

ユーザーが無効化できるオプショナル機能に属するタスクのように、依存先タスクがタスクグラフに必ず含まれるとは限らない場合は、`coretask.FromActiveGraph` または `coretask.FromActiveFeatures` のスコープを指定した参照を `coretask.UseOptional` でバインドします:

```go
var SafeConsumerTask = coretask.Define(
    SafeConsumerTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (string, error) {
        // UseOptional には ScopeAll より狭いスコープである FromActiveGraph または FromActiveFeatures を指定します
        optInput := coretask.UseOptional(b, OptionalTaskID.Ref(coretask.FromActiveGraph))
        return func(ctx context.Context) (string, error) {
            if val, ok := optInput.Get(ctx); ok {
                return val, nil
            }
            return "fallback", nil
        }
    },
)
```

### 3.3 タグによるファンイン依存関係 (`coretask.UseTag`)

複数のログパーサーがログサマリーを生成する場合のように、複数のプロデューサが同一型のアイテムを生成するケースでは、`Tag[T]` と `coretask.UseTag` を使用します:

```go
// 1. contract でタグを宣言
var LogItemTag = coretask.NewTag[*LogItem]("khi.google.com/log-items")

// 2. プロデューサタスクが ProvidesTag でタグの提供を宣言。必要に応じて WithTagPriority で優先度を指定可能
var ParserTaskA = coretask.Define(
    ParserTaskAID,
    func(b *coretask.Binder) func(ctx context.Context) (*LogItem, error) {
        sourceLogs := coretask.Use(b, SourceLogRef)
        return func(ctx context.Context) (*LogItem, error) {
            return parseLogA(ctx, sourceLogs.Get(ctx))
        }
    },
    coretask.ProvidesTag(LogItemTag, coretask.WithTagPriority(10)),
)

// 3. コンシューマタスクが UseTag で全アクティブプロデューサの結果を集約取得
var AggregatorTask = coretask.Define(
    AggregatorTaskID,
    func(b *coretask.Binder) func(ctx context.Context) ([]*LogItem, error) {
        itemsInput := coretask.UseTag(b, LogItemTag.Ref())
        return func(ctx context.Context) ([]*LogItem, error) {
            return itemsInput.Get(ctx), nil
        }
    },
)
```

#### ファンインにおける循環依存と Priority による安定したグラフの実現

ファンイン集約を用いる際、以下のような前提条件が揃うとタスクグラフに循環参照が生じます:

1. **クロスインベントリ依存による前提条件**:
   監査ログパーサーやコンテナログパーサーのように複数の独立したログパーサーが存在し、それぞれが IP アドレス一覧やコンテナ ID 一覧といった異なるインベントリのプロデューサでありつつ、他方のインベントリをクエリ生成のために消費する構造を持つ場合です。単体では非循環な DAG であっても、ユーザーが両方の機能を同時に有効化した際、ファンイン集約 (`TagReference`) を介して相互依存ループが形成されます。
2. **Priority による決定論的枝刈りと安定したグラフ**:
   循環を解消するためにエッジを任意に選んで切り落とすと、実行環境やタスク登録順序によって実行順序やデータフローが変動し、再現性のない不安定なグラフになってしまいます。
   - デフォルトで `DefaultTagPriority = 100` が設定される `coretask.WithTagPriority(priority)` により、プロデューサ側がデータの確度や寄与度を宣言します。数値が小さいほど高優先度として扱われます。
   - グラフリゾルバはサイクルを形成するファンインエッジを決定論的に枝刈りし、常に安全で一意かつ安定した単一ステージの DAG を導出します。

詳細なアーキテクチャ背景は、[概念ガイド: 5. ファンインにおける循環依存の前提条件と Priority によるグラフ安定化](../khi-task-system-concept.md#5-ファンインにおける循環依存の前提条件と-priority-によるグラフ安定化) を参照してください。

### 3.4 依存関係スコープの明示指定 (`coretask.From*`)

`coretask.Use` のデフォルトである `ScopeAll` や、`coretask.UseTag` のデフォルトである `ScopeActiveFeatures` といったスコープ解決を上書きしたい場合は、参照作成時に `coretask.FromActiveGraph` や `coretask.FromActiveFeatures` のスコープオプションを渡します:

```go
var AdvancedConsumerTask = coretask.Define(
    AdvancedConsumerTaskID,
    func(b *coretask.Binder) func(ctx context.Context) (ResultType, error) {
        // アクティブグラフにすでに含まれているタグプロデューサにのみ遅延バインドする
        itemsInput := coretask.UseTag(b, LogItemTag.Ref(coretask.FromActiveGraph))

        // 今回のインスペクションで対象機能が有効な場合にのみオプショナル依存をバインドする
        optInput := coretask.UseOptional(b, OptionalTaskID.Ref(coretask.FromActiveFeatures))

        return func(ctx context.Context) (ResultType, error) {
            items := itemsInput.Get(ctx)
            optVal, _ := optInput.Get(ctx)
            return computeResult(items, optVal), nil
        }
    },
)
```

## 4. タスク内でのログ出力 (`slog`)

タスクのデバッグやエラー解析を行う際は、標準の `fmt.Println` ではなく、`slog.InfoContext` や `slog.ErrorContext` といったコンテキスト付きの構造化ロガーである `slog` を使用してログを出力します。

```go
slog.InfoContext(ctx, "processing int value", "intValue", value)
```

これにより、ログメッセージにインスペクションのトレース ID や実行時コンテキスト情報が自動的に付与され、Cloud Logging やローカルデバッグログからの追跡が容易になります。

## 5. タスクのパッケージ構造とパッケージ名規約

KHI のすべての検査タスクは、**単一機能ごとに専用のパッケージとして分離する** アーキテクチャ原則を採用しています。
各タスクは `pkg/task/inspection/<パッケージ名>/` 配下の固有のフォルダー内に定義し、さらにその配下で以下の 2 つのディレクトリへと明確に分離する必要があります:

```text
pkg/task/inspection/<パッケージ名>/
├── contract/  # 公開インターフェース・タスク ID・型定義のみを置く
└── impl/      # 実際のタスク定義やログ処理ロジックを置く
```

### 1. `contract` フォルダーの責務

- その機能が他のタスクへと公開する **タスク ID**、**インターフェース**、および構造体や列挙型といった **公開データ構造** のみを定義します。
- **実処理を含む関数や `coretask.Define(...)` によるタスク実装自体を記述してはなりません。**
- 他のあらゆるタスクパッケージからインポート可能な唯一の公開レイヤーです。

### 2. `impl` フォルダーの責務

- `contract` で定義されたタスク ID に紐づく `var SomeTask = coretask.Define(...)` や `inspectiontaskbase.DefineInspectionTask(...)` といった実際のタスク、およびパーサー・マッパーロジックの実装コードを記述します。
- `init()` や `Register(...)` といった初期化関数を配置します。
- **他の機能パッケージの `impl` をインポートすることは禁止されています。** 別の機能に依存する場合は、必ずその機能の `contract` のみをインポートし、タスク依存関係としてタスクグラフ上で連携してください。

### 3. パッケージ名 (`package` 宣言) の命名規約

Go では、単に `contract` や `impl` というパッケージ名で宣言すると、異なる機能間でインポート時の識別名が衝突したりコード上で出処が分かりにくくなります。
そのため、`contract` ディレクトリおよび `impl` ディレクトリ内の Go ソースファイルでは、**親の機能パッケージ名に `_contract` または `_impl` のサフィックスを結合したパッケージ名で宣言する** 規約を採用しています:

- **`contract/` 配下のファイル**: `package example_contract` のように `package <機能名>_contract` と宣言します。
- **`impl/` 配下のファイル**: `package example_impl` のように `package <機能名>_impl` と宣言します。

他のタスクから型や ID を参照する際は、必ずこの `<機能名>_contract` パッケージのみをインポートしてください。

## 6. インスペクションタスクの実行モード (`Run` と `DryRun`)

KHI のインスペクションタスクは、実行されるシチュエーションに応じて **`Run` モード** と **`DryRun` モード** のいずれかで呼び出されます。これはタスクグラフ実行における根本的なライフサイクル機能であり、重い解析処理と UI の軽量なインタラクションを明確に分離するために設計されています。

- **`Run` モード (`TaskModeRun`)**: ユーザーが「Start Inspection」ボタンをクリックして分析を開始した際に選択される通常実行モードです。実際のログクエリ、構文解析、履歴ファイルである KHI ファイルの生成・シリアライズを実行します。
- **`DryRun` モード (`TaskModeDryRun`)**: ユーザーが「New Inspection」画面で入力パラメータを変更したり、フォーム項目やオートコンプリート候補を動的に取得する際に実行される軽量なモードです。UI の応答性を維持するため、時間のかかるログ取得や解析処理はこのモードではスキップされます。

### 実行モードを判定する具体的なコード例

すべてのインスペクションタスクおよび低レベルタスクユーティリティは、引数として渡される `inspectioncore_contract.InspectionTaskModeType` (`taskMode`) を評価し、現在の実行モードに応じて処理を切り替える実装にします。
以下は、`DryRun` 時には重い処理を行わずに軽量な空結果や必要な UI メタデータのみを返し、`Run` モード時にのみ実際の解析処理を実行する標準的な Go 実装例です:

```go
var ExampleInspectionTask = inspectiontaskbase.DefineInspectionTask(
    ExampleInspectionTaskID,
    func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[ResultType] {
        logsInput := coretask.Use(b, SourceLogsTaskID.Ref())
        return func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (ResultType, error) {
            // 1. DryRun モードの判定: フォーム設定用や軽量実行時は、重いログ取得や解析をスキップして即座に返す
            if taskMode == inspectioncore_contract.TaskModeDryRun {
                return ResultType{}, nil
            }

            // 2. Run モード時: 実際のログ取得および時間のかかる解析・計算処理を実行する
            result, err := doHeavyAnalysis(ctx, logsInput.Get(ctx))
            if err != nil {
                return ResultType{}, err
            }
            return result, nil
        }
    },
    progress.WithTitle("Analyze source logs"),
)
```

このモードによる早期リターンパターンをすべてのタスクで一貫して適用することにより、KHI は複雑なログ分析タスクグラフを構成している場合でも、「New Inspection」画面での快適かつ高速なインタラクションを実現しています。

## 7. タスクのテスト

作成された個々のタスクやタスクグラフは、KHI が提供するテストユーティリティを使用して独立してテストできます。
ここでは `tasktest` パッケージを利用したテストについて説明します。

### 7.1 `tasktest.Run`

単一タスクの振る舞いを独立して検証する場合は `tasktest.Run` を呼び出し、上流タスクの入力値をポイント・ツー・ポイント参照向けの `tasktest.Given` やタグファンイン参照向けの `tasktest.GivenTag` で指定します。

```go
func TestIntGeneratorTask(t *testing.T) {
    res, err := tasktest.Run(t, t.Context(), IntGeneratorTask)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 1 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 1, res)
    }
}

func TestDoubleIntTask(t *testing.T) {
    res, err := tasktest.Run(
        t,
        t.Context(),
        DoubleIntTask,
        tasktest.Given(IntGeneratorTaskID.Ref(), 5),
    )
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res != 10 {
        t.Errorf("res mismatch (-want +got):\n- %v\n+ %v", 10, res)
    }
}
```

`tasktest.Run` は、対象タスクの `Binder` に宣言されたすべての必須入力が `tasktest.Given` で与えられていること、および未宣言の入力が渡されていないことを検証します。

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
