CREATE TABLE user_sources (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_id   UUID        NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, source_id)
);

CREATE INDEX idx_user_sources_user_id ON user_sources(user_id);
