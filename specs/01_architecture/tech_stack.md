# 技術スタック選定

## フロントエンド

| 技術 | バージョン目安 | 選定理由 |
|------|--------------|---------|
| Next.js | 14.x (App Router) | SSR・SSG・RSC を統合。SEO と UX の両立が容易 |
| TypeScript | 5.x | 型安全性。API レスポンス型の共有が可能 |
| Tailwind CSS | 3.x | 高速 UI 開発。デザイン一貫性を保ちやすい |
| React Query (TanStack Query) | 5.x | キャッシュ管理・ポーリング・楽観的更新を宣言的に記述可能 |
| Zod | 3.x | API レスポンスのランタイムバリデーションとスキーマ共有 |

### フロントエンド設計方針
- Server Components をデフォルトとし、インタラクティブな部分のみ Client Components にする。
- ページコンポーネントはデータフェッチのみ担当。表示ロジックはプレゼンテーションコンポーネントに分離。
- 型定義は `types/` ディレクトリに集約し、API クライアントと UI で共有する。

---

## バックエンド

| 技術 | バージョン目安 | 選定理由 |
|------|--------------|---------|
| Go | 1.22.x | 高パフォーマンス・低リソース消費。並行処理が得意 |
| net/http (標準ライブラリ) | - | シンプルなルーティングには標準ライブラリで十分 |
| chi | v5.x | 軽量ルーター。ミドルウェアチェーンが書きやすい |
| sqlc | 2.x | SQL からタイプセーフな Go コードを生成。ORM の魔法を避ける |
| golang-migrate | 4.x | DB マイグレーション管理 |
| pgx | v5.x | PostgreSQL ドライバー。pgvector 対応 |
| go-redis | v9.x | Redis クライアント |
| zap | 2.x | 構造化ロギング |
| testify | 1.x | テストアサーション |

### バックエンド設計方針
- **Clean Architecture** ライクなレイヤー分離: handler → usecase → repository。
- handler は HTTP の変換のみ担当。ビジネスロジックは usecase に置く。
- repository は DB アクセスを抽象化し、テストではモックに差し替え可能にする。
- 設定値はすべて環境変数から読み込む（`config` パッケージで集約管理）。

---

## データレイヤー

| 技術 | バージョン目安 | 選定理由 |
|------|--------------|---------|
| PostgreSQL | 16.x | 信頼性・pgvector 拡張・全文検索 |
| pgvector | 0.7.x | 埋め込みベクトルの類似検索。外部検索エンジン不要（MVP） |
| Redis | 7.x | 高速キャッシュ・ジョブキュー（Streams）・レートリミット |

### DB 設計方針
- 全テーブルに `created_at`, `updated_at` を持たせる。
- 論理削除は必要なテーブルのみに限定し、`deleted_at` カラムで管理。
- ベクトル類似検索には IVFFlat インデックスを使用（データ量次第で HNSW に移行）。

---

## AI / データ処理

| 技術 | 選定理由 |
|------|---------|
| OpenAI API (text-embedding-3-small) | 低コスト・高品質な埋め込みベクトル生成 |
| OpenAI API (gpt-4o-mini) | 要約・タグ抽出。コスト効率が高い |
| gocolly または カスタムフェッチャー | RSS XML および記事本文のフェッチ |
| bluemonday | HTML サニタイズ |
| golang.org/x/net/html | HTML パース・テキスト抽出 |

### AI 利用方針
- プロンプトはコードと分離して `prompts/` ディレクトリで管理。
- 要約・タグはモデル名・バージョンを `article_summaries.model_version` に記録。
- 本番で使用するプロンプトは変更前にテスト用データセットで検証する。
- LLM API のレートリミットに備え、ワーカーはバースト制御付きのリトライを実装する。

---

## インフラストラクチャ

| 技術 | 選定理由 |
|------|---------|
| AWS ECS (Fargate) | サーバーレスコンテナ。インフラ管理コストが低い |
| AWS RDS (PostgreSQL) | マネージド DB。Multi-AZ によるフェイルオーバー |
| AWS ElastiCache (Redis) | マネージド Redis |
| AWS ALB | ロードバランシング・SSL 終端 |
| AWS CloudFront | CDN・静的アセット配信 |
| AWS S3 | 静的ファイル・コンテンツスナップショット保存 |
| AWS ECR | コンテナイメージレジストリ |
| Terraform | Infrastructure as Code |
| AWS CloudWatch | ログ集約・メトリクス・アラート（MVP） |

---

## DevOps

| 技術 | 選定理由 |
|------|---------|
| GitHub Actions | CI/CD。コスト無料枠が広い |
| Docker | コンテナビルド。本番と開発環境の差異を最小化 |
| docker-compose | ローカル開発環境（DB・Redis・全サービス起動） |

---

## MVP フェーズでの意図的な省略

| 技術 | 省略理由 | 将来的な導入タイミング |
|------|---------|---------------------|
| Meilisearch / OpenSearch | pgvector で全文検索は MVP 要件を満たせる | 記事数 50万件超 |
| gRPC | REST で十分。複雑性を避ける | サービス分割時 |
| Kafka | Redis Streams で MVP は対応可能 | ストリーミング要件が増えた時 |
| Python ML モジュール | Go 単体で MVP のレコメンドを実装可能 | 協調フィルタリング導入時 |
| Prometheus + Grafana | CloudWatch で MVP は十分 | 自社監視を強化したい時 |
