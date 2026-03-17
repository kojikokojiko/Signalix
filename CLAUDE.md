# Signalix — Claude Code ガイドライン

## プロジェクト概要

AI パワードパーソナライズ RSS フィードプラットフォーム。
詳細は `specs/README.md` を参照。

---

## 仕様書の扱い（最重要）

**実装前に仕様書を確認し、変更が生じた場合は必ず仕様書を更新すること。**

### ルール

1. **仕様変更は仕様書を先に更新する**
   - API のエンドポイント・レスポンス形式・エラーコードを変更する場合は
     `specs/03_api/` の該当ファイルを先に更新してから実装する。
   - DB スキーマを変更する場合は `specs/04_database/schema.md` を先に更新する。
   - インフラ構成を変更する場合は `specs/07_infrastructure/architecture.md` を更新する。

2. **実装と仕様書の乖離を作らない**
   - コードを書いた結果、仕様書と実際の挙動が異なる場合は、
     実装を直すか仕様書を更新するかを明示的に判断してユーザーに確認する。
   - 「仕様書には A と書いてあるが、実装上 B にした」という状態を放置しない。

3. **仕様書は日本語で記述する**
   - `specs/` 以下のドキュメントはすべて日本語で書く。
   - コード内のコメント・変数名・関数名は英語で構わない。

### 仕様書の場所

```
specs/
├── 01_architecture/   システム構成・データフロー・技術スタック
├── 02_domain/         エンティティ定義・ユーザーストーリー
├── 03_api/            API エンドポイント仕様（認証・記事・レコメンド等）
├── 04_database/       DB スキーマ・インデックス設計
├── 05_pipeline/       RSS インジェスション・AI 処理・レコメンドパイプライン
├── 06_frontend/       画面仕様・コンポーネント設計・状態管理
├── 07_infrastructure/ インフラ構成（Fly.io 本番・AWS Appendix）
└── 08_testing/        テスト戦略・テストケース
```

---

## 開発方針

### SPEC 駆動 + TDD

1. 仕様書を確定してから実装を始める。
2. テストを実装より先に書く（テストが仕様の証明になる）。
3. テストが通ることを確認してからコードをコミットする。

### 技術スタック

- **バックエンド**: Go 1.22 + chi + sqlc + pgx
- **フロントエンド**: Next.js 14 (App Router) + TypeScript + Tailwind CSS + React Query
- **DB**: Supabase（PostgreSQL 16 + pgvector）
- **Redis**: Upstash
- **デプロイ**: Fly.io

### ディレクトリ構成（予定）

```
/
├── backend/       Go API サーバー
├── worker/        Go ワーカー（RSS・処理・レコメンド）
├── frontend/      Next.js フロントエンド
├── migrations/    DB マイグレーションファイル
├── specs/         仕様書
└── CLAUDE.md      このファイル
```

---

## コーディング規約

### Go

- `internal/` パッケージ構成: `handler` → `usecase` → `repository` のレイヤー分離を守る。
- エラーは `fmt.Errorf("context: %w", err)` でラップして伝播する。
- 設定値はすべて環境変数から読み込み、`internal/config` パッケージで集約する。

### TypeScript / React

- `any` 型の使用禁止。API レスポンスは Zod でランタイムバリデーションする。
- Server Components をデフォルトとし、インタラクティブな部分のみ `"use client"` にする。

### 共通

- シークレット（API キー・パスワード等）をコードにハードコードしない。
- `TODO:` コメントは残さず、その場で対処するか Issue を立てる。
