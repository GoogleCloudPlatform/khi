Language: [English](/docs/en/setup-guide/oss-kubernetes-clusters.md) | [日本語]

# OSS Kubernetesクラスタのログの可視化（Loki）

Kubernetes History Inspector (KHI) は、kube-apiserver の監査ログを使用して、様々な情報を視覚化できます。このチュートリアルでは、[kind](https://kind.sigs.k8s.io/)を介してセットアップされたKubernetes環境内で、[Loki](https://grafana.com/oss/loki/)を使用して集約された監査ログを活用し、KHI で Kubernetes リソースの状態を視覚化する方法を紹介します。

## 前提条件

下記のツールがインストールされていること

* [Docker](https://docs.docker.com/get-docker/) または [Podman](https://podman.io/getting-started/installation)
* [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
* [Helm](https://helm.sh/docs/intro/install/)
* [LogCLI](https://grafana.com/docs/loki/latest/query/logcli/getting-started/) (Lokiをクエリするため)

## 1. クラスターの構築

はじめに、監査ログを有効にした `kind` Kubernetes クラスターの作成から始めます。

### a. 監査ポリシーの作成

`kube-apiserver` は、ユーザー、管理者、およびシステムコンポーネントによって実行されたアクションを監査ログとして記録します。監査ポリシーを使用すると、`kube-apiserver` が何をログに記録するかを設定できます。`audit-policy` という新しいディレクトリに `audit-policy.yaml` というファイルを作成し、以下の内容を記述してください。

```yaml
# audit-policy/audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
# ConfigMapやSecretなど内容にセンシティブなものが含まれうる場合にはMetadataレベルとする
rules:
- level: Metadata
  resources:
  - group: "" # core API group
    resources: ["configmaps", "secrets"]

# その他のリソースではリクエストとレスポンスの内容を含める。
# これにより、KHIはより詳細な情報を提示することができる
- level: RequestResponse
```

**監査レベル:**

* `level: Metadata`: リクエストメタデータ（リクエストしたユーザー、タイムスタンプ、リソース、動詞など）を記録しますが、リクエストまたはレスポンスボディは記録しません。
* `level: RequestResponse`: リクエストメタデータだけでなく、リクエストとレスポンスのボディも記録します。このレベルは最も詳細な情報を提供します。

### b. Kind設定ファイルの作成

次に、`kind` 設定ファイル（例: `kind-config.yaml`）を作成します。クラスター構造を定義し、ポリシーファイルをマウントして監査ログのパスを指定することで監査ロギングを有効にするためのものです。

```yaml
# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  # Mount the audit policy directory into the control-plane node
  extraMounts:
  - hostPath: audit-policy/ # Directory to audit-policy.yaml on your host machine
    containerPath: /etc/kubernetes/audit # Path inside the control-plane container
    readOnly: true
  kubeadmConfigPatches:
  - |
    apiVersion: kubeadm.k8s.io/v1beta3
    kind: ClusterConfiguration
    metadata:
      name: config
    apiServer:
      extraArgs:
        # Tell the API server where the audit policy is
        audit-policy-file: "/etc/kubernetes/audit/audit-policy.yaml"
        # Tell the API server where to write audit logs
        audit-log-path: "/var/log/kubernetes/audit.log"
      extraVolumes:
        # Mount the audit policy directory into the apiserver pod
        - name: audit-config
          hostPath: /etc/kubernetes/audit
          mountPath: /etc/kubernetes/audit
          readOnly: true
          pathType: Directory
        # Mount the host log directory into the apiserver pod to write logs
        - name: audit-logs
          hostPath: /var/log/kubernetes
          mountPath: /var/log/kubernetes
          readOnly: false
          pathType: DirectoryOrCreate # Creates the directory if it doesn't exist
- role: worker
- role: worker
- role: worker
```

### c. kindクラスターの作成

下記の設定ファイルを使用して`kind`ラスターを作成します。

```bash
kind create cluster --config kind-config.yaml
```

このコマンドは、監査ロギングが構成された、1つのコントロールプレーンノードと3つのワーカーノードを持つKubernetesクラスターをブートストラップします。

## 2. (任意) Lokiのデプロイ

このステップは任意です。すでに稼働中のLokiインスタンス（セルフホスト型またはGrafana Cloud）がある場合は、Fluent Bitを構成してそこにログを送信できます。そうでない場合は、このチュートリアル用に`kind`クラスター内にシンプルなLokiインスタンスをデプロイしてください。

### a. Loki Values Fileの作成

Loki Helmチャート用に`loki-values.yaml` ファイルを作成してください。

```yaml
# loki-values.yaml
loki:
  commonConfig:
    replication_factor: 1
  schemaConfig:
    configs:
      - from: "2025-01-01"
        store: tsdb
        object_store: s3
        schema: v13
        index:
          prefix: loki_index_
          period: 24h
minio:
  enabled: true
deploymentMode: SingleBinary
singleBinary:
  replicas: 1
backend:
  replicas: 0
read:
  replicas: 0
write:
  replicas: 0
```

### b. Helmを使用したLokiのインストール

Grafana Helmレポジトリを追加してLokiをインストールします。

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install loki grafana/loki -f loki-values.yaml --namespace khi --create-namespace
```

全てのLoki Pod(MinIOを有効にしている場合はこれも含む)のステータスが `Running`になるまで待ちます。

```bash
kubectl get pods -n khi -l app.kubernetes.io/instance=loki
```

## 3. Fluent Bitのデプロイ

Kubernetesクラスター内の全ノード（コントロールプレーンを含む）からログを収集し、それらをLokiに送信するために、軽量なログの処理と転送を提供するFluent Bitを使用します。Fluent BitはDaemonSetとしてデプロイされるため、各ノード上で実行され、そのノードのログを収集・転送します。

### a. Fluent Bit Values Fileの作成

`fluentbit-values.yaml`ファイルを作成します。

```yaml
# fluentbit-values.yaml
# Ensure Fluent Bit runs on the control-plane node to access audit logs
tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
config:
  inputs: |
    [INPUT]
        Name             tail
        Path             /var/log/containers/*.log
        multiline.parser cri
        Tag              kube.*
        Read_from_Head   On
        Mem_Buf_Limit    5MB
        Skip_Long_Lines  On
    [INPUT]
        Name              tail
        Tag               kubevar.audit
        Path              /mnt/audit/audit.log
        Parser            json
        Read_from_Head    On
        DB                /var/log/flb_audit.db
        Mem_Buf_Limit     16MB
        Buffer_Max_Size   8MB
        Buffer_Chunk_Size 8MB
        Refresh_Interval  10
  filters: |
    # Filter to add Kubernetes metadata to container logs
    [FILTER]
        Name                kubernetes
        Match               kube.*
        Merge_Log           On
        Keep_Log            Off
        K8S-Logging.Parser  On
        K8S-Logging.Exclude On
    # Filter to add a 'job' label to audit logs for easier querying in Loki
    [FILTER]
        Name    modify
        Match   kubevar.audit
        Add     job      audit
  outputs: |
    [OUTPUT]
        Name            loki
        Match           *
        Host            loki-gateway.khi.svc.cluster.local
        Port            80
        Label_Keys      $job,$kubernetes_namespace_name, $kubernetes_pod_name, $kubernetes_container_name
        Tenant_Id       KHI
# Volume to mount the kind-node's audit log directory
extraVolumes:
  - name: auditlog
    hostPath:
      path: /var/log/kubernetes # Path on the node where audit logs are written
# Mount the audit log volume into the Fluent Bit container
extraVolumeMounts:
  - name: auditlog
    mountPath: /mnt/audit # Path inside the Fluent Bit container
    readOnly: true
```

### b. Helmを利用したFluent Bitのインストール

Fluent Helmレポジトリを追加してFluent Bitをインストールします。

```bash
helm repo add fluent https://fluent.github.io/helm-charts
helm repo update
helm install fluentbit fluent/fluent-bit --values fluentbit-values.yaml --namespace khi
```

全てのFluent Bitポッドのステータスが Runningになるまで待ちます。

```bash
kubectl get pods -n khi -l app.kubernetes.io/instance=fluentbit
```

## 4. サンプル監査ログの生成

検査するデータを生成するために、クラスター上でいくつかの基本的な操作を実行してみましょう。ここでは、Nginx Deploymentの作成、スケール、削除を行います。

```bash
# Create a deployment with 3 replicas
kubectl create deployment nginx --image nginx --replicas 3

# Scale up the deployment
kubectl scale deployment nginx --replicas 5

# Scale down the deployment
kubectl scale deployment nginx --replicas 1

# Delete the deployment
kubectl delete deployment nginx
```

APIサーバーは操作によって監査ログを生成し、Fluent Bitはそれらを収集してLokiに転送します。

## 5. LogCLIで監査ログをエクスポート

次に、`logcli`を使ってLokiから収集された監査ログを取得します。

### a. Lokiサービスのポートフォワード

Lokiサービスにローカルマシンからアクセスできるようにします。下記のコマンドを別のターミナルから実行し、実行中のままにしてください。

```bash
kubectl port-forward --namespace khi service/loki-gateway 8000:80
```

### b. Lokiへクエリ

`logcli`を使用してLokiから監査ログをクエリし、`audit_log_export.jsonl` という名前のファイルに保存します。`--from` および `--to` のタイムスタンプは、`kubectl` コマンドを実行した時間範囲を網羅するように調整してください。

```bash
logcli query '{job="audit"}' \
    --org-id=KHI \
    --timezone=UTC \
    --from="2025-04-08T00:00:00Z" \
    --to="2025-04-09T00:00:00Z" \
    --output=raw \
    --limit=0 \
    --addr=http://localhost:8000 \
    -q > audit_log_export.jsonl
```

**解説:**

* `'{job="audit"}'`: `job` ラベルが `audit` に設定されているログを選択します（Fluent Bitのフィルターによって追加されます）。
* `--org-id=KHI`: Fluent Bitで設定したテナントIDを指定します。
* `--timezone=UTC`: タイムスタンプにUTCタイムゾーンを使用します。
* `--from`, `--to`: クエリの時間範囲を定義します（必要に応じて調整してください）。RFC3339形式（例：YYYY-MM-DDTHH:MM:SSZ）を使用します。
* `--output=raw`: タイムスタンプやラベルなしで、ログ行のみを出力します。
* `--limit=0`: 一致するすべてのログ行を取得します（制限なし）。
* `--addr=http://localhost:8000`: ポートフォワードされたLoki Serviceのアドレスです。

>
> ここでは `--output=raw` を使用します。 これは、このチュートリアルにおいてFluent Bitが生のJSON監査ログ行を直接Lokiに送信するように設定されているためです。もしお使いのロギングパイプラインがJSONをパースし、Loki内に構造化されたメタデータとして保存している場合（例えば、LokiのJSONパーサーやパイプラインステージを使用している場合）、`--output=raw`は使用できません。その場合、Lokiをクエリし、KHIが必要とする元のJSONL形式を自分で再構築する必要があります。これは、`logcli`を別の出力形式（例えば`--output=json`）で使用し、`jq`のようなツールで結果を処理することで実現できる可能性があります。

## 6. KHIでログを可視化

最後に、エクスポートされた監査ログをKHIで検査しましょう。

### a.　KHIの実行

Docckerを使用してKHIサーバーを実行します。

```bash
# You don't need to pass Google Cloud credentials to the container.
docker run --rm -p 127.0.0.1:8080:8080 asia.gcr.io/kubernetes-history-inspector/release:latest
```

サーバーが稼働していることを示す出力が表示されるはずです。

```bash
global > INFO Initializing Kubernetes History Inspector...
global > INFO Starting Kubernetes History Inspector server...
 Starting KHI server with listening 0.0.0.0:8080
For Cloud Shell users:
        Click this address >> http://localhost:8080 << Click this address

(For users of the other environments: Access http://localhost:8080 with your browser. Consider SSH port-forwarding when you run KHI over SSH.)

```

### b. KHIのUIへアクセス

ブラウザを開いて`http://localhost:8080`へアクセスします。

### c. 新しいInspectionの作成

1. "New Inspection"ボタンをクリックします。
2. "Inspection Type"に"OSS Kubernetes Cluster" を選択します。

![new-inspection](/docs/en/images/oss/new-inspection.png)

### d. ログファイルをアップロードして実行

1. "Input Parameters"セクションの"File Upload"で"Browse"をクリックするか、作成した `audit_log_export.jsonl` ファイルをドラッグ＆ドロップします。
2. "Upload"ボタンをクリックし、ファイルのアップロードが完了するのを待ちます。
3. "Run"ボタンをクリックします。
4. 読み込みプロセスが完了するのを待ちます。

![input-param](/docs/en/images/oss/input-param.png)

### e. ビジュアリゼーションの確認

読み込みが終了したら、"Open"ボタンをクリックしてビジュアリゼーションを表示します。さまざまなビューがありますが、特にタイムラインビューを確認してください。Nginx Deploymentの作成、スケール、削除に関連するイベントが表示され、クラスターの状態が時間の経過とともにどのように変化したかを確認できるはずです。

![timeline](/docs/en/images/oss/timeline.png)

## 7. クリーンアップ

このチュートリアル中に作成されたリソースを削除するには、`kind`クラスターを削除します。

```bash
kind delete cluster
```

OSS Kubernetes クラスターと Loki を KHI と共に使用するチュートリアルはこれで終了です。これらの手順を既存のロギングインフラストラクチャに統合するために応用できます。
