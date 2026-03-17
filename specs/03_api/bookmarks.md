# API 仕様: ブックマーク (Bookmarks)

## エンドポイント一覧

| メソッド | パス | 認証必須 | 説明 |
|---------|------|---------|------|
| GET | `/api/v1/bookmarks` | 必要 | ブックマーク一覧取得 |
| POST | `/api/v1/bookmarks` | 必要 | ブックマーク追加 |
| DELETE | `/api/v1/bookmarks/:article_id` | 必要 | ブックマーク削除 |

---

## GET /api/v1/bookmarks

ユーザーのブックマーク一覧を保存日時の降順で取得する。

### クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `page` | integer | 1 | ページ番号 |
| `per_page` | integer | 20 | 1ページあたりの件数（最大 100） |

### レスポンス: 200 OK

```json
{
  "data": [
    {
      "bookmark_id": "bookmark-uuid-001",
      "bookmarked_at": "2024-03-15T10:00:00Z",
      "article": {
        "id": "article-uuid-001",
        "title": "Go 1.23 のジェネリクス改善について",
        "url": "https://example.com/go-1.23-generics",
        "source": {
          "id": "source-uuid-001",
          "name": "Go Blog",
          "site_url": "https://blog.golang.org"
        },
        "author": "Go Team",
        "language": "en",
        "published_at": "2024-03-15T08:00:00Z",
        "summary": "Go 1.23 ではジェネリクスの型推論が大幅に改善され...",
        "tags": [
          { "id": "tag-uuid-001", "name": "go", "category": "language" }
        ],
        "trend_score": 0.87
      }
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3,
    "has_next": true,
    "has_prev": false
  }
}
```

---

## POST /api/v1/bookmarks

記事をブックマークに追加する。

### リクエストボディ

```json
{
  "article_id": "article-uuid-001"
}
```

| フィールド | 型 | 必須 | バリデーション |
|-----------|-----|------|------------|
| `article_id` | string (UUID) | ✓ | 存在する記事の ID |

### レスポンス: 201 Created

```json
{
  "data": {
    "bookmark_id": "bookmark-uuid-001",
    "article_id": "article-uuid-001",
    "bookmarked_at": "2024-03-15T10:00:00Z"
  }
}
```

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| 記事が存在しない | 404 | `article_not_found` |
| 既にブックマーク済み | 409 | `already_bookmarked` |

**冪等性:** `Idempotency-Key` ヘッダーを使用することで、ネットワークエラー時の二重登録を防止できる。

---

## DELETE /api/v1/bookmarks/:article_id

ブックマークを削除する。

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|-----|------|
| `article_id` | string (UUID) | 記事 ID |

### レスポンス: 204 No Content

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| ブックマークが存在しない | 404 | `bookmark_not_found` |
