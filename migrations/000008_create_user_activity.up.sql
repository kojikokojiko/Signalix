CREATE TABLE bookmarks (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id  UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, article_id)
);

CREATE INDEX idx_bookmarks_user_created
    ON bookmarks (user_id, created_at DESC);

CREATE TABLE user_feedback (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id      UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    feedback_type   TEXT        NOT NULL
                        CHECK (feedback_type IN ('like', 'dislike', 'save', 'click', 'hide')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- like/dislike/save/hide は (user_id, article_id) で最大1件
CREATE UNIQUE INDEX uq_user_feedback_non_click
    ON user_feedback (user_id, article_id)
    WHERE feedback_type != 'click';

CREATE INDEX idx_user_feedback_user_article
    ON user_feedback (user_id, article_id, feedback_type);
