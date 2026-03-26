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

CREATE TRIGGER set_updated_at_sources
    BEFORE UPDATE ON sources
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE articles (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id       UUID        NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    url             TEXT        NOT NULL,
    url_hash        TEXT        NOT NULL UNIQUE,
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

CREATE TRIGGER set_updated_at_articles
    BEFORE UPDATE ON articles
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE INDEX idx_articles_status_created
    ON articles (status, created_at ASC)
    WHERE status IN ('pending', 'failed');

CREATE INDEX idx_articles_source_status_published
    ON articles (source_id, status, published_at DESC)
    WHERE status = 'processed';

CREATE INDEX idx_articles_trend_published
    ON articles (trend_score DESC, published_at DESC)
    WHERE status = 'processed';

CREATE INDEX idx_articles_published_at
    ON articles (published_at DESC)
    WHERE status = 'processed';
