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

CREATE INDEX idx_recommendation_logs_user_score
    ON recommendation_logs (user_id, total_score DESC, expires_at);

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

CREATE INDEX idx_ingestion_jobs_source_started
    ON ingestion_jobs (source_id, started_at DESC);

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

CREATE INDEX idx_processing_jobs_status_queued
    ON processing_jobs (status, queued_at ASC)
    WHERE status IN ('queued', 'running');
