# インフラストラクチャ仕様: 本番構成（Fly.io）

## 設計方針

個人運用・ポートフォリオ用途を前提に、**シンプルさと低コストを最優先**した構成。

- コンピュート: **Fly.io**（デプロイが `fly deploy` 1 コマンド、TLS 自動）
- DB: **Supabase**（PostgreSQL + pgvector + コネクションプーリング込み）
- Redis: **Upstash**（サーバーレス、個人トラフィックなら無料枠内）
- インフラ管理: Terraform 不要。Fly.io は `fly.toml` で宣言的に管理。

> AWS を使ったより本格的なアーキテクチャは
> [appendix_aws_architecture.md](./appendix_aws_architecture.md) を参照。

---

## 全体構成図

```
インターネット
    │
Fly.io Edge（TLS 終端・ロードバランシング・DDoS 保護 込み）
    │
    ├── signalix.io/*      → Fly App: Next.js フロントエンド
    ├── api.signalix.io/*  → Fly App: Go API サーバー
    └── [内部通信]         → Fly App: ワーカー（RSS・処理・レコメンド）

外部サービス:
  Supabase    : PostgreSQL + pgvector
  Upstash     : Redis（キャッシュ・ジョブキュー）
  OpenAI API  : 埋め込み生成・要約・タグ抽出
```

---

## Fly.io アプリ構成

### アプリ一覧

| アプリ名 | 役割 | サイズ | 台数 |
|---------|------|-------|-----|
| `signalix-api` | Go API サーバー | shared-cpu-1x / 256MB | 1 |
| `signalix-frontend` | Next.js SSR | shared-cpu-1x / 256MB | 1 |
| `signalix-worker` | RSS・処理・レコメンドワーカー（1 プロセス） | shared-cpu-1x / 512MB | 1 |

**ワーカーを 1 プロセスにまとめる理由:**
- 個人用途では同時処理数が少ないため分離不要。
- RSS インジェスション・記事処理・レコメンド計算を 1 バイナリの goroutine で実行する。
- スケールが必要になった時点で分離する。

### fly.toml（API サーバー）

```toml
# fly.toml (signalix-api)
app = "signalix-api"
primary_region = "nrt"  # 東京

[build]
  dockerfile = "backend/Dockerfile"

[http_service]
  internal_port = 8080
  force_https   = true
  auto_stop_machines  = true   # アクセスがない時は自動停止（コスト削減）
  auto_start_machines = true   # リクエスト来たら自動起動

  [http_service.concurrency]
    type       = "requests"
    hard_limit = 200
    soft_limit = 150

[[vm]]
  size   = "shared-cpu-1x"
  memory = "256mb"
```

### fly.toml（ワーカー）

```toml
# fly.toml (signalix-worker)
app = "signalix-worker"
primary_region = "nrt"

[build]
  dockerfile = "worker/Dockerfile"

# HTTP エンドポイントなし（ワーカーはポートを公開しない）
[processes]
  worker = "./worker"

[[vm]]
  size   = "shared-cpu-1x"
  memory = "512mb"

# スケジュール実行（RSS フェッチ: 1時間ごと）
[[statics]]
  # ワーカー内部でスケジューラを実装するため Fly Cron は使わない
```

### 内部通信

Fly.io では同一 Organization 内のアプリが `<app-name>.internal` で通信できる（WireGuard）。

```
signalix-frontend → api.signalix.io（パブリック）または
                  → signalix-api.internal:8080（内部、レイテンシ低）
signalix-worker   → signalix-api.internal:8080（必要な場合）
```

---

## Supabase

### プラン

| フェーズ | プラン | 月額 |
|--------|-------|------|
| 開発・検証 | Free | $0（1週間非アクティブで一時停止） |
| **本番** | **Pro** | **$25** |

### 設定

```
リージョン: Northeast Asia (Tokyo) ← Fly.io nrt リージョンに合わせる
有効化する拡張: vector, uuid-ossp
```

### 接続 URL の使い分け

```bash
# アプリ用（Transaction Mode / コネクションプーリング）
DATABASE_URL=postgresql://postgres.[ref]:pass@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres

# マイグレーション用（Direct）
DATABASE_URL_DIRECT=postgresql://postgres:pass@db.[ref].supabase.co:5432/postgres
```

