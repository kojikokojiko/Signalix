# データベーススキーマ仕様

## 前提

- データベース: PostgreSQL 16
- 拡張機能: `pgvector`（埋め込みベクトル）、`uuid-ossp`（UUID 生成）
- 全テーブルに `created_at`、`updated_at` を持たせる（`updated_at` は自動更新トリガー）。
- 外部キーには `ON DELETE CASCADE` または `ON DELETE RESTRICT` を明示する。

---

## 拡張機能

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";
```

---

## テーブル定義

### users

```sql
CREATE TABLE users (
    id                  UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    email               TEXT        NOT NULL UNIQUE,
    password_hash       TEXT        NOT NULL,
    display_name        TEXT        NOT NULL,
    avatar_url          TEXT,
    preferred_language  TEXT        NOT NULL DEFAULT 'en',
    is_admin            BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### user_interests

```sql
CREATE TABLE user_interests (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_id      UUID        NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    weight      FLOAT       NOT NULL DEFAULT 0.5 CHECK (weight >= 0.0 AND weight <= 1.0),
    source      TEXT        NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'inferred')),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tag_id)
);
```

---

### sources

```sql
CREATE TABLE sources (
    id                      UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                    TEXT        NOT NULL,
    feed_url                TEXT        NOT NULL UNIQUE,
    site_url                TEXT        NOT NULL,
    description             TEXT,
    category                TEXT        NOT NULL,
    language                TEXT        NOT NULL DEFAULT 'en',
    fetch_interval_minutes  INTEGER     NOT NULL DEFAULT 60 CHECK (fetch_interval_minutes >= 15),
    quality_score           FLOAT       NOT NULL DEFAULT 0.7 CHECK (quality_score >= 0.0 AND quality_score <= 1.0),
    status                  TEXT        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'paused', 'degraded', 'disabled')),
    last_fetched_at         TIMESTAMPTZ,
    consecutive_failures    INTEGER     NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### articles

```sql
CREATE TABLE articles (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id       UUID        NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    url             TEXT        NOT NULL,
    url_hash        TEXT        NOT NULL UNIQUE,  -- SHA-256 of url
    title           TEXT        NOT NULL,
    raw_content     TEXT,
    clean_content   TEXT,
    author          TEXT,
    language        TEXT,
    published_at    TIMESTAMPTZ,
    trend_score     FLOAT       NOT NULL DEFAULT 0.0 CHECK (trend_score >= 0.0 AND trend_score <= 1.0),
    status          TEXT        NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'processing', 'processed', 'failed', 'skipped')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### article_summaries

```sql
CREATE TABLE article_summaries (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id      UUID        NOT NULL UNIQUE REFERENCES articles(id) ON DELETE CASCADE,
    summary_text    TEXT        NOT NULL,
    model_name      TEXT        NOT NULL,
    model_version   TEXT        NOT NULL,
    prompt_version  TEXT        NOT NULL,
    token_count     INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### article_embeddings

```sql
CREATE TABLE article_embeddings (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id      UUID        NOT NULL UNIQUE REFERENCES articles(id) ON DELETE CASCADE,
    embedding       vector(1536) NOT NULL,
    model_name      TEXT        NOT NULL,
    model_version   TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### tags

```sql
CREATE TABLE tags (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT        NOT NULL UNIQUE,
    category    TEXT        NOT NULL DEFAULT 'topic',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

### article_tags

```sql
CREATE TABLE article_tags (
    article_id  UUID    NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag_id      UUID    NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    confidence  FLOAT   NOT NULL DEFAULT 1.0 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (article_id, tag_id)
);
```

---

### bookmarks

```sql
CREATE TABLE bookmarks (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id  UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, article_id)
);
```

---

### user_feedback

```sql
CREATE TABLE user_feedback (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id      UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    feedback_type   TEXT        NOT NULL
                        CHECK (feedback_type IN ('like', 'dislike', 'save', 'click', 'hide')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- like/dislike/save/hide は (user_id, article_id) で最大1件
-- click は複数記録可能
CREATE UNIQUE INDEX uq_user_feedback_non_click
    ON user_feedback (user_id, article_id)
    WHERE feedback_type != 'click';
```

---

### recommendation_logs

```sql
CREATE TABLE recommendation_logs (
    id                      UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id              UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    total_score             FLOAT       NOT NULL,
    relevance_score         FLOAT       NOT NULL DEFAULT 0.0,
    freshness_score         FLOAT       NOT NULL DEFAULT 0.0,
    trend_score             FLOAT       NOT NULL DEFAULT 0.0,
    source_quality_score    FLOAT       NOT NULL DEFAULT 0.0,
    personalization_boost   FLOAT       NOT NULL DEFAULT 0.0,
    explanation             TEXT        NOT NULL,
    generated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    UNIQUE (user_id, article_id)
);
```

---

### ingestion_jobs

```sql
CREATE TABLE ingestion_jobs (
    id                  UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id           UUID        NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    status              TEXT        NOT NULL DEFAULT 'running'
                            CHECK (status IN ('running', 'completed', 'failed')),
    articles_found      INTEGER     NOT NULL DEFAULT 0,
    articles_new        INTEGER     NOT NULL DEFAULT 0,
    articles_skipped    INTEGER     NOT NULL DEFAULT 0,
    error_message       TEXT,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ
);
```

---

### processing_jobs

```sql
CREATE TABLE processing_jobs (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id      UUID        NOT NULL UNIQUE REFERENCES articles(id) ON DELETE CASCADE,
    status          TEXT        NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    current_stage   TEXT,
    retry_count     INTEGER     NOT NULL DEFAULT 0,
    max_retries     INTEGER     NOT NULL DEFAULT 3,
    last_error      TEXT,
    queued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ
);
```

---

## 自動更新トリガー

```sql
-- updated_at を自動更新するファンクション
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 各テーブルにトリガーを設定
CREATE TRIGGER set_updated_at_users
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_sources
    BEFORE UPDATE ON sources
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_articles
    BEFORE UPDATE ON articles
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
```

---

## シードデータ（タグマスター）

```sql
INSERT INTO tags (name, category) VALUES
    -- プログラミング言語
    ('go', 'language'),
    ('rust', 'language'),
    ('python', 'language'),
    ('typescript', 'language'),
    ('javascript', 'language'),
    -- インフラ・DevOps
    ('kubernetes', 'infrastructure'),
    ('docker', 'infrastructure'),
    ('terraform', 'infrastructure'),
    ('aws', 'infrastructure'),
    -- AI・ML
    ('llm', 'ai'),
    ('machine-learning', 'ai'),
    ('ai-infrastructure', 'ai'),
    ('generative-ai', 'ai'),
    -- トピック
    ('backend', 'topic'),
    ('frontend', 'topic'),
    ('database', 'topic'),
    ('security', 'topic'),
    ('performance', 'topic'),
    ('open-source', 'topic'),
    ('startup', 'topic');
```
