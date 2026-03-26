-- フロントエンドのオンボーディング画面で使用するタグを追加
INSERT INTO tags (name, category) VALUES
    ('java', 'language'),
    ('kotlin', 'language'),
    ('swift', 'language'),
    ('ruby', 'language'),
    ('gcp', 'infrastructure'),
    ('azure', 'infrastructure'),
    ('linux', 'infrastructure'),
    ('deep-learning', 'ai'),
    ('nlp', 'ai'),
    ('computer-vision', 'ai'),
    ('architecture', 'topic'),
    ('devops', 'topic')
ON CONFLICT (name) DO NOTHING;