### マイグレーション

```bash
# ローカルからマイグレーション実行
migrate -path ./migrations -database "$DATABASE_URL_DIRECT" up

# CI/CD（GitHub Actions）からも同様
```

### ローカル開発

```bash
# Supabase CLI でローカル環境を再現
supabase start
# → PostgreSQL (pgvector 込み) が localhost:54322 で起動
# → Studio UI が localhost:54323 で起動
```

---

## Upstash Redis

### プラン

```
プラン: Pay as you go
リージョン: ap-northeast-1（Tokyo）

個人トラフィック想定コスト:
  フィードキャッシュ + ジョブキュー + レートリミット
  → 約 50,000〜100,000 コマンド/日
  → $0〜3/月（無料枠 10,000/日 を超えた分のみ課金）
```

### 接続

```go
rdb := redis.NewClient(&redis.Options{
    Addr:      os.Getenv("UPSTASH_REDIS_URL"),   // rediss://...
    Password:  os.Getenv("UPSTASH_REDIS_TOKEN"),
    TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
})
```

Redis Streams（ジョブキュー）は Upstash が完全サポート。既存コードの変更なし。

---

## シークレット管理

Fly.io の Secrets 機能を使用（`fly secrets set`）。環境変数として ECS タスクに注入される。

```bash
fly secrets set \
  DATABASE_URL="postgresql://..." \
  DATABASE_URL_DIRECT="postgresql://..." \
  UPSTASH_REDIS_URL="rediss://..." \
  UPSTASH_REDIS_TOKEN="..." \
  OPENAI_API_KEY="sk-..." \
  JWT_SECRET="..." \
  --app signalix-api

# ワーカーにも同様に設定
fly secrets set ... --app signalix-worker
```

---

## デプロイフロー

```bash
# API サーバー
cd backend
fly deploy --app signalix-api

# フロントエンド
cd frontend
fly deploy --app signalix-frontend

# ワーカー
cd worker
fly deploy --app signalix-worker
```

GitHub Actions からの自動デプロイ（`fly deploy` を CI から呼ぶだけ）:

```yaml
# .github/workflows/deploy.yml
- name: Deploy API
  run: fly deploy --app signalix-api --remote-only
  env:
    FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

---

## カスタムドメイン・TLS

```bash
# カスタムドメインの設定（TLS 証明書は自動発行）
fly certs add signalix.io --app signalix-frontend
fly certs add api.signalix.io --app signalix-api

# DNS レコード（Fly.io から指示される A/AAAA/CNAME を設定）
```

---

## コスト試算

| サービス | 月額概算 |
|---------|---------|
| Fly.io（signalix-api、常時起動） | ~$4 (shared-cpu-1x, 256MB) |
| Fly.io（signalix-frontend、常時起動） | ~$4 |
| Fly.io（signalix-worker、常時起動） | ~$6 (512MB) |
| Supabase Pro | $25 |
| Upstash Redis | $0〜3 |
| OpenAI API（1,000 記事/日） | ~$9 |
| **合計** | **~$48〜51/月** |

**auto_stop_machines = true** を活用すれば、アクセスがない時間帯に自動停止するため
フロントエンドや API は $1〜2/月 まで下がる可能性がある（ただしコールドスタートが発生）。

---

## 環境設計

| 環境 | 構成 | Supabase | Upstash |
|------|------|---------|--------|
| `production` | Fly.io（Tokyo） | Pro $25 | Pay-as-you-go |
| `staging` | Fly.io（Free tier, auto-stop） | Free プロジェクト | Free |
| `development` | docker-compose ローカル | `supabase start` | `redis` コンテナ |

---

## 監視・ログ

```bash
# リアルタイムログ確認
fly logs --app signalix-api

# メトリクス確認（Fly.io ダッシュボード）
fly dashboard --app signalix-api
```

Fly.io は CPU・メモリ・リクエスト数のメトリクスをダッシュボードで確認できる。
より詳細な監視が必要な場合は Grafana Cloud（無料枠あり）との連携を検討する。
