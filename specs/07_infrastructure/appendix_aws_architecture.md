# [Appendix] AWS 本格構成

> **このドキュメントについて**
>
> 本番環境は [architecture.md](./architecture.md)（Fly.io 構成）を使用する。
> このドキュメントは、商用・チーム規模へのスケールアップ時や
> AWS を用いた技術ポートフォリオとしての参考資料として残す。

---

## この構成が適するシナリオ

- 複数人チームでの運用
- 高トラフィック（月間数十万 PV 以上）
- SLA・可用性要件が厳しい（99.9% 以上）
- AWS の知識・経験をアピールしたい場合

---

## 設計方針

**「プロダクションレベルの可用性・スケーラビリティ」**を前提とした、AWS ネイティブな構成。

- スケーラビリティより **コストと運用の簡潔さ** を優先する
- PostgreSQL → **Supabase**（pgvector 込み）
- Redis → **Upstash**（サーバーレス、ほぼ無料枠内）
- ECS はパブリックサブネットに置いて **NAT Gateway を廃止**
- ALB は廃止し、**ECS タスクに直接アクセス**（1 台構成）

---

## 全体構成図

```
インターネット
    │
Route 53 (DNS)
    │
ECS Fargate: API サーバー  ─── Supabase (PostgreSQL + pgvector)
ECS Fargate: Next.js        └── Upstash Redis
ECS Fargate: Workers             └── OpenAI API（外部）

[VPC: 10.0.0.0/16]
  └── パブリックサブネット (1 AZ)
        └── ECS Fargate Tasks（全サービス）
              assign_public_ip = ENABLED
              セキュリティグループで制御
```

### ALB を廃止する理由

- 個人利用なので複数タスクへのロードバランシングは不要。
- ECS Fargate の Service に desired_count=1 で十分。
- ALB 廃止で **$20/月** 削減。
- Route 53 の A レコードを ECS タスクの IP に向ける（タスク再起動時は IP が変わるため、
  後述の `ECS Service Connect` または Elastic IP で対応）。

### ネットワーク構成の選択肢

**シンプル案（MVP）:** ECS タスクに Elastic IP を割り当てて Route 53 から直接指定する。
タスク再起動時は IP が変わらない（Elastic IP は固定）。

```
Route 53 A レコード → Elastic IP → ECS Fargate タスク
```

ただし Elastic IP は ECS Fargate ネイティブではサポートされないため、
実用上は **ALB を最小構成で維持するか、後述の Fly.io/Railway への移行**を検討。

---

## 代替案: Fly.io（AWS より更に安くシンプル）

個人用途かつポートフォリオ展示なら、**AWS ECS + ALB をやめて Fly.io** が最もコスパが良い。

| 比較軸 | AWS ECS + ALB | Fly.io |
|-------|--------------|--------|
| 月額 | ~$128 | **~$10〜20** |
| デプロイ | docker build + ECR push + ECS update | `fly deploy` 1 コマンド |
| TLS/ドメイン | ALB + ACM | 自動（無料） |
| スケーリング | ECS Auto Scaling | `fly scale` コマンド |
| ポートフォリオ映え | ECS/Fargate は技術的に本格的 | シンプルすぎる印象 |

**Fly.io 構成の場合:**
- `fly.toml` でサービス定義、`fly deploy` でデプロイ
- 内部 DNS で複数サービスが通信可能（API・フロント・ワーカー）
- Supabase + Upstash はそのまま使用

---

## 推奨構成（2 パターン）

### パターン A: AWS ECS（ポートフォリオとして技術力を見せたい場合）

```
Route 53 → ALB → ECS Fargate（パブリックサブネット, desired_count=1）
                 + Supabase + Upstash
```

**月額: ~$75**（ALB $20 + ECS $60 + Route53 $1 + S3 $1 + OpenAI $9 - Supabase $25 + Upstash $5）

### パターン B: Fly.io（コスト最優先の場合）

```
Fly.io → API サーバー（Go）
       → Next.js フロントエンド
       → ワーカー群（RSS・処理・レコメンド）
       + Supabase + Upstash
```

**月額: ~$30〜40**（Fly.io ~$15 + Supabase $25 + Upstash $5 + OpenAI $9）

---

## パターン A 詳細: AWS ECS 最小構成

### ECS サービス

