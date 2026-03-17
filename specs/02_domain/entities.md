# ドメインエンティティ仕様

## エンティティ一覧と関係

```
users
  ├── user_interests (多)
  ├── bookmarks (多)
  ├── user_feedback (多)
  └── recommendation_logs (多)

sources
  └── articles (多)
        ├── article_summaries (1)
        ├── article_embeddings (1)
        ├── article_tags (多) ── tags
        ├── bookmarks (多)
        ├── user_feedback (多)
        └── recommendation_logs (多)

ingestion_jobs
  └── sources (多)

processing_jobs
  └── articles (1)
```

---

## エンティティ詳細

### users

ユーザーアカウント情報。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| email | TEXT | 一意。メールアドレス |
| password_hash | TEXT | bcrypt ハッシュ |
| display_name | TEXT | 表示名 |
| avatar_url | TEXT | アバター画像 URL（任意） |
| preferred_language | TEXT | 優先言語コード（例: "ja", "en"）デフォルト "en" |
| is_admin | BOOLEAN | 管理者フラグ |
| is_active | BOOLEAN | アカウント有効フラグ |
| last_login_at | TIMESTAMPTZ | 最終ログイン日時 |
| created_at | TIMESTAMPTZ | 作成日時 |
| updated_at | TIMESTAMPTZ | 更新日時 |

---

### user_interests

ユーザーの興味・関心プロフィール。タグ単位で管理する。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| user_id | UUID | FK: users.id |
| tag_id | UUID | FK: tags.id |
| weight | FLOAT | 興味の強さ（0.0 〜 1.0）。初期値 0.5 |
| source | TEXT | 重み付けの発生源: `manual`（ユーザー設定）/ `inferred`（行動推定） |
| updated_at | TIMESTAMPTZ | 最終更新日時 |

**ビジネスルール:**
- `user_id + tag_id` は一意制約。
- 行動シグナルによる `weight` 更新は加重移動平均で計算する。
- `weight` が 0.1 未満のものは定期的に削除してプロフィールを整理する。

---

### sources

RSS フィードのソース情報。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| name | TEXT | ソース名（例: "Hacker News"） |
| feed_url | TEXT | RSS フィード URL |
| site_url | TEXT | サイトのルート URL |
| description | TEXT | ソースの説明 |
| category | TEXT | カテゴリ（例: "tech", "ai", "startup"） |
| language | TEXT | コンテンツ言語コード |
| fetch_interval_minutes | INTEGER | フェッチ間隔（分）。デフォルト 60 |
| quality_score | FLOAT | ソース品質スコア（0.0 〜 1.0）。初期値 0.7 |
| status | TEXT | `active` / `paused` / `degraded` / `disabled` |
| last_fetched_at | TIMESTAMPTZ | 最終フェッチ日時 |
| consecutive_failures | INTEGER | 連続失敗回数。デフォルト 0 |
| created_at | TIMESTAMPTZ | 作成日時 |
| updated_at | TIMESTAMPTZ | 更新日時 |

**ステータス遷移:**
- `active` → `degraded`: 連続失敗 3 回
- `degraded` → `active`: フェッチ成功時
- `degraded` → `disabled`: 連続失敗 10 回または管理者操作
- `active` ↔ `paused`: 管理者操作

---

### articles

RSS から取得した記事のマスターレコード。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| source_id | UUID | FK: sources.id |
| url | TEXT | 記事の元 URL |
| url_hash | TEXT | URL の SHA-256。一意制約 |
| title | TEXT | 記事タイトル |
| raw_content | TEXT | 取得時の生コンテンツ（HTML） |
| clean_content | TEXT | HTML 除去済みのクリーンテキスト |
| author | TEXT | 著者名（任意） |
| language | TEXT | 検出された言語コード |
| published_at | TIMESTAMPTZ | 記事の公開日時 |
| trend_score | FLOAT | トレンドスコア（0.0 〜 1.0）。デフォルト 0.0 |
| status | TEXT | `pending` / `processing` / `processed` / `failed` / `skipped` |
| created_at | TIMESTAMPTZ | インジェスト日時 |
| updated_at | TIMESTAMPTZ | 更新日時 |

**ステータス遷移:**
- `pending` → `processing`: ワーカーがジョブを取得した時
- `processing` → `processed`: 全ステージ完了時
- `processing` → `failed`: リトライ上限超過時
- `pending` → `skipped`: コンテンツが短すぎるなど処理対象外の場合

---

### article_summaries

