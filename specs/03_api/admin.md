# API 仕様: 管理者 (Admin)

すべての管理者エンドポイントは `is_admin: true` の JWT が必要。
一般ユーザーが呼び出した場合は `403 Forbidden` を返す。

## エンドポイント一覧

| メソッド | パス | 説明 |
|---------|------|------|
| GET | `/api/v1/admin/sources` | ソース一覧（管理者用フル情報） |
| POST | `/api/v1/admin/sources` | ソース新規登録 |
| PATCH | `/api/v1/admin/sources/:id` | ソース情報更新 |
| DELETE | `/api/v1/admin/sources/:id` | ソース削除 |
| POST | `/api/v1/admin/sources/:id/fetch` | 手動フェッチ実行 |
| GET | `/api/v1/admin/ingestion-jobs` | インジェスションジョブ一覧 |
| POST | `/api/v1/admin/ingestion-jobs/:id/retry` | ジョブリトライ |
| GET | `/api/v1/admin/processing-jobs` | 記事処理ジョブ一覧 |
| POST | `/api/v1/admin/processing-jobs/:id/retry` | 処理ジョブリトライ |
| GET | `/api/v1/admin/stats` | システム統計 |

---

## POST /api/v1/admin/sources

新しい RSS ソースを登録する。

### リクエストボディ

```json
{
  "name": "Go Blog",
  "feed_url": "https://blog.golang.org/feed.atom",
  "site_url": "https://blog.golang.org",
  "description": "Go 公式チームのブログ",
  "category": "tech",
  "language": "en",
  "fetch_interval_minutes": 60,
  "quality_score": 0.9
}
```

| フィールド | 型 | 必須 | バリデーション |
|-----------|-----|------|------------|
| `name` | string | ✓ | 1〜100 文字 |
| `feed_url` | string | ✓ | 有効な URL。一意 |
| `site_url` | string | ✓ | 有効な URL |
| `description` | string | - | 最大 500 文字 |
| `category` | string | ✓ | 定義済みカテゴリ値 |
| `language` | string | ✓ | ISO 639-1 言語コード |
| `fetch_interval_minutes` | integer | - | 15〜1440（デフォルト: 60） |
| `quality_score` | number | - | 0.0〜1.0（デフォルト: 0.7） |

**カテゴリ定義値:** `tech`, `ai`, `startup`, `infrastructure`, `backend`, `frontend`, `security`, `data`, `other`

### レスポンス: 201 Created

```json
{
  "data": {
    "id": "source-uuid-001",
    "name": "Go Blog",
    "feed_url": "https://blog.golang.org/feed.atom",
    "site_url": "https://blog.golang.org",
    "description": "Go 公式チームのブログ",
    "category": "tech",
    "language": "en",
    "fetch_interval_minutes": 60,
    "quality_score": 0.9,
    "status": "active",
    "last_fetched_at": null,
    "consecutive_failures": 0,
    "created_at": "2024-03-15T10:00:00Z",
    "updated_at": "2024-03-15T10:00:00Z"
  }
}
```

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| feed_url 重複 | 409 | `feed_url_already_exists` |
| バリデーションエラー | 400 | `validation_error` |

---

## PATCH /api/v1/admin/sources/:id

ソース情報を部分更新する。

更新可能フィールド: `name`, `description`, `category`, `fetch_interval_minutes`, `quality_score`, `status`

### リクエストボディ（例）

```json
{
  "status": "paused",
  "quality_score": 0.6
}
```

### レスポンス: 200 OK

更新後のソース情報を返す。

---

## POST /api/v1/admin/sources/:id/fetch

指定ソースの即時フェッチを手動でトリガーする。

### リクエスト

ボディ不要。

### レスポンス: 202 Accepted

```json
{
  "data": {
    "job_id": "job-uuid-001",
    "message": "フェッチジョブをキューに追加しました"
  }
}
```

---

## GET /api/v1/admin/ingestion-jobs

インジェスションジョブの一覧を取得する。

### クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `source_id` | string | - | ソースでフィルタ |
| `status` | string | - | `running`, `completed`, `failed` でフィルタ |
| `page` | integer | 1 | ページ番号 |
| `per_page` | integer | 50 | 1ページあたりの件数 |

### レスポンス: 200 OK

```json
{
  "data": [
    {
      "id": "job-uuid-001",
      "source": {
        "id": "source-uuid-001",
        "name": "Go Blog"
      },
      "status": "completed",
      "articles_found": 10,
      "articles_new": 3,
      "articles_skipped": 7,
      "error_message": null,
      "started_at": "2024-03-15T09:00:00Z",
      "completed_at": "2024-03-15T09:00:15Z"
    }
  ],
  "pagination": { ... }
}
```

---

## GET /api/v1/admin/stats

システム全体の統計情報を取得する。

### レスポンス: 200 OK

```json
{
  "data": {
    "sources": {
      "total": 25,
      "active": 23,
      "degraded": 2,
      "disabled": 0
    },
    "articles": {
      "total": 45820,
      "processed": 45600,
      "pending": 180,
      "failed": 40
    },
    "ingestion_jobs": {
      "last_24h_completed": 480,
      "last_24h_failed": 3
    },
    "processing_jobs": {
      "queue_depth": 12,
      "last_24h_completed": 350,
      "last_24h_failed": 2
    },
    "users": {
      "total": 1200,
      "active_last_7d": 340
    }
  }
}
```
