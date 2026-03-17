# API 仕様: フィードソース (Sources)

## エンドポイント一覧

| メソッド | パス | 認証必須 | 説明 |
|---------|------|---------|------|
| GET | `/api/v1/sources` | 不要 | ソース一覧取得 |
| GET | `/api/v1/sources/:id` | 不要 | ソース詳細取得 |

---

## ソースオブジェクト

```json
{
  "id": "source-uuid-001",
  "name": "Go Blog",
  "site_url": "https://blog.golang.org",
  "description": "Go 公式チームのブログ",
  "category": "tech",
  "language": "en",
  "quality_score": 0.95,
  "status": "active",
  "last_fetched_at": "2024-03-15T09:00:00Z",
  "article_count": 1250,
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

## GET /api/v1/sources

公開ソースの一覧を取得する。

### クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `category` | string | - | カテゴリでフィルタ |
| `language` | string | - | 言語でフィルタ |
| `page` | integer | 1 | ページ番号 |
| `per_page` | integer | 50 | 1ページあたりの件数 |

`status=active` のソースのみ返す。

### レスポンス: 200 OK

```json
{
  "data": [
    { /* ソースオブジェクト */ }
  ],
  "pagination": { ... }
}
```

---

## GET /api/v1/sources/:id

ソースの詳細情報と最近の記事を取得する。

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|-----|------|
| `id` | string (UUID) | ソース ID |

### レスポンス: 200 OK

```json
{
  "data": {
    "source": { /* ソースオブジェクト */ },
    "recent_articles": [
      { /* ArticleSummary */ },
      { /* ArticleSummary */ }
    ]
  }
}
```

`recent_articles` は最新 5 件を返す。
