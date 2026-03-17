# パイプライン仕様: 記事処理

## 概要

記事処理ワーカーは、`pending` 状態の記事を取得し、正規化・埋め込み生成・AI 要約・
タグ付け・スコアリングの 5 ステージで処理する。各ステージは独立しており、
失敗時に個別リトライが可能。

---

## ワーカー設計

### 並行性

- 複数レプリカで稼働（ECS の desired count で制御）。
- 各レプリカは Redis Streams からジョブを消費（Consumer Group パターン）。
- 同一記事の二重処理は Consumer Group の ACK 機構で防ぐ。

### ポーリング設定

```
Consumer Group: "article_processing_workers"
Stream: "stream:article_processing"
バッチ取得数: 5 件/ポーリング
ポーリング間隔: 1 秒（ブロッキング XREADGROUP）
未 ACK ジョブのリクレーム: 5 分以上経過したら他ワーカーが引き継ぐ
```

---

## 処理ステージ詳細

### ステージ 1: 正規化（Normalize）

```
入力: articles.raw_content (HTML)
出力: articles.clean_content (プレーンテキスト)
      articles.language
      articles.status = 'processing'
```

**処理内容:**

1. bluemonday で安全な HTML タグのみに制限（スクリプト・スタイル除去）。
2. `golang.org/x/net/html` でテキストノードを抽出。
3. 連続する空白・改行を正規化。
4. 言語検出（`lingua-go` ライブラリ使用）→ `articles.language` に保存。
5. クリーンテキストが 50 文字未満なら `status='skipped'` にして処理終了。

---

### ステージ 2: 埋め込み生成（Embed）

```
入力: articles.clean_content の先頭 8,000 文字
出力: article_embeddings（1536 次元ベクトル）
```

**OpenAI API 呼び出し仕様:**

```
モデル: text-embedding-3-small
エンコーディング: cl100k_base
入力: title + "\n\n" + clean_content の先頭 8000 文字
次元数: 1536（デフォルト）
```

**エラーハンドリング:**
- 429 (Rate Limit): 60 秒待機してリトライ。
- 5xx: 指数バックオフ（1s, 2s, 4s）でリトライ。
- 最大リトライ: 3 回。

**保存:**
```sql
INSERT INTO article_embeddings
    (article_id, embedding, model_name, model_version)
VALUES
    (?, ?, 'text-embedding-3-small', '2024-02-01');
```

---

### ステージ 3: 要約生成（Summarize）

```
入力: articles.title + articles.clean_content の先頭 4,000 文字
出力: article_summaries（要約テキスト）
```

**プロンプト仕様（v1.0）:**

```
System:
あなたは技術記事の要約専門家です。
以下のルールに従って要約を作成してください:
- 2〜4 文で収める
- 記事の主要な技術的ポイントを含める
- 「なぜ重要か」を 1 文で説明する
- 箇条書きは使わない
- 日本語か英語か、記事の言語に合わせる

User:
タイトル: {title}

内容:
{clean_content の先頭 4000 文字}
```

**OpenAI API 呼び出し仕様:**

```
モデル: gpt-4o-mini
temperature: 0.3
max_tokens: 300
```

**保存:**
```sql
INSERT INTO article_summaries
    (article_id, summary_text, model_name, model_version, prompt_version, token_count)
VALUES
    (?, ?, 'gpt-4o-mini', '2024-07-18', 'v1.0', ?);
```

---

### ステージ 4: タグ付け（Tag）

```
入力: articles.title + articles.clean_content の先頭 2,000 文字
出力: article_tags（タグリスト）
```

**プロンプト仕様（v1.0）:**

```
System:
あなたは技術記事の分類専門家です。
以下の記事に対して、最も適切なタグを 3〜7 個 JSON で返してください。
タグは以下のリストから選択してください:
{tags マスターの name 一覧}

必ず以下の JSON 形式のみで返してください（余計なテキスト不可）:
{"tags": [{"name": "go", "confidence": 0.95}, ...]}

User:
タイトル: {title}

内容:
{clean_content の先頭 2000 文字}
```

**OpenAI API 呼び出し仕様:**

```
モデル: gpt-4o-mini
temperature: 0.1
max_tokens: 200
response_format: {"type": "json_object"}
```

**保存:**
1. JSON レスポンスをパース。
2. タグ名が `tags` テーブルに存在しない場合はスキップ（新規タグは作成しない）。
3. confidence が 0.5 以上のタグのみを保存。

```sql
INSERT INTO article_tags (article_id, tag_id, confidence)
VALUES (?, ?, ?)
ON CONFLICT (article_id, tag_id) DO UPDATE SET confidence = EXCLUDED.confidence;
```

---

### ステージ 5: トレンドスコア計算（Score）

```
入力: articles テーブルのメタデータ
出力: articles.trend_score (0.0 〜 1.0)
```

**スコア計算式:**

```
trend_score = 0.7 * time_decay(published_at) + 0.3 * source_quality_score

time_decay(published_at) =
  1.0   : 公開から 6 時間以内
  0.85  : 公開から 12 時間以内
  0.70  : 公開から 24 時間以内
  0.50  : 公開から 3 日以内
  0.30  : 公開から 7 日以内
  0.10  : 公開から 30 日以内
  0.05  : 30 日以上前
```

**Phase 2 以降で追加予定:**
- 同一 URL がソーシャルでシェアされた回数（外部 API）
- 同一トピックの記事が集中して公開されている場合のバースト検出

---

## ステージ失敗時の挙動

```go
type StageError struct {
    Stage   string
    Err     error
    Retryable bool
}
```

| エラー種別 | Retryable | 処理 |
|----------|-----------|------|
| LLM API タイムアウト | true | 指数バックオフでリトライ |
| LLM API 429 | true | 60 秒待機してリトライ |
| LLM API 不正レスポンス（JSON パース失敗） | true（最大2回） | リトライ |
| LLM API 4xx (クォータ超過等) | false | 失敗として記録、アラート発行 |
| DB 接続エラー | true | リトライ |
| コンテンツが短すぎる | false | `status='skipped'` |

---

## 処理後の後処理

全ステージ完了後:

```sql
UPDATE articles SET status='processed', updated_at=NOW()
WHERE id=?;

UPDATE processing_jobs
SET status='completed', completed_at=NOW()
WHERE article_id=?;
```

レコメンドワーカーへの通知:
```
Redis Streams に投入:
Stream: "stream:recommendation_refresh"
Message: {"trigger": "new_articles", "article_id": "uuid"}
```

---

## メトリクス・ロギング

```json
{
  "level": "info",
  "event": "article_processed",
  "article_id": "uuid",
  "source_id": "uuid",
  "stages_completed": ["normalize", "embed", "summarize", "tag", "score"],
  "total_duration_ms": 3200,
  "llm_tokens_used": 450,
  "timestamp": "2024-03-15T09:05:00Z"
}
```

CloudWatch メトリクス:
- `signalix/processing/article_duration_ms` (Milliseconds)
- `signalix/processing/llm_tokens_used` (Count)
- `signalix/processing/stage_failures` (Count, Dimension: stage)
- `signalix/processing/queue_depth` (Count) ← 毎分発行