```hcl
resource "aws_ecs_service" "api_server" {
  name          = "signalix-api"
  desired_count = 1        # 個人用途: 1 台
  launch_type   = "FARGATE"

  network_configuration {
    subnets          = [aws_subnet.public.id]
    security_groups  = [aws_security_group.api.id]
    assign_public_ip = "ENABLED"
  }
}
```

| サービス | CPU | メモリ | 台数 |
|---------|-----|-------|-----|
| API サーバー | 256 | 512 | 1 |
| Next.js フロント | 256 | 512 | 1 |
| RSS + 処理 + レコメンドワーカー | 512 | 1024 | 1（1 プロセスにまとめる） |

**ワーカーを 1 プロセスにまとめる:** 個人用途なら RSS・処理・レコメンドワーカーを
1 つの Go バイナリで goroutine として動かせば十分。ECS タスクが 1 つで済む。

### セキュリティグループ

```hcl
# API: 外部から 443 / 8080
resource "aws_security_group" "api" {
  ingress { from_port = 443; cidr_blocks = ["0.0.0.0/0"] }
  egress  { from_port = 0;   cidr_blocks = ["0.0.0.0/0"] }
}

# フロント: 外部から 443 / 3000
resource "aws_security_group" "frontend" {
  ingress { from_port = 443; cidr_blocks = ["0.0.0.0/0"] }
  egress  { from_port = 0;   cidr_blocks = ["0.0.0.0/0"] }
}
```

### ALB の扱い

個人用途なので ALB なしで、Route 53 が ECS タスクの IP を直接指す。
タスク再起動時に IP 変更が発生するのが唯一の問題。

**対策（シンプル順）:**
1. **起動スクリプトで Route 53 を自動更新**: タスク起動時に自身の IP を Route 53 に登録する Lambda or スクリプト。
2. **ALB を使う（$20/月 追加）**: 安定性が必要になったタイミングで追加。
3. **Fly.io や Railway に移行**: そもそも問題にならない。

---

## Supabase 設定（パターン A / B 共通）

```
プラン: Pro ($25/月)
リージョン: us-east-1（AWS と同じリージョン推奨）

有効化する拡張:
  vector (pgvector)   ← 必須
  uuid-ossp           ← 必須

接続:
  Transaction Mode: アプリ用（短命接続）
  Direct: マイグレーション用のみ

ローカル開発:
  supabase start  → http://localhost:54323 (Studio UI)
```

---

## Upstash Redis 設定（パターン A / B 共通）

```
プラン: Pay as you go（個人用途は無料枠でほぼ収まる）
リージョン: us-east-1

用途:
  - フィードキャッシュ: TTL 5 分
  - ジョブキュー: Redis Streams
  - レートリミット: sliding window

予想月額: $0〜5（個人トラフィックなら無料枠内に収まる可能性が高い）
```

---

## コスト試算

| 構成 | 月額概算 | 備考 |
|------|---------|------|
| 旧（AWS フル、RDS/ElastiCache/NAT込み） | ~$283 | 当初案 |
| パターン A（ECS + Supabase + Upstash、ALB あり） | **~$75** | ポートフォリオ向け |
| パターン A（ALB なし、Route53 直接） | **~$55** | タスク再起動時に IP 変更あり |
| パターン B（Fly.io + Supabase + Upstash） | **~$35〜50** | 最安、運用最シンプル |

---

## シークレット管理

```
AWS Secrets Manager（パターン A）または 環境変数（パターン B）:
  DATABASE_URL          : Supabase Transaction Mode URL
  DATABASE_URL_DIRECT   : Supabase Direct URL（マイグレーション用）
  REDIS_URL             : Upstash Redis URL
  REDIS_TOKEN           : Upstash 認証トークン
  OPENAI_API_KEY        : OpenAI API キー
  JWT_SECRET            : JWT 署名シークレット
```

---

## 環境設計

| 環境 | 構成 | Supabase | Upstash |
|------|------|---------|--------|
| `production` | ECS or Fly.io | Pro ($25) | Pay-as-you-go |
| `staging` | Fly.io free tier | Free プロジェクト | Free |
| `development` | docker-compose ローカル | `supabase start` | `redis` コンテナ |

---

## 決定事項（要確認）

以下はどちらを選ぶか方針を決めてから仕様を確定する:

- [ ] **パターン A（ECS）** vs **パターン B（Fly.io）** どちらにするか
- [ ] ALB あり / なし
- [ ] ワーカーを 1 プロセスにまとめる vs 分離する
