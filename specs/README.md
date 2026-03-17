# Signalix — 仕様書インデックス

Signalix は AI を活用したパーソナライズド RSS フィードプラットフォームです。
このディレクトリにはすべての設計・エンジニアリング仕様が格納されています。

## ディレクトリ構成

```
specs/
├── 01_architecture/
│   ├── overview.md           # システム概要、コンポーネント境界
│   ├── data_flow.md          # エンドツーエンドのデータフロー
│   └── tech_stack.md         # 技術選定と選定理由
├── 02_domain/
│   ├── entities.md           # コアドメインエンティティとリレーション
│   └── user_stories.md       # ペルソナ別ユーザーストーリー
├── 03_api/
│   ├── conventions.md        # バージョニング・認証・エラー形式・ページネーション
│   ├── auth.md               # 認証エンドポイント
│   ├── users.md              # ユーザープロフィール・設定
│   ├── sources.md            # フィードソース管理
│   ├── articles.md           # 記事取得
│   ├── recommendations.md    # パーソナライズフィード・トレンドフィード
│   ├── bookmarks.md          # ブックマーク管理
│   ├── feedback.md           # ユーザーフィードバックシグナル
│   └── admin.md              # 管理者操作
├── 04_database/
│   ├── schema.md             # 全テーブルのDDL・カラム定義
│   └── indexes.md            # インデックス戦略と根拠
├── 05_pipeline/
│   ├── rss_ingestion.md      # RSS フェッチ・パースワーカー
│   ├── article_processing.md # 正規化・埋め込み・要約・タグ付け
│   ├── recommendation.md     # レコメンドスコアリングパイプライン
│   └── ai_integration.md     # LLM・埋め込みサービスの契約仕様
├── 06_frontend/
│   ├── screens.md            # 画面一覧とレイアウト仕様
│   ├── components.md         # 共通コンポーネント設計
│   └── state_management.md   # データフェッチとクライアント状態管理
├── 07_infrastructure/
│   ├── architecture.md               # 本番構成（Fly.io + Supabase + Upstash）
│   ├── appendix_aws_architecture.md  # [Appendix] AWS 本格構成（ポートフォリオ参照用）
│   ├── cicd.md                       # CI/CD パイプライン設計
│   └── observability.md              # ロギング・メトリクス・アラート
└── 08_testing/
    ├── strategy.md           # テスト全体戦略とカバレッジ目標
    ├── backend_tests.md      # Go ユニット・統合テストパターン
    ├── frontend_tests.md     # React コンポーネント・E2E テストパターン
    └── api_contract_tests.md # API コントラクト・統合テストケース
```

## 開発方針

**SPEC駆動 + TDD**

1. 実装開始前に仕様を確定する。
2. API コントラクトを先に定義し、バックエンド・フロントエンドはそれに準拠して実装する。
3. テストは実装ロジックより先に書く。
4. 破壊的変更は必ず仕様更新を先行させる。

## MVP スコープ概要

| 領域 | MVP | Phase 2 | Phase 3 |
|------|-----|---------|---------|
| 認証 | メール/パスワード | OAuth (Google) | MFA |
| インジェスション | スケジュール RSS フェッチ | Webhook トリガー | リアルタイムストリーミング |
| AI | 要約 + タグ | 埋め込みベースレコメンド | ハイブリッド協調フィルタリング |
| レコメンド | コンテンツベース | 行動シグナル | 協調フィルタリング |
| フロントエンド | フィード + 詳細 + ブックマーク | トピックフォロー | ダイジェストメール |

## 重要な制約

- 明示的に記載がない限り、すべての公開 API エンドポイントは JWT 認証を必須とする。
- レコメンド理由はスコアとともに保存・提供する。
- バックグラウンドジョブは冪等かつリトライ可能でなければならない。
- AI 出力はモデル名・バージョンとともにバージョン管理する。
