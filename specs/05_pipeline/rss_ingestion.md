# パイプライン仕様: RSS インジェスション

## 概要

RSS インジェスションワーカーは、登録済みのフィードソースから定期的に記事を収集し、
`articles` テーブルへの挿入と処理ジョブのキューイングを担当する。

---

## ワーカー設計

### 実行モード

- **スケジュール実行**: 内部ティッカーで全 `status=active` ソースの `fetch_interval_minutes` を
  チェックし、フェッチ期限を過ぎたソースを順次処理する。
- **手動実行**: 管理者 API `POST /admin/sources/:id/fetch` からもトリガー可能。

### 並行性

- 各ソースのフェッチは Goroutine で並列実行（最大同時実行数: 10）。
- 同一ソースの二重実行を防ぐため、実行中ソースを Redis で排他制御する。
  - キー: `lock:ingestion:{source_id}`
  - TTL: 5 分（フェッチが 5 分で終わらなければロックを自動解放）

---

## フェッチ処理フロー

```
1. 対象ソースの選定
   SELECT * FROM sources
   WHERE status = 'active'
     AND (last_fetched_at IS NULL
          OR last_fetched_at < NOW() - (fetch_interval_minutes * INTERVAL '1 minute'))
   ORDER BY last_fetched_at ASC NULLS FIRST
   LIMIT 20;

2. ソースごとに以下を実行（Goroutine）:
   a. Redis でフェッチロック取得（失敗→スキップ）
   b. INSERT ingestion_jobs (status='running', source_id=?, started_at=NOW())
   c. HTTP GET feed_url
      - タイムアウト: 30 秒
      - User-Agent: "Signalix-Bot/1.0"
      - Gzip デコード対応
   d. RSS/Atom XML パース
   e. エントリーごとに記事処理
   f. sources テーブルの last_fetched_at を更新
   g. ingestion_jobs を complete/failed で更新
   h. Redisロック解放
```

---

## RSS/Atom パース仕様

対応フォーマット:
- RSS 2.0
- Atom 1.0
- RSS 1.0 (RDF)

### フィールドマッピング

| articles カラム | RSS 2.0 | Atom 1.0 |
|----------------|---------|---------|
| `url` | `<link>` | `<link href>` |
| `title` | `<title>` | `<title>` |
| `raw_content` | `<content:encoded>` または `<description>` | `<content>` または `<summary>` |
| `author` | `<author>` または `<dc:creator>` | `<author><name>` |
| `published_at` | `<pubDate>` | `<published>` または `<updated>` |

### コンテンツの取得優先順位

1. `<content:encoded>` 優先（フルテキスト）
2. `<content>` (Atom)
3. `<description>` / `<summary>`（抜粋）
4. いずれもなければ空文字列（`skipped` として扱う可能性あり）

---

## 重複排除

```go
// URL を正規化してハッシュ化
func normalizeURL(rawURL string) string {
    u, _ := url.Parse(rawURL)
    // フラグメントを除去
    u.Fragment = ""
    // クエリパラメータをソート（utm_* 等は除去）
    q := u.Query()
    for key := range q {
        if strings.HasPrefix(key, "utm_") {
            q.Del(key)
        }
    }
    u.RawQuery = q.Encode()
    return u.String()
}

func articleURLHash(normalizedURL string) string {
    h := sha256.Sum256([]byte(normalizedURL))
    return hex.EncodeToString(h[:])
}
```

- `url_hash` の UNIQUE 制約違反は `ON CONFLICT DO NOTHING` で無視する。
- スキップされた記事数は `ingestion_jobs.articles_skipped` に記録する。

---

## コンテンツが薄い記事の除外

以下の条件に該当する場合、`status='skipped'` として保存し処理対象から除外する:

- `raw_content` が 100 文字未満
- タイトルが空文字列

---

## エラーハンドリング

| エラー種別 | 処理 |
|----------|------|
| HTTP タイムアウト（30 秒） | `ingestion_jobs` を `failed` に更新。`sources.consecutive_failures` をインクリメント |
| HTTP 4xx (401, 403, 404) | 同上。ソースの設定確認が必要なため、`consecutive_failures >= 3` で `status='degraded'` |
| HTTP 5xx | 同上（一時的エラーとして扱う） |
| XML パースエラー | 同上 |
| DB エラー（記事挿入） | ロールバック。ジョブ失敗として記録 |

### ソースの自動ステータス変更

```
consecutive_failures >= 3  → status='degraded'
consecutive_failures >= 10 → status='disabled'
フェッチ成功時              → consecutive_failures=0, status='active'（degraded の場合）
```

---

## 処理後のジョブ投入

新規に挿入された記事は、即座に処理ジョブキューに追加する。

```go
// Redis Streams にジョブを投入
// Stream: "stream:article_processing"
// Message: {"article_id": "uuid", "priority": "normal"}
```

---

## メトリクス・ロギング

各ジョブ完了時にログ出力:

```json
{
  "level": "info",
  "event": "ingestion_job_completed",
  "source_id": "uuid",
  "source_name": "Go Blog",
  "articles_found": 10,
  "articles_new": 3,
  "articles_skipped": 7,
  "duration_ms": 1250,
  "timestamp": "2024-03-15T09:00:15Z"
}
```

CloudWatch メトリクスとして発行:
- `signalix/ingestion/articles_new` (Count)
- `signalix/ingestion/job_duration_ms` (Milliseconds)
- `signalix/ingestion/job_failures` (Count)
