# インデックス設計仕様

## 設計方針

- インデックスは「クエリパターンを先に定義し、それに必要なものだけ作る」という原則に従う。
- 過度なインデックスは書き込みパフォーマンスを低下させるため、必要なもののみ定義する。
- EXPLAIN ANALYZE で定期的にクエリプランを検証し、不要なインデックスを削除する。

---

## 主要クエリパターン

| クエリ | 使用テーブル | フィルタ条件 | ソート |
|-------|------------|------------|-------|
| パーソナライズフィード取得 | recommendation_logs, articles | `user_id`, `expires_at > NOW()` | `total_score DESC` |
| トレンド記事取得 | articles | `status='processed'`, `published_at > N days ago` | `trend_score DESC` |
| 記事検索 | articles, article_summaries | `title ILIKE`, `status='processed'` | `published_at DESC` |
| タグフィルタ | article_tags, articles | `tag_id IN (...)` | `published_at DESC` |
| ブックマーク一覧 | bookmarks, articles | `user_id` | `created_at DESC` |
| 埋め込み類似検索 | article_embeddings | embedding vector similarity | cosine distance |
| ソース別記事 | articles | `source_id`, `status='processed'` | `published_at DESC` |
| 処理待ち記事 | articles | `status='pending'` | `created_at ASC` |

---

## インデックス定義

### articles テーブル

```sql
-- URL ハッシュ（重複チェック。UNIQUE 制約でカバー済みだが明示）
-- ※ url_hash は UNIQUE 制約のため自動的にインデックスが作られる

-- ステータス別取得（処理ワーカーがポーリング）
CREATE INDEX idx_articles_status_created
    ON articles (status, created_at ASC)
    WHERE status IN ('pending', 'failed');

-- ソース別・ステータス別取得
CREATE INDEX idx_articles_source_status_published
    ON articles (source_id, status, published_at DESC)
    WHERE status = 'processed';

-- トレンドフィード取得（status=processed, trend_score DESC）
CREATE INDEX idx_articles_trend_published
    ON articles (trend_score DESC, published_at DESC)
    WHERE status = 'processed';

-- 公開日時でのソート（一般的な記事一覧）
CREATE INDEX idx_articles_published_at
    ON articles (published_at DESC)
    WHERE status = 'processed';

-- 言語フィルタ
CREATE INDEX idx_articles_language_published
    ON articles (language, published_at DESC)
    WHERE status = 'processed';
```

---

### article_tags テーブル

```sql
-- タグ別記事取得（複数タグの IN 検索に対応）
CREATE INDEX idx_article_tags_tag_id
    ON article_tags (tag_id, article_id);
```

---

### recommendation_logs テーブル

```sql
-- ユーザーのフィード取得（最重要クエリ）
CREATE INDEX idx_recommendation_logs_user_score
    ON recommendation_logs (user_id, total_score DESC, expires_at)
    WHERE expires_at > NOW();

-- 期限切れレコードのクリーンアップ
CREATE INDEX idx_recommendation_logs_expires
    ON recommendation_logs (expires_at)
    WHERE expires_at < NOW();
```

---

### bookmarks テーブル

```sql
-- ユーザーのブックマーク一覧
CREATE INDEX idx_bookmarks_user_created
    ON bookmarks (user_id, created_at DESC);

-- 記事がブックマークされているかの確認
-- ※ UNIQUE 制約 (user_id, article_id) が自動的にインデックスを作る
```

---

### user_feedback テーブル

```sql
-- ユーザーの記事フィードバック確認
CREATE INDEX idx_user_feedback_user_article
    ON user_feedback (user_id, article_id, feedback_type);

-- レコメンド計算用: タグ重みの更新トリガー
CREATE INDEX idx_user_feedback_created
    ON user_feedback (user_id, created_at DESC);
```

---

### user_interests テーブル

```sql
-- ユーザーの興味プロフィール取得（レコメンド計算時）
CREATE INDEX idx_user_interests_user_weight
    ON user_interests (user_id, weight DESC);
```

---

### ingestion_jobs テーブル

```sql
-- ソース別ジョブ履歴
CREATE INDEX idx_ingestion_jobs_source_started
    ON ingestion_jobs (source_id, started_at DESC);

-- 失敗ジョブの確認
CREATE INDEX idx_ingestion_jobs_status
    ON ingestion_jobs (status)
    WHERE status = 'failed';
```

---

### processing_jobs テーブル

```sql
-- キューイング状態の確認
CREATE INDEX idx_processing_jobs_status_queued
    ON processing_jobs (status, queued_at ASC)
    WHERE status IN ('queued', 'running');
```

---

## 埋め込みベクトルインデックス（pgvector）

```sql
-- コサイン類似度検索用 IVFFlat インデックス
-- lists パラメータ: sqrt(行数) が目安。100万行なら ~100
CREATE INDEX idx_article_embeddings_vector
    ON article_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
```

**注意事項:**
- IVFFlat は近似最近傍探索のため、完全一致ではない。
- データ量が 100 万件を超えた場合は HNSW インデックスへの移行を検討する。
- インデックス作成前に十分なデータが必要（最低でも `lists * 39` 行推奨）。
- `SET ivfflat.probes = 10;` で精度とパフォーマンスのバランスを調整する。

---

## 全文検索

MVP では PostgreSQL の組み込み全文検索を使用する。

```sql
-- タイトルと要約テキストの全文検索用インデックス
CREATE INDEX idx_articles_title_fts
    ON articles
    USING gin(to_tsvector('english', title));

-- 要約テキスト（article_summaries テーブル）
CREATE INDEX idx_article_summaries_fts
    ON article_summaries
    USING gin(to_tsvector('english', summary_text));
```

**将来的な移行:** データ量増加・検索精度改善が必要になった時点で Meilisearch または
OpenSearch への移行を検討する。

---

## 定期メンテナンス

```sql
-- VACUUM と ANALYZE を定期実行（週次推奨）
VACUUM ANALYZE articles;
VACUUM ANALYZE recommendation_logs;
VACUUM ANALYZE user_feedback;

-- 期限切れレコメンドの削除（日次バッチ）
DELETE FROM recommendation_logs WHERE expires_at < NOW();

-- 古い ingestion_jobs の削除（日次バッチ、14日以上前）
DELETE FROM ingestion_jobs WHERE started_at < NOW() - INTERVAL '14 days';

-- 古い processing_jobs の削除（日次バッチ、14日以上前）
DELETE FROM processing_jobs
    WHERE completed_at < NOW() - INTERVAL '14 days'
    AND status IN ('completed', 'failed');
```
