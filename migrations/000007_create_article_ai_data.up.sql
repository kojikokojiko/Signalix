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

CREATE TABLE article_embeddings (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id      UUID        NOT NULL UNIQUE REFERENCES articles(id) ON DELETE CASCADE,
    embedding       vector(1536) NOT NULL,
    model_name      TEXT        NOT NULL,
    model_version   TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE article_tags (
    article_id  UUID    NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag_id      UUID    NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    confidence  FLOAT   NOT NULL DEFAULT 1.0 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (article_id, tag_id)
);

CREATE INDEX idx_article_tags_tag_id
    ON article_tags (tag_id, article_id);
