# hashctl

`hashctl`は、ファイルのSHA-256を計算するCLI兼HTTPサービスです。
Goで実装したステートレスなサービスをコンテナ化し、Kustomizeを使ってkind上のKubernetesへデプロイする練習を目的に作成しました。

## できること

- ローカルファイルのSHA-256をCLIで計算する
- HTTP APIへファイルを送り、サーバー側でSHA-256を計算する
- ファイルを保存せず、ストリームとして処理する
- health/readinessエンドポイントを提供する
- 非root・読み取り専用ファイルシステムのコンテナとして実行する
- Kubernetes上で2つのPodを稼働させ、Service経由で利用する

## 構成

```text
hashctl CLI
    │ POST /hash
    ▼
Kubernetes Service :80
    │ EndpointSliceに登録されたReadyなPodへ転送
    ▼
hashctl Pod :8080 × 2
```

Serviceは固定された接続先を提供し、EndpointSliceは現在利用できるPodのIPアドレスとポートを管理します。Podが置き換わってIPアドレスが変化しても、クライアントは同じServiceを利用できます。

## 必要なもの

- Go 1.26以上
- Docker
- kind
- kubectl

## CLIの使い方

### ローカルで計算する

```bash
go run . local README.md
```

出力例：

```json
{"filename":"README.md","size":1234,"sha256":"..."}
```

### HTTPサービスを起動する

```bash
go run . serve --listen :8080
```

別のターミナルから利用します。

```bash
curl http://localhost:8080/healthz

go run . remote \
  --server http://localhost:8080 \
  README.md
```

### HTTP API

| メソッド | パス | 用途 |
| --- | --- | --- |
| `POST` | `/hash` | multipartの`file`フィールドを受け取り、SHA-256を返す |
| `GET` | `/healthz` | プロセスの生存確認 |
| `GET` | `/readyz` | リクエストを受け付けられるかの確認 |

アップロード上限は環境変数`MAX_UPLOAD_BYTES`で指定します。未指定時は10 MiBです。

## テスト

```bash
go test ./...
```

テストでは、通常ファイル、空ファイル、読み込みエラー、引数エラー、HTTP API、アップロード上限、remote CLIを確認しています。HTTPクライアントを差し替えることで、実際の待受ポートを使わずに通信をテストします。

## Docker

イメージをビルドします。

```bash
docker build -t hashctl:dev .
```

権限を制限して起動します。

```bash
docker run --rm \
  --name hashctl \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges \
  -p 8080:8080 \
  hashctl:dev
```

Dockerfileはマルチステージビルドを使用します。Goのビルド環境で静的バイナリを作り、最終的な`scratch`イメージには`/hashctl`だけを配置します。

## Kubernetesへデプロイする

### 1. kindクラスタを作成する

```bash
kind create cluster --name hashctl-practice
```

### 2. イメージをkindへ読み込む

```bash
docker build -t hashctl:dev .
kind load docker-image hashctl:dev --name hashctl-practice
```

### 3. Kustomizeで適用する

```bash
kubectl apply -k deploy/overlays/kind
kubectl rollout status deployment/hashctl -n hashctl
kubectl get all -n hashctl
```

`deploy/base`には共通のNamespace、ConfigMap、Deployment、Serviceがあります。`deploy/overlays/kind`では、Deploymentが利用するイメージをローカルの`hashctl:dev`へ差し替えます。

Deploymentには以下を設定しています。

- replicas: 2
- readiness/liveness probe
- CPU・メモリのrequests/limits
- 非rootユーザー
- 権限昇格の禁止
- Linux capabilitiesの削除
- 読み取り専用root filesystem
- SIGTERMを受けた際のgraceful shutdown

### 4. Serviceへ接続する

```bash
kubectl port-forward -n hashctl service/hashctl 8080:80
```

別のターミナルから利用します。

```bash
go run . remote \
  --server http://localhost:8080 \
  README.md
```

`port-forward`は転送先のPodを1つ選ぶ開発・調査用のトンネルです。選択されたPodを削除すると、他のPodが正常でも接続が切れるため、再実行が必要です。

## Kubernetesの状態確認

```bash
kubectl get deployment,replicaset,pod,service,configmap \
  -n hashctl \
  -o wide

kubectl get endpointslice \
  -n hashctl \
  -l kubernetes.io/service-name=hashctl \
  -o wide
```

クラスタ内部のService経路は、Kubernetes API経由でも確認できます。

```bash
kubectl get --raw \
  "/api/v1/namespaces/hashctl/services/http:hashctl:80/proxy/healthz"
```

## 障害演習

### Pod削除と自己修復

```bash
kubectl get pods -n hashctl -w
```

別のターミナルでPodを1つ削除します。

```bash
kubectl delete pod \
  -n hashctl \
  "$(kubectl get pods -n hashctl \
    -l app.kubernetes.io/name=hashctl \
    -o jsonpath='{.items[0].metadata.name}')"
```

ReplicaSetが新しいPodを作成し、Deploymentで指定した2個へ戻ることを確認します。

### ImagePullBackOff

存在しないイメージへ更新します。

```bash
kubectl set image deployment/hashctl \
  hashctl=hashctl:does-not-exist \
  -n hashctl

kubectl get pods -n hashctl
kubectl describe pod -n hashctl <ImagePullBackOffのPod名>
kubectl get events -n hashctl --sort-by=.lastTimestamp
```

復旧します。

```bash
kubectl rollout undo deployment/hashctl -n hashctl
kubectl rollout status deployment/hashctl -n hashctl
```

### readiness probeの失敗

readiness probeを存在しないパスへ変更します。

```bash
kubectl patch deployment hashctl \
  -n hashctl \
  --type=json \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/httpGet/path","value":"/not-ready"}]'
```

新しいPodが`Running`かつ`0/1`となり、EndpointSliceでは`ready: false`になることを確認します。

```bash
kubectl get pods -n hashctl
kubectl describe pod -n hashctl <0/1のPod名>
kubectl get endpointslice -n hashctl -o yaml
```

復旧します。

```bash
kubectl rollout undo deployment/hashctl -n hashctl
kubectl rollout status deployment/hashctl -n hashctl
```

## 本番運用に向けた追加事項

このプロジェクトはローカル学習用です。本番サービスとして運用する場合は、少なくとも以下を検討します。

- 認証・認可と通信の暗号化
- Prometheus形式のメトリクスとアラート
- 構造化ログとリクエストID
- NetworkPolicy
- コンテナイメージの脆弱性検査と署名
- CIによるテスト、イメージビルド、manifest検証
- CD/GitOpsによる変更反映とロールバック
- 負荷試験に基づくresourcesとアップロード上限の調整

## 学んだこと

- `io.Reader`を使うと、ローカルファイルとHTTPアップロードで同じハッシュ計算処理を再利用できる
- ステートレスな設計は複数Replicaへ水平展開しやすい
- Deploymentは宣言されたReplica数を維持する
- readiness probeに失敗したPodは、起動中でもServiceの通常の転送対象から外れる
- Serviceは固定の窓口を提供し、EndpointSliceが変化するPodの接続先を管理する
- `Running`だけでなく、Ready状態、Events、EndpointSliceまで確認することが障害調査では重要