AI が生成した記事要約。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| article_id | UUID | FK: articles.id。一意 |
| summary_text | TEXT | 要約本文（2〜5 文） |
| model_name | TEXT | 使用した LLM モデル名（例: "gpt-4o-mini"） |
| model_version | TEXT | モデルバージョン（例: "2024-07-18"） |
| prompt_version | TEXT | 使用したプロンプトのバージョン（例: "v1.2"） |
| token_count | INTEGER | 生成に使ったトークン数 |
| created_at | TIMESTAMPTZ | 生成日時 |

---

### article_embeddings

記事のベクトル埋め込み。類似検索に使用。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| article_id | UUID | FK: articles.id。一意 |
| embedding | vector(1536) | 埋め込みベクトル（text-embedding-3-small は 1536 次元） |
| model_name | TEXT | 使用した埋め込みモデル名 |
| model_version | TEXT | モデルバージョン |
| created_at | TIMESTAMPTZ | 生成日時 |

---

### tags

タグマスター。記事とユーザー興味の共通語彙。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| name | TEXT | タグ名（例: "go", "machine-learning", "kubernetes"）。一意 |
| category | TEXT | タグカテゴリ（例: "language", "framework", "topic"） |
| created_at | TIMESTAMPTZ | 作成日時 |

---

### article_tags

記事とタグの多対多中間テーブル。

| カラム | 型 | 説明 |
|-------|-----|------|
| article_id | UUID | FK: articles.id |
| tag_id | UUID | FK: tags.id |
| confidence | FLOAT | AI によるタグ付けの信頼度（0.0 〜 1.0） |
| created_at | TIMESTAMPTZ | 付与日時 |

**制約:** `(article_id, tag_id)` が複合プライマリキー。

---

### bookmarks

ユーザーが保存した記事。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| user_id | UUID | FK: users.id |
| article_id | UUID | FK: articles.id |
| created_at | TIMESTAMPTZ | 保存日時 |

**制約:** `(user_id, article_id)` は一意。

---

### user_feedback

ユーザーの行動シグナル。レコメンドの学習に使用。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| user_id | UUID | FK: users.id |
| article_id | UUID | FK: articles.id |
| feedback_type | TEXT | `like` / `dislike` / `save` / `click` / `hide` |
| created_at | TIMESTAMPTZ | シグナル発生日時 |

**フィードバック種別の意味:**
- `like`: 明示的に良いと評価
- `dislike`: 明示的に悪いと評価
- `save`: ブックマーク保存（自動記録）
- `click`: 記事を開いた（自動記録）
- `hide`: この記事を非表示にする

---

### recommendation_logs

ユーザーごとのレコメンドスコアと説明を格納。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| user_id | UUID | FK: users.id |
| article_id | UUID | FK: articles.id |
| total_score | FLOAT | 最終スコア（降順ソートに使用） |
| relevance_score | FLOAT | コサイン類似度スコア |
| freshness_score | FLOAT | 新しさスコア |
| trend_score | FLOAT | トレンドスコア |
| source_quality_score | FLOAT | ソース品質スコア |
| personalization_boost | FLOAT | 個人化ブースト値 |
| explanation | TEXT | 推薦理由（例: "よく読む Go バックエンド記事に類似"） |
| generated_at | TIMESTAMPTZ | スコア計算日時 |
| expires_at | TIMESTAMPTZ | このスコアの有効期限。30日後 |

**制約:** `(user_id, article_id)` は一意。新しいスコア計算時に UPSERT する。

---

### ingestion_jobs

RSS フェッチジョブの実行記録。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| source_id | UUID | FK: sources.id |
| status | TEXT | `running` / `completed` / `failed` |
| articles_found | INTEGER | 発見した記事数 |
| articles_new | INTEGER | 新規挿入した記事数 |
| articles_skipped | INTEGER | 重複でスキップした記事数 |
| error_message | TEXT | エラー詳細（失敗時） |
| started_at | TIMESTAMPTZ | 開始日時 |
| completed_at | TIMESTAMPTZ | 完了日時 |

---

### processing_jobs

記事処理ジョブの実行記録。

| カラム | 型 | 説明 |
|-------|-----|------|
| id | UUID | プライマリキー |
| article_id | UUID | FK: articles.id |
| status | TEXT | `queued` / `running` / `completed` / `failed` |
| current_stage | TEXT | 現在処理中のステージ名 |
| retry_count | INTEGER | リトライ回数。デフォルト 0 |
| max_retries | INTEGER | 最大リトライ回数。デフォルト 3 |
| last_error | TEXT | 直近のエラー詳細 |
| queued_at | TIMESTAMPTZ | キュー投入日時 |
| started_at | TIMESTAMPTZ | 処理開始日時 |
| completed_at | TIMESTAMPTZ | 完了日時 |
