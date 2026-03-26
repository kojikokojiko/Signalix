CREATE TABLE user_interests (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_id      UUID        NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    weight      FLOAT       NOT NULL DEFAULT 0.5 CHECK (weight >= 0.0 AND weight <= 1.0),
    source      TEXT        NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'inferred')),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tag_id)
);

CREATE INDEX idx_user_interests_user_weight
    ON user_interests (user_id, weight DESC);
