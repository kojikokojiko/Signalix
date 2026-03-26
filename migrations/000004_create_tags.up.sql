CREATE TABLE tags (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT        NOT NULL UNIQUE,
    category    TEXT        NOT NULL DEFAULT 'topic',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- シードデータ
INSERT INTO tags (name, category) VALUES
    ('go', 'language'),
    ('rust', 'language'),
    ('python', 'language'),
    ('typescript', 'language'),
    ('javascript', 'language'),
    ('kubernetes', 'infrastructure'),
    ('docker', 'infrastructure'),
    ('terraform', 'infrastructure'),
    ('aws', 'infrastructure'),
    ('llm', 'ai'),
    ('machine-learning', 'ai'),
    ('ai-infrastructure', 'ai'),
    ('generative-ai', 'ai'),
    ('backend', 'topic'),
    ('frontend', 'topic'),
    ('database', 'topic'),
    ('security', 'topic'),
    ('performance', 'topic'),
    ('open-source', 'topic'),
    ('startup', 'topic');
